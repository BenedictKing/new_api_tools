package cache

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	_ "modernc.org/sqlite"
)

// ThreeTierManager implements three-tier caching architecture
// L1: Redis (hot cache, optional)
// L2: SQLite (local persistent cache, required)
// L3: PostgreSQL/MySQL (read-only data source)
type ThreeTierManager struct {
	mu             sync.RWMutex
	redisAvailable bool
	localDB        *sql.DB
}

var (
	threeTierMgr     *ThreeTierManager
	threeTierMgrOnce sync.Once
)

// GetThreeTierManager returns singleton instance of three-tier cache manager
func GetThreeTierManager() *ThreeTierManager {
	threeTierMgrOnce.Do(func() {
		// Initialize cache SQLite database
		cacheDBPath := filepath.Join(os.Getenv("DATA_DIR"), "cache.db")
		if cacheDBPath == "cache.db" {
			cacheDBPath = "./data/cache.db"
		}

		// Create directory if not exists
		if err := os.MkdirAll(filepath.Dir(cacheDBPath), 0755); err != nil {
			log.Error().Err(err).Msg("failed to create cache db directory")
		}

		// Open SQLite database
		db, err := sql.Open("sqlite", cacheDBPath)
		if err != nil {
			log.Error().Err(err).Msg("failed to open cache database")
		} else {
			db.SetMaxOpenConns(1)
		}

		threeTierMgr = &ThreeTierManager{
			redisAvailable: Available(),
			localDB:        db,
		}
		threeTierMgr.initSQLiteTables()
	})
	return threeTierMgr
}

// initSQLiteTables initializes SQLite cache tables
func (m *ThreeTierManager) initSQLiteTables() {
	if m.localDB == nil {
		log.Warn().Msg("local database not initialized, SQLite cache unavailable")
		return
	}

	tables := []string{
		// Leaderboard cache
		`CREATE TABLE IF NOT EXISTS leaderboard_cache (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			window TEXT NOT NULL,
			sort_by TEXT NOT NULL,
			data TEXT NOT NULL,
			snapshot_time INTEGER NOT NULL,
			expires_at INTEGER NOT NULL,
			UNIQUE(window, sort_by)
		)`,

		// IP monitoring cache
		`CREATE TABLE IF NOT EXISTS ip_monitoring_cache (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			window TEXT NOT NULL,
			data TEXT NOT NULL,
			snapshot_time INTEGER NOT NULL,
			expires_at INTEGER NOT NULL,
			UNIQUE(type, window)
		)`,

		// Generic cache
		`CREATE TABLE IF NOT EXISTS generic_cache (
			key TEXT PRIMARY KEY,
			data TEXT NOT NULL,
			snapshot_time INTEGER NOT NULL,
			expires_at INTEGER NOT NULL
		)`,

		// Slot cache for incremental caching
		`CREATE TABLE IF NOT EXISTS slot_cache (
			slot_key TEXT PRIMARY KEY,
			window TEXT NOT NULL,
			sort_by TEXT NOT NULL,
			slot_start INTEGER NOT NULL,
			slot_end INTEGER NOT NULL,
			data TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			expires_at INTEGER NOT NULL
		)`,
	}

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_leaderboard_expires ON leaderboard_cache(expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_ip_monitoring_expires ON ip_monitoring_cache(expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_generic_expires ON generic_cache(expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_slot_cache_expires ON slot_cache(expires_at)",
	}

	for _, ddl := range tables {
		if _, err := m.localDB.Exec(ddl); err != nil {
			log.Error().Err(err).Msg("failed to create cache table")
		}
	}

	for _, ddl := range indexes {
		if _, err := m.localDB.Exec(ddl); err != nil {
			log.Error().Err(err).Msg("failed to create cache index")
		}
	}

	log.Info().Msg("SQLite cache tables initialized")
}

// Get retrieves from cache (L1 Redis -> L2 SQLite)
func (m *ThreeTierManager) Get(key string, dest interface{}) error {
	// L1: Try Redis first
	if m.redisAvailable {
		found, err := Get().GetJSON(key, dest)
		if err != nil {
			log.Debug().Err(err).Str("key", key).Msg("Redis get failed, trying SQLite")
		} else if found {
			return nil
		}
	}

	// L2: Fallback to SQLite
	return m.getFromSQLite(key, dest)
}

// Set writes to cache (both L1 and L2)
func (m *ThreeTierManager) Set(key string, value interface{}, ttl time.Duration) error {
	// L1: Write to Redis
	if m.redisAvailable {
		if err := Get().Set(key, value, ttl); err != nil {
			log.Debug().Err(err).Str("key", key).Msg("Redis set failed")
		}
	}

	// L2: Write to SQLite
	return m.setToSQLite(key, value, ttl)
}

// Delete removes from cache
func (m *ThreeTierManager) Delete(key string) error {
	// L1: Delete from Redis
	if m.redisAvailable {
		_ = Get().Delete(key)
	}

	// L2: Delete from SQLite
	return m.deleteFromSQLite(key)
}

// getFromSQLite retrieves from SQLite cache
func (m *ThreeTierManager) getFromSQLite(key string, dest interface{}) error {
	if m.localDB == nil {
		return ErrCacheMiss
	}

	var data string
	var expiresAt int64

	err := m.localDB.QueryRow(`
		SELECT data, expires_at FROM generic_cache
		WHERE key = ? AND expires_at > ?
	`, key, time.Now().Unix()).Scan(&data, &expiresAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrCacheMiss
		}
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

// setToSQLite writes to SQLite cache
func (m *ThreeTierManager) setToSQLite(key string, value interface{}, ttl time.Duration) error {
	if m.localDB == nil {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	expiresAt := now + int64(ttl.Seconds())

	_, err = m.localDB.Exec(`
		INSERT INTO generic_cache (key, data, snapshot_time, expires_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			data = excluded.data,
			snapshot_time = excluded.snapshot_time,
			expires_at = excluded.expires_at
	`, key, string(data), now, expiresAt)

	return err
}

// deleteFromSQLite deletes from SQLite cache
func (m *ThreeTierManager) deleteFromSQLite(key string) error {
	if m.localDB == nil {
		return nil
	}

	_, err := m.localDB.Exec("DELETE FROM generic_cache WHERE key = ?", key)
	return err
}

// SetLeaderboard caches leaderboard data
func (m *ThreeTierManager) SetLeaderboard(window, sortBy string, data interface{}, ttl time.Duration) error {
	key := CacheKey("leaderboard", window, sortBy)

	// L1: Redis
	if m.redisAvailable {
		_ = Get().Set(key, data, ttl)
	}

	// L2: SQLite
	if m.localDB == nil {
		return nil
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	expiresAt := now + int64(ttl.Seconds())

	_, err = m.localDB.Exec(`
		INSERT INTO leaderboard_cache (window, sort_by, data, snapshot_time, expires_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(window, sort_by) DO UPDATE SET
			data = excluded.data,
			snapshot_time = excluded.snapshot_time,
			expires_at = excluded.expires_at
	`, window, sortBy, string(jsonData), now, expiresAt)

	return err
}

// GetLeaderboard retrieves leaderboard from cache
func (m *ThreeTierManager) GetLeaderboard(window, sortBy string, dest interface{}) error {
	key := CacheKey("leaderboard", window, sortBy)

	// L1: Redis
	if m.redisAvailable {
		found, err := Get().GetJSON(key, dest)
		if err == nil && found {
			return nil
		}
	}

	// L2: SQLite
	if m.localDB == nil {
		return ErrCacheMiss
	}

	var data string
	err := m.localDB.QueryRow(`
		SELECT data FROM leaderboard_cache
		WHERE window = ? AND sort_by = ? AND expires_at > ?
	`, window, sortBy, time.Now().Unix()).Scan(&data)

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrCacheMiss
		}
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

// SetIPMonitoring caches IP monitoring data
func (m *ThreeTierManager) SetIPMonitoring(monitorType, window string, data interface{}, ttl time.Duration) error {
	key := CacheKey("ip_monitoring", monitorType, window)

	// L1: Redis
	if m.redisAvailable {
		_ = Get().Set(key, data, ttl)
	}

	// L2: SQLite
	if m.localDB == nil {
		return nil
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	expiresAt := now + int64(ttl.Seconds())

	_, err = m.localDB.Exec(`
		INSERT INTO ip_monitoring_cache (type, window, data, snapshot_time, expires_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(type, window) DO UPDATE SET
			data = excluded.data,
			snapshot_time = excluded.snapshot_time,
			expires_at = excluded.expires_at
	`, monitorType, window, string(jsonData), now, expiresAt)

	return err
}

// GetIPMonitoring retrieves IP monitoring data from cache
func (m *ThreeTierManager) GetIPMonitoring(monitorType, window string, dest interface{}) error {
	key := CacheKey("ip_monitoring", monitorType, window)

	// L1: Redis
	if m.redisAvailable {
		found, err := Get().GetJSON(key, dest)
		if err == nil && found {
			return nil
		}
	}

	// L2: SQLite
	if m.localDB == nil {
		return ErrCacheMiss
	}

	var data string
	err := m.localDB.QueryRow(`
		SELECT data FROM ip_monitoring_cache
		WHERE type = ? AND window = ? AND expires_at > ?
	`, monitorType, window, time.Now().Unix()).Scan(&data)

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrCacheMiss
		}
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

// CleanupExpired removes expired cache entries
func (m *ThreeTierManager) CleanupExpired() (int64, error) {
	if m.localDB == nil {
		return 0, nil
	}

	now := time.Now().Unix()
	var total int64

	tables := []string{"generic_cache", "leaderboard_cache", "ip_monitoring_cache", "slot_cache"}
	for _, table := range tables {
		result, err := m.localDB.Exec(fmt.Sprintf("DELETE FROM %s WHERE expires_at < ?", table), now)
		if err != nil {
			log.Error().Err(err).Str("table", table).Msg("failed to cleanup expired cache")
			continue
		}
		count, _ := result.RowsAffected()
		total += count
	}

	if total > 0 {
		log.Info().Int64("count", total).Msg("cleaned up expired cache entries")
	}

	return total, nil
}

// RestoreToRedis restores cache from SQLite to Redis
func (m *ThreeTierManager) RestoreToRedis() (int, error) {
	if !m.redisAvailable || m.localDB == nil {
		return 0, nil
	}

	now := time.Now().Unix()
	restored := 0

	rows, err := m.localDB.Query("SELECT key, data, expires_at FROM generic_cache WHERE expires_at > ?", now)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var key, data string
		var expiresAt int64
		if err := rows.Scan(&key, &data, &expiresAt); err != nil {
			continue
		}

		ttl := time.Duration(expiresAt-now) * time.Second
		if ttl > 0 {
			var value interface{}
			if json.Unmarshal([]byte(data), &value) == nil {
				if Get().Set(key, value, ttl) == nil {
					restored++
				}
			}
		}
	}

	log.Info().Int("count", restored).Msg("restored cache from SQLite to Redis")
	return restored, nil
}

// GetStats returns cache statistics
func (m *ThreeTierManager) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"redis_available": m.redisAvailable,
	}

	// Redis stats
	if m.redisAvailable {
		if info, err := GetInfo(); err == nil {
			stats["redis_keys"] = info.KeyCount
			stats["redis_memory"] = info.MemoryUsed
			stats["redis_hit_rate"] = info.HitRate
		}
	}

	// SQLite stats
	if m.localDB != nil {
		now := time.Now().Unix()
		tables := map[string]string{
			"generic":        "generic_cache",
			"leaderboard":    "leaderboard_cache",
			"ip_monitoring":  "ip_monitoring_cache",
			"slot":           "slot_cache",
		}

		var total int64
		for key, table := range tables {
			var count int64
			_ = m.localDB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE expires_at > ?", table), now).Scan(&count)
			stats["sqlite_"+key] = count
			total += count
		}
		stats["sqlite_total"] = total
	}

	return stats
}

// IsRedisAvailable checks if Redis is available
func (m *ThreeTierManager) IsRedisAvailable() bool {
	return m.redisAvailable
}

// RefreshRedisStatus refreshes Redis availability status
func (m *ThreeTierManager) RefreshRedisStatus() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.redisAvailable = Available()
}
