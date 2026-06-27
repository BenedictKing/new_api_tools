package cache

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// SlotConfig defines time slot configuration for incremental caching
type SlotConfig struct {
	SlotSize  int64 // Slot size in seconds
	SlotCount int   // Number of slots
	TTL       int64 // Cache expiration time in seconds
}

// Slot configurations: only 3d, 7d, 14d use incremental caching
var slotConfigs = map[string]SlotConfig{
	"3d": {
		SlotSize:  6 * 3600,      // 6 hours per slot
		SlotCount: 12,            // 12 slots
		TTL:       7 * 24 * 3600, // 7 days expiration
	},
	"7d": {
		SlotSize:  12 * 3600,      // 12 hours per slot
		SlotCount: 14,             // 14 slots
		TTL:       14 * 24 * 3600, // 14 days expiration
	},
	"14d": {
		SlotSize:  24 * 3600,      // 24 hours per slot
		SlotCount: 14,             // 14 slots
		TTL:       21 * 24 * 3600, // 21 days expiration
	},
}

// IncrementalPeriods lists periods that support incremental caching
var IncrementalPeriods = map[string]bool{
	"3d":  true,
	"7d":  true,
	"14d": true,
}

// SlotData represents cached slot data
type SlotData struct {
	SlotKey   string      `json:"slot_key"`
	Window    string      `json:"window"`
	SortBy    string      `json:"sort_by"`
	SlotStart int64       `json:"slot_start"`
	SlotEnd   int64       `json:"slot_end"`
	Data      interface{} `json:"data"`
	CreatedAt int64       `json:"created_at"`
	ExpiresAt int64       `json:"expires_at"`
}

// SlotInfo represents slot time range information
type SlotInfo struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
	Index int   `json:"index"`
}

// IsIncrementalWindow checks if window supports incremental caching
func IsIncrementalWindow(window string) bool {
	return IncrementalPeriods[window]
}

// GetSlotConfig returns slot configuration for window
func GetSlotConfig(window string) *SlotConfig {
	if cfg, ok := slotConfigs[window]; ok {
		return &cfg
	}
	return nil
}

// CalculateSlots calculates slot list for time window
func CalculateSlots(window string) []SlotInfo {
	cfg := GetSlotConfig(window)
	if cfg == nil {
		return nil
	}

	now := time.Now().Unix()
	slots := make([]SlotInfo, cfg.SlotCount)

	// Calculate current slot start (round down to slot boundary)
	currentSlotStart := (now / cfg.SlotSize) * cfg.SlotSize

	for i := 0; i < cfg.SlotCount; i++ {
		slotStart := currentSlotStart - int64(i)*cfg.SlotSize
		slotEnd := slotStart + cfg.SlotSize
		slots[cfg.SlotCount-1-i] = SlotInfo{
			Start: slotStart,
			End:   slotEnd,
			Index: cfg.SlotCount - 1 - i,
		}
	}

	return slots
}

// SetSlotCache sets slot cache data
func (m *ThreeTierManager) SetSlotCache(window, sortBy string, slotStart, slotEnd int64, data interface{}) error {
	cfg := GetSlotConfig(window)
	if cfg == nil {
		return fmt.Errorf("unsupported incremental cache window: %s", window)
	}

	slotKey := fmt.Sprintf("%s:%s:%d", window, sortBy, slotStart)
	now := time.Now().Unix()
	expiresAt := now + cfg.TTL

	// L1: Redis
	if m.redisAvailable {
		redisKey := CacheKey("slot", slotKey)
		slotData := &SlotData{
			SlotKey:   slotKey,
			Window:    window,
			SortBy:    sortBy,
			SlotStart: slotStart,
			SlotEnd:   slotEnd,
			Data:      data,
			CreatedAt: now,
			ExpiresAt: expiresAt,
		}
		_ = Get().Set(redisKey, slotData, time.Duration(cfg.TTL)*time.Second)
	}

	// L2: SQLite
	if m.localDB == nil {
		return nil
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = m.localDB.Exec(`
		INSERT INTO slot_cache (slot_key, window, sort_by, slot_start, slot_end, data, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(slot_key) DO UPDATE SET
			data = excluded.data,
			created_at = excluded.created_at,
			expires_at = excluded.expires_at
	`, slotKey, window, sortBy, slotStart, slotEnd, string(jsonData), now, expiresAt)

	return err
}

// GetSlotCache retrieves slot cache data
func (m *ThreeTierManager) GetSlotCache(window, sortBy string, slotStart int64, dest interface{}) error {
	slotKey := fmt.Sprintf("%s:%s:%d", window, sortBy, slotStart)

	// L1: Redis
	if m.redisAvailable {
		redisKey := CacheKey("slot", slotKey)
		var slotData SlotData
		found, err := Get().GetJSON(redisKey, &slotData)
		if err == nil && found {
			// Copy Data field to dest
			dataBytes, _ := json.Marshal(slotData.Data)
			return json.Unmarshal(dataBytes, dest)
		}
	}

	// L2: SQLite
	if m.localDB == nil {
		return ErrCacheMiss
	}

	var data string
	err := m.localDB.QueryRow(`
		SELECT data FROM slot_cache
		WHERE slot_key = ? AND expires_at > ?
	`, slotKey, time.Now().Unix()).Scan(&data)

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrCacheMiss
		}
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

// GetMissingSlots returns list of slots not in cache
func (m *ThreeTierManager) GetMissingSlots(window, sortBy string) []SlotInfo {
	slots := CalculateSlots(window)
	if slots == nil {
		return nil
	}

	missing := make([]SlotInfo, 0)
	for _, slot := range slots {
		var dummy interface{}
		if err := m.GetSlotCache(window, sortBy, slot.Start, &dummy); err != nil {
			missing = append(missing, slot)
		}
	}

	return missing
}

// GetCachedSlots returns list of slots in cache
func (m *ThreeTierManager) GetCachedSlots(window, sortBy string) []SlotInfo {
	slots := CalculateSlots(window)
	if slots == nil {
		return nil
	}

	cached := make([]SlotInfo, 0)
	for _, slot := range slots {
		var dummy interface{}
		if err := m.GetSlotCache(window, sortBy, slot.Start, &dummy); err == nil {
			cached = append(cached, slot)
		}
	}

	return cached
}

// AggregateSlotData aggregates slot data
// aggregator function is used to merge data from multiple slots
func (m *ThreeTierManager) AggregateSlotData(window, sortBy string, aggregator func([]interface{}) interface{}) (interface{}, error) {
	slots := CalculateSlots(window)
	if slots == nil {
		return nil, fmt.Errorf("unsupported incremental cache window: %s", window)
	}

	allData := make([]interface{}, 0, len(slots))
	for _, slot := range slots {
		var data interface{}
		if err := m.GetSlotCache(window, sortBy, slot.Start, &data); err == nil {
			allData = append(allData, data)
		}
	}

	if len(allData) == 0 {
		return nil, ErrCacheMiss
	}

	return aggregator(allData), nil
}

// ClearSlotCache clears slot cache for window and sort type
func (m *ThreeTierManager) ClearSlotCache(window, sortBy string) error {
	// L1: Redis
	if m.redisAvailable {
		pattern := CacheKey("slot", fmt.Sprintf("%s:%s:*", window, sortBy))
		_, _ = DeletePattern(pattern)
	}

	// L2: SQLite
	if m.localDB == nil {
		return nil
	}

	_, err := m.localDB.Exec("DELETE FROM slot_cache WHERE window = ? AND sort_by = ?", window, sortBy)
	return err
}

// GetSlotCacheStats returns slot cache statistics
func (m *ThreeTierManager) GetSlotCacheStats() map[string]interface{} {
	stats := make(map[string]interface{})

	for window := range slotConfigs {
		windowStats := make(map[string]interface{})
		slots := CalculateSlots(window)

		// Statistics for various sort types
		sortTypes := []string{"requests", "quota", "failure_rate"}
		for _, sortBy := range sortTypes {
			cached := m.GetCachedSlots(window, sortBy)
			windowStats[sortBy] = map[string]interface{}{
				"total":  len(slots),
				"cached": len(cached),
				"rate":   float64(len(cached)) / float64(len(slots)) * 100,
			}
		}

		stats[window] = windowStats
	}

	return stats
}

// WarmupSlotCache pre-warms slot cache
// fetchSlotData function is used to fetch data for a single slot
func (m *ThreeTierManager) WarmupSlotCache(window, sortBy string, fetchSlotData func(start, end int64) (interface{}, error)) error {
	missing := m.GetMissingSlots(window, sortBy)
	if len(missing) == 0 {
		log.Debug().
			Str("window", window).
			Str("sort_by", sortBy).
			Msg("slot cache complete, no warmup needed")
		return nil
	}

	log.Info().
		Str("window", window).
		Str("sort_by", sortBy).
		Int("missing", len(missing)).
		Msg("starting slot cache warmup")

	for _, slot := range missing {
		data, err := fetchSlotData(slot.Start, slot.End)
		if err != nil {
			log.Warn().
				Err(err).
				Str("window", window).
				Int64("slot_start", slot.Start).
				Msg("failed to fetch slot data")
			continue
		}

		if err := m.SetSlotCache(window, sortBy, slot.Start, slot.End, data); err != nil {
			log.Warn().
				Err(err).
				Str("window", window).
				Int64("slot_start", slot.Start).
				Msg("failed to set slot cache")
		}
	}

	log.Info().
		Str("window", window).
		Str("sort_by", sortBy).
		Msg("slot cache warmup completed")

	return nil
}
