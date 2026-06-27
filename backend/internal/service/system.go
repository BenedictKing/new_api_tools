package service

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/new-api-tools/backend/internal/database"
)

// ScaleLevel describes detected deployment scale.
type ScaleLevel string

const (
	ScaleSmall  ScaleLevel = "small"
	ScaleMedium ScaleLevel = "medium"
	ScaleLarge  ScaleLevel = "large"
	ScaleXLarge ScaleLevel = "xlarge"
)

// ScaleMetrics contains key scale indicators.
type ScaleMetrics struct {
	TotalUsers     int64   `json:"total_users"`
	ActiveUsers24h int64   `json:"active_users_24h"`
	Logs24h        int64   `json:"logs_24h"`
	TotalLogs      int64   `json:"total_logs"`
	RPMAvg         float64 `json:"rpm_avg"`
}

// ScaleSettings contains recommended frontend/cache behavior.
type ScaleSettings struct {
	Scale                   ScaleLevel `json:"scale"`
	CacheTTL                int        `json:"cache_ttl"`
	RefreshInterval         int        `json:"refresh_interval"`
	FrontendRefreshInterval int        `json:"frontend_refresh_interval"`
	Description             string     `json:"description"`
}

// ScaleDetectionResult is returned by /api/system/scale.
type ScaleDetectionResult struct {
	Scale    ScaleLevel    `json:"scale"`
	Metrics  ScaleMetrics  `json:"metrics"`
	Settings ScaleSettings `json:"settings"`
	Cached   bool          `json:"cached"`
}

// RefreshEstimate estimates dashboard refresh cost.
type RefreshEstimate struct {
	ShowEstimate           bool       `json:"show_estimate"`
	Scale                  ScaleLevel `json:"scale"`
	EstimatedLogs          int64      `json:"estimated_logs"`
	EstimatedLogsFormatted string     `json:"estimated_logs_formatted"`
	EstimatedSeconds       int        `json:"estimated_seconds"`
	EstimatedTimeFormatted string     `json:"estimated_time_formatted"`
	Warning                string     `json:"warning"`
	LogsPerSecond          int64      `json:"logs_per_second"`
	QueryComplexity        string     `json:"query_complexity"`
}

// SystemService detects system scale and query estimates.
type SystemService struct {
	db    *database.Manager
	logDB *database.Manager
}

var (
	systemSvcMu       sync.RWMutex
	cachedScaleResult *ScaleDetectionResult
	cachedScaleAt     time.Time
)

// NewSystemService creates a system service.
func NewSystemService() *SystemService {
	return &SystemService{db: database.Get(), logDB: database.GetLog()}
}

// DetectScale detects current system scale.
func (s *SystemService) DetectScale(forceRefresh bool) (*ScaleDetectionResult, error) {
	if !forceRefresh {
		systemSvcMu.RLock()
		if cachedScaleResult != nil && time.Since(cachedScaleAt) < 5*time.Minute {
			result := *cachedScaleResult
			result.Cached = true
			systemSvcMu.RUnlock()
			return &result, nil
		}
		systemSvcMu.RUnlock()
	}

	metrics := s.collectMetrics()
	scale := determineScale(metrics)
	result := &ScaleDetectionResult{
		Scale:    scale,
		Metrics:  metrics,
		Settings: scaleSettings(scale),
		Cached:   false,
	}

	systemSvcMu.Lock()
	cachedScaleResult = result
	cachedScaleAt = time.Now()
	systemSvcMu.Unlock()
	return result, nil
}

// GetRefreshEstimate estimates dashboard refresh duration for a period.
func (s *SystemService) GetRefreshEstimate(period string) (*RefreshEstimate, error) {
	detected, err := s.DetectScale(false)
	if err != nil {
		return nil, err
	}
	ratio := periodRatio(period)
	estimatedLogs := int64(float64(detected.Metrics.Logs24h) * ratio)
	if estimatedLogs == 0 && detected.Metrics.TotalLogs > 0 {
		estimatedLogs = int64(float64(detected.Metrics.TotalLogs) * ratio / 30)
	}
	logsPerSecond := int64(100000)
	factor := 1.0
	complexity := "normal"
	switch detected.Scale {
	case ScaleLarge:
		factor = 1.2
		complexity = "high"
	case ScaleXLarge:
		factor = 1.5
		complexity = "very_high"
	}
	seconds := int(math.Ceil(float64(estimatedLogs) / float64(logsPerSecond) * factor))
	if seconds < 5 {
		seconds = 5
	}
	if seconds > 300 {
		seconds = 300
	}
	show := detected.Scale == ScaleLarge || detected.Scale == ScaleXLarge || estimatedLogs >= 1000000
	warning := ""
	if show {
		warning = "当前系统数据量较大，刷新可能需要较长时间"
	}
	return &RefreshEstimate{
		ShowEstimate:           show,
		Scale:                  detected.Scale,
		EstimatedLogs:          estimatedLogs,
		EstimatedLogsFormatted: formatCompactNumber(estimatedLogs),
		EstimatedSeconds:       seconds,
		EstimatedTimeFormatted: formatSeconds(seconds),
		Warning:                warning,
		LogsPerSecond:          logsPerSecond,
		QueryComplexity:        complexity,
	}, nil
}

func (s *SystemService) collectMetrics() ScaleMetrics {
	metrics := ScaleMetrics{}
	metrics.TotalUsers = s.countUsers()
	metrics.ActiveUsers24h = s.countDistinctLogs("user_id", 24*time.Hour)
	metrics.Logs24h = s.countLogsSince(24 * time.Hour)
	logs1h := s.countLogsSince(time.Hour)
	metrics.RPMAvg = math.Round(float64(logs1h)/60*100) / 100
	metrics.TotalLogs = s.estimateTotalLogs()
	return metrics
}

func (s *SystemService) countUsers() int64 {
	query := "SELECT COUNT(*) AS count FROM users"
	if s.db.ColumnExists("users", "deleted_at") {
		query += " WHERE deleted_at IS NULL"
	}
	row, err := s.db.QueryOneWithTimeout(5*time.Second, query)
	if err != nil {
		return 0
	}
	return mapValueToInt64(row, "count")
}

func (s *SystemService) countLogsSince(window time.Duration) int64 {
	start := time.Now().Add(-window).Unix()
	query := s.logDB.RebindQuery("SELECT COUNT(*) AS count FROM logs WHERE created_at >= ?")
	row, err := s.logDB.QueryOneWithTimeout(8*time.Second, query, start)
	if err != nil {
		return 0
	}
	return mapValueToInt64(row, "count")
}

func (s *SystemService) countDistinctLogs(column string, window time.Duration) int64 {
	start := time.Now().Add(-window).Unix()
	query := s.logDB.RebindQuery(fmt.Sprintf("SELECT COUNT(DISTINCT %s) AS count FROM logs WHERE created_at >= ? AND %s IS NOT NULL", column, column))
	row, err := s.logDB.QueryOneWithTimeout(8*time.Second, query, start)
	if err != nil {
		return 0
	}
	return mapValueToInt64(row, "count")
}

func (s *SystemService) estimateTotalLogs() int64 {
	var query string
	if s.logDB.IsPG {
		query = "SELECT COALESCE(reltuples::bigint, 0) AS count FROM pg_class WHERE relname = 'logs'"
	} else {
		query = "SELECT COALESCE(TABLE_ROWS, 0) AS count FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'logs'"
	}
	row, err := s.logDB.QueryOneWithTimeout(5*time.Second, query)
	if err == nil {
		if count := mapValueToInt64(row, "count"); count > 0 {
			return count
		}
	}
	row, err = s.logDB.QueryOneWithTimeout(15*time.Second, "SELECT COUNT(*) AS count FROM logs")
	if err != nil {
		return 0
	}
	return mapValueToInt64(row, "count")
}

func determineScale(metrics ScaleMetrics) ScaleLevel {
	switch {
	case metrics.TotalUsers > 50000 || metrics.Logs24h > 10000000 || metrics.RPMAvg > 7000:
		return ScaleXLarge
	case metrics.TotalUsers > 10000 || metrics.Logs24h > 1000000 || metrics.RPMAvg > 700:
		return ScaleLarge
	case metrics.TotalUsers > 1000 || metrics.Logs24h > 100000 || metrics.RPMAvg > 70:
		return ScaleMedium
	default:
		return ScaleSmall
	}
}

func scaleSettings(scale ScaleLevel) ScaleSettings {
	switch scale {
	case ScaleXLarge:
		return ScaleSettings{Scale: scale, CacheTTL: 900, RefreshInterval: 900, FrontendRefreshInterval: 300, Description: "超大型系统"}
	case ScaleLarge:
		return ScaleSettings{Scale: scale, CacheTTL: 600, RefreshInterval: 600, FrontendRefreshInterval: 180, Description: "大型系统"}
	case ScaleMedium:
		return ScaleSettings{Scale: scale, CacheTTL: 300, RefreshInterval: 300, FrontendRefreshInterval: 60, Description: "中型系统"}
	default:
		return ScaleSettings{Scale: scale, CacheTTL: 120, RefreshInterval: 120, FrontendRefreshInterval: 30, Description: "小型系统"}
	}
}

func periodRatio(period string) float64 {
	switch strings.ToLower(period) {
	case "1h":
		return 1.0 / 24.0
	case "6h":
		return 0.25
	case "12h":
		return 0.5
	case "3d":
		return 3
	case "7d":
		return 7
	case "14d":
		return 14
	case "30d":
		return 30
	default:
		return 1
	}
}

func formatCompactNumber(n int64) string {
	if n >= 100000000 {
		return fmt.Sprintf("%.1f亿", float64(n)/100000000)
	}
	if n >= 10000 {
		return fmt.Sprintf("%.1f万", float64(n)/10000)
	}
	return strconv.FormatInt(n, 10)
}

func formatSeconds(seconds int) string {
	if seconds >= 60 {
		return fmt.Sprintf("约 %d 分 %d 秒", seconds/60, seconds%60)
	}
	return fmt.Sprintf("约 %d 秒", seconds)
}

func mapValueToInt64(row map[string]interface{}, key string) int64 {
	if row == nil {
		return 0
	}
	value, ok := row[key]
	if !ok {
		return 0
	}
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case uint64:
		return int64(v)
	case float64:
		return int64(v)
	case []byte:
		n, _ := strconv.ParseInt(string(v), 10, 64)
		return n
	case string:
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	default:
		return 0
	}
}
