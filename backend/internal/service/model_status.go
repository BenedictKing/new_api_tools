package service

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/new-api-tools/backend/internal/cache"
	"github.com/new-api-tools/backend/internal/database"
)

// Constants for model status
var (
	AvailableTimeWindows = []string{"1h", "6h", "12h", "24h"}
	DefaultTimeWindow    = "24h"
	AvailableThemes      = []string{
		"daylight", "obsidian", "minimal", "neon", "forest", "ocean", "terminal",
		"cupertino", "material", "openai", "anthropic", "vercel", "linear",
		"stripe", "github", "discord", "tesla",
	}
	DefaultTheme = "daylight"
	// LegacyThemeMap maps old theme names to valid ones
	LegacyThemeMap = map[string]string{
		"light":  "daylight",
		"dark":   "obsidian",
		"system": "daylight",
	}
	AvailableRefreshIntervals = []int{0, 30, 60, 120, 300}
	AvailableSortModes        = []string{"default", "availability", "custom"}
)

// Time window slot configurations: {totalSeconds, numSlots, slotSeconds}
// Must match Python backend and frontend TIME_WINDOWS exactly
type timeWindowConfig struct {
	totalSeconds int64
	numSlots     int
	slotSeconds  int64
}

type modelStatusSlotInfo struct {
	total   int64
	success int64
	failure int64
	empty   int64
}

var timeWindowConfigs = map[string]timeWindowConfig{
	"1h":  {3600, 60, 60},    // 1 hour, 60 slots, 1 minute each
	"6h":  {21600, 24, 900},  // 6 hours, 24 slots, 15 minutes each
	"12h": {43200, 24, 1800}, // 12 hours, 24 slots, 30 minutes each
	"24h": {86400, 24, 3600}, // 24 hours, 24 slots, 1 hour each
}

// getStatusColor determines status color based on success rate (matches Python backend)
func getStatusColor(successRate float64, totalRequests int64) string {
	if totalRequests == 0 {
		return "green" // No requests = no issues
	}
	if successRate >= 95 {
		return "green"
	} else if successRate >= 80 {
		return "yellow"
	}
	return "red"
}

// roundRate rounds a float to 2 decimal places
func roundRate(rate float64) float64 {
	return math.Round(rate*100) / 100
}

func normalizeModelStatusGroup(group string) string {
	group = strings.TrimSpace(group)
	if group == "" || strings.EqualFold(group, "all") {
		return "all"
	}
	if strings.EqualFold(group, "default") {
		return "default"
	}
	return group
}

func modelStatusCachePart(value string) string {
	if value == "" {
		return "empty"
	}
	return url.QueryEscape(value)
}

func chunkInt64s(values []int64, size int) [][]int64 {
	if size <= 0 || len(values) <= size {
		return [][]int64{values}
	}
	chunks := make([][]int64, 0, (len(values)+size-1)/size)
	for start := 0; start < len(values); start += size {
		end := start + size
		if end > len(values) {
			end = len(values)
		}
		chunks = append(chunks, values[start:end])
	}
	return chunks
}

// ModelStatusService handles model availability monitoring
type ModelStatusService struct {
	db    *database.Manager
	logDB *database.Manager
}

func (s *ModelStatusService) normalizedTokenGroupExpr(alias string) string {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}
	return fmt.Sprintf("COALESCE(NULLIF(%s%s, ''), 'default')", prefix, s.getGroupCol())
}

func (s *ModelStatusService) tokenIDsForGroup(group string) ([]int64, error) {
	group = normalizeModelStatusGroup(group)
	if group == "all" {
		return nil, nil
	}

	query := s.db.RebindQuery(fmt.Sprintf(`
		SELECT id
		FROM tokens
		WHERE %s = ?`, s.normalizedTokenGroupExpr("")))
	rows, err := s.db.Query(query, group)
	if err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		id := toInt64(row["id"])
		if id > 0 {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (s *ModelStatusService) appendTokenIDFilter(conditions []string, args []interface{}, tokenIDs []int64) ([]string, []interface{}) {
	if len(tokenIDs) == 0 {
		return append(conditions, "1 = 0"), args
	}

	placeholders := make([]string, 0, len(tokenIDs))
	for _, id := range tokenIDs {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	conditions = append(conditions, fmt.Sprintf("l.token_id IN (%s)", strings.Join(placeholders, ",")))
	return conditions, args
}

func (s *ModelStatusService) groupModelsFromLogs(group string, startTime int64) ([]string, error) {
	group = normalizeModelStatusGroup(group)
	if group == "all" {
		return []string{}, nil
	}

	tokenIDs, err := s.tokenIDsForGroup(group)
	if err != nil {
		return nil, err
	}
	if len(tokenIDs) == 0 {
		return []string{}, nil
	}

	seen := map[string]bool{}
	for _, chunk := range chunkInt64s(tokenIDs, 500) {
		conditions := []string{"l.type IN (2, 5)", "l.model_name != ''", "l.created_at >= ?"}
		args := []interface{}{startTime}
		conditions, args = s.appendTokenIDFilter(conditions, args, chunk)

		query := s.logDB.RebindQuery(fmt.Sprintf(`
			SELECT DISTINCT l.model_name
			FROM logs l
			WHERE %s
			ORDER BY l.model_name`, strings.Join(conditions, " AND ")))
		rows, err := s.logDB.Query(query, args...)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if name, ok := row["model_name"].(string); ok && name != "" {
				seen[name] = true
			}
		}
	}

	models := make([]string, 0, len(seen))
	for name := range seen {
		models = append(models, name)
	}
	sort.Strings(models)
	return models, nil
}

func (s *ModelStatusService) queryAvailableModels(startTime int64, tokenIDs []int64) ([]map[string]interface{}, error) {
	conditions := []string{"l.type IN (2, 5)", "l.model_name != ''", "l.created_at >= ?"}
	args := []interface{}{startTime}
	if tokenIDs != nil {
		conditions, args = s.appendTokenIDFilter(conditions, args, tokenIDs)
	}

	query := s.logDB.RebindQuery(fmt.Sprintf(`
		SELECT l.model_name, COUNT(*) as request_count_24h
		FROM logs l
		WHERE %s
		GROUP BY l.model_name
		ORDER BY request_count_24h DESC`, strings.Join(conditions, " AND ")))
	return s.logDB.Query(query, args...)
}

func (s *ModelStatusService) queryModelStatusRows(modelName string, startTime, now, slotSeconds int64, tokenIDs []int64) ([]map[string]interface{}, error) {
	conditions := []string{"l.model_name = ?", "l.created_at >= ?", "l.created_at < ?", "l.type IN (2, 5)"}
	args := []interface{}{modelName, startTime, now}
	if tokenIDs != nil {
		conditions, args = s.appendTokenIDFilter(conditions, args, tokenIDs)
	}

	slotQuery := s.logDB.RebindQuery(fmt.Sprintf(`
		SELECT FLOOR((l.created_at - %d) / %d) as slot_idx,
			COUNT(*) as total,
			SUM(CASE WHEN l.type = 2 AND l.completion_tokens > 0 THEN 1 ELSE 0 END) as success,
			SUM(CASE WHEN l.type = 5 THEN 1 ELSE 0 END) as failure,
			SUM(CASE WHEN l.type = 2 AND l.completion_tokens = 0 THEN 1 ELSE 0 END) as empty
		FROM logs l
		WHERE %s
		GROUP BY FLOOR((l.created_at - %d) / %d)`,
		startTime, slotSeconds,
		strings.Join(conditions, " AND "),
		startTime, slotSeconds))
	return s.logDB.Query(slotQuery, args...)
}

// NewModelStatusService creates a new ModelStatusService
func NewModelStatusService() *ModelStatusService {
	return &ModelStatusService{db: database.Get(), logDB: database.GetLog()}
}

// GetAvailableModels returns all models with 24h request counts
func (s *ModelStatusService) GetAvailableModels(group string, noCache bool) ([]map[string]interface{}, error) {
	group = normalizeModelStatusGroup(group)
	cacheKey := fmt.Sprintf("model_status:available_models:%s", modelStatusCachePart(group))
	cm := cache.Get()
	var cached []map[string]interface{}
	found, _ := cm.GetJSON(cacheKey, &cached)
	if found && !noCache {
		return cached, nil
	}

	startTime := time.Now().Unix() - 86400
	var rows []map[string]interface{}
	if group == "all" {
		var err error
		rows, err = s.queryAvailableModels(startTime, nil)
		if err != nil {
			return nil, err
		}
	} else {
		tokenIDs, err := s.tokenIDsForGroup(group)
		if err != nil {
			return nil, err
		}
		counts := map[string]int64{}
		for _, chunk := range chunkInt64s(tokenIDs, 500) {
			chunkRows, err := s.queryAvailableModels(startTime, chunk)
			if err != nil {
				return nil, err
			}
			for _, row := range chunkRows {
				name, _ := row["model_name"].(string)
				if name != "" {
					counts[name] += toInt64(row["request_count_24h"])
				}
			}
		}
		rows = make([]map[string]interface{}, 0, len(counts))
		for name, count := range counts {
			rows = append(rows, map[string]interface{}{"model_name": name, "request_count_24h": count})
		}
		sort.Slice(rows, func(i, j int) bool {
			return toInt64(rows[i]["request_count_24h"]) > toInt64(rows[j]["request_count_24h"])
		})
	}

	cm.Set(cacheKey, rows, 5*time.Minute)
	return rows, nil
}

// GetModelStatus returns status for a specific model
// Uses a single GROUP BY FLOOR query (matches Python backend optimization)
func (s *ModelStatusService) GetModelStatus(modelName, window, group string, noCache bool) (map[string]interface{}, error) {
	group = normalizeModelStatusGroup(group)
	cacheKey := fmt.Sprintf("model_status:status:%s:%s:%s", modelStatusCachePart(group), modelStatusCachePart(modelName), modelStatusCachePart(window))
	cm := cache.Get()
	var cached map[string]interface{}
	found, _ := cm.GetJSON(cacheKey, &cached)
	if found && !noCache {
		return cached, nil
	}

	// Get window configuration (dynamic slot count per window)
	twConfig, ok := timeWindowConfigs[window]
	if !ok {
		twConfig = timeWindowConfigs["24h"]
	}

	now := time.Now().Unix()
	startTime := now - twConfig.totalSeconds
	numSlots := twConfig.numSlots
	slotSeconds := twConfig.slotSeconds

	// Single optimized query — aggregate by time slot using FLOOR division
	// This reduces N queries to 1 query per model (matches Python backend)
	//
	// Success counting strategy:
	//   - type=2 with completion_tokens > 0 → definite success
	//   - type=2 with completion_tokens = 0 → empty response (likely failure)
	//   - type=5 → explicit failure (if NewAPI version supports it)
	// This ensures correct success rate even when NewAPI doesn't log type=5 failures.
	var rows []map[string]interface{}
	if group == "all" {
		var err error
		rows, err = s.queryModelStatusRows(modelName, startTime, now, slotSeconds, nil)
		if err != nil {
			return nil, err
		}
	} else {
		tokenIDs, err := s.tokenIDsForGroup(group)
		if err != nil {
			return nil, err
		}
		slotTotals := map[int64]*modelStatusSlotInfo{}
		for _, chunk := range chunkInt64s(tokenIDs, 500) {
			chunkRows, err := s.queryModelStatusRows(modelName, startTime, now, slotSeconds, chunk)
			if err != nil {
				return nil, err
			}
			for _, row := range chunkRows {
				idx := toInt64(row["slot_idx"])
				info := slotTotals[idx]
				if info == nil {
					info = &modelStatusSlotInfo{}
					slotTotals[idx] = info
				}
				info.total += toInt64(row["total"])
				info.success += toInt64(row["success"])
				info.failure += toInt64(row["failure"])
				info.empty += toInt64(row["empty"])
			}
		}
		rows = make([]map[string]interface{}, 0, len(slotTotals))
		for idx, info := range slotTotals {
			rows = append(rows, map[string]interface{}{
				"slot_idx": idx,
				"total":    info.total,
				"success":  info.success,
				"failure":  info.failure,
				"empty":    info.empty,
			})
		}
	}

	// Initialize all slots with zeros
	slotMap := make(map[int64]*modelStatusSlotInfo, numSlots)

	// Fill in actual data from query results
	if rows != nil {
		for _, row := range rows {
			idx := toInt64(row["slot_idx"])
			if idx >= 0 && idx < int64(numSlots) {
				slotMap[idx] = &modelStatusSlotInfo{
					total:   toInt64(row["total"]),
					success: toInt64(row["success"]),
					failure: toInt64(row["failure"]),
					empty:   toInt64(row["empty"]),
				}
			}
		}
	}

	// Build slot_data list with status colors
	slotData := make([]map[string]interface{}, 0, numSlots)
	totalReqs := int64(0)
	totalSuccess := int64(0)
	totalFailure := int64(0)
	totalEmpty := int64(0)

	for i := 0; i < numSlots; i++ {
		slotStart := startTime + int64(i)*slotSeconds
		slotEnd := slotStart + slotSeconds

		si := slotMap[int64(i)]
		slotTotal := int64(0)
		slotSuccess := int64(0)
		slotFailure := int64(0)
		slotEmpty := int64(0)
		if si != nil {
			slotTotal = si.total
			slotSuccess = si.success
			slotFailure = si.failure
			slotEmpty = si.empty
		}

		slotRate := float64(100)
		if slotTotal > 0 {
			slotRate = float64(slotSuccess) / float64(slotTotal) * 100
		}

		slotData = append(slotData, map[string]interface{}{
			"slot":           i,
			"start_time":     slotStart,
			"end_time":       slotEnd,
			"total_requests": slotTotal,
			"success_count":  slotSuccess,
			"failure_count":  slotFailure,
			"empty_count":    slotEmpty,
			"success_rate":   roundRate(slotRate),
			"status":         getStatusColor(slotRate, slotTotal),
		})

		totalReqs += slotTotal
		totalSuccess += slotSuccess
		totalFailure += slotFailure
		totalEmpty += slotEmpty
	}

	overallRate := float64(100)
	if totalReqs > 0 {
		overallRate = float64(totalSuccess) / float64(totalReqs) * 100
	}

	result := map[string]interface{}{
		"model_name":     modelName,
		"display_name":   modelName,
		"time_window":    window,
		"group":          group,
		"total_requests": totalReqs,
		"success_count":  totalSuccess,
		"failure_count":  totalFailure,
		"empty_count":    totalEmpty,
		"success_rate":   roundRate(overallRate),
		"current_status": getStatusColor(overallRate, totalReqs),
		"slot_data":      slotData,
	}

	cm.Set(cacheKey, result, 30*time.Second)
	return result, nil
}

// GetMultipleModelsStatus returns status for multiple models
func (s *ModelStatusService) GetMultipleModelsStatus(modelNames []string, window, group string, noCache bool) ([]map[string]interface{}, error) {
	results := make([]map[string]interface{}, 0, len(modelNames))
	for _, name := range modelNames {
		status, err := s.GetModelStatus(name, window, group, noCache)
		if err != nil {
			continue
		}
		results = append(results, status)
	}
	return results, nil
}

// GetAllModelsStatus returns status for all models that have requests
func (s *ModelStatusService) GetAllModelsStatus(window, group string, noCache bool) ([]map[string]interface{}, error) {
	models, err := s.GetAvailableModels(group, noCache)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(models))
	for _, m := range models {
		if name, ok := m["model_name"].(string); ok {
			names = append(names, name)
		}
	}

	return s.GetMultipleModelsStatus(names, window, group, noCache)
}

// GetTokenGroups 返回令牌分组列表及其关联的模型（基于 abilities 表）
func (s *ModelStatusService) GetTokenGroups() ([]map[string]interface{}, error) {
	cm := cache.Get()
	var cached []map[string]interface{}
	found, _ := cm.GetJSON("model_status:token_groups", &cached)
	if found {
		return cached, nil
	}

	groupExpr := s.normalizedTokenGroupExpr("")
	query := s.db.RebindQuery(fmt.Sprintf(`
		SELECT %s as group_name,
			COUNT(*) as token_count,
			SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) as active_count
		FROM tokens
		WHERE deleted_at IS NULL
		GROUP BY %s
		ORDER BY token_count DESC`, groupExpr, groupExpr))

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}

	// 一次性读出 NewAPI 的分组描述（UserUsableGroups）和倍率（GroupRatio）
	descMap, ratioMap := s.loadGroupMetadata()
	startTime := time.Now().Unix() - 86400

	results := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		groupName := fmt.Sprintf("%v", row["group_name"])
		if groupName == "" || groupName == "<nil>" {
			groupName = "default"
		}

		modelNames, err := s.groupModelsFromLogs(groupName, startTime)
		if err != nil {
			modelNames = []string{}
		}

		entry := map[string]interface{}{
			"group_name":   groupName,
			"token_count":  row["token_count"],
			"active_count": row["active_count"],
			"model_count":  len(modelNames),
			"models":       modelNames,
		}
		if d, ok := descMap[groupName]; ok && d != "" && d != groupName {
			entry["description"] = d
		}
		if r, ok := ratioMap[groupName]; ok {
			entry["ratio"] = r
		}
		results = append(results, entry)
	}

	cm.Set("model_status:token_groups", results, 5*time.Minute)
	return results, nil
}

// loadGroupMetadata 一次性从 NewAPI 的 options 表读出分组描述和倍率配置。
// 返回两张 map，缺失时为 nil 不影响主流程。
func (s *ModelStatusService) loadGroupMetadata() (descMap map[string]string, ratioMap map[string]float64) {
	descMap = map[string]string{}
	ratioMap = map[string]float64{}

	keyCol := `"key"`
	if !s.db.IsPG {
		keyCol = "`key`"
	}
	query := s.db.RebindQuery(fmt.Sprintf(
		`SELECT %s as opt_key, value FROM options WHERE %s IN ('UserUsableGroups', 'GroupRatio')`,
		keyCol, keyCol))

	rows, err := s.db.Query(query)
	if err != nil {
		return
	}
	for _, row := range rows {
		key := fmt.Sprintf("%v", row["opt_key"])
		val, _ := row["value"].(string)
		if val == "" {
			continue
		}
		switch key {
		case "UserUsableGroups":
			_ = json.Unmarshal([]byte(val), &descMap)
		case "GroupRatio":
			// GroupRatio 的值可能是 number 或 string number，先按 number 解
			raw := map[string]interface{}{}
			if err := json.Unmarshal([]byte(val), &raw); err == nil {
				for k, v := range raw {
					switch n := v.(type) {
					case float64:
						ratioMap[k] = n
					case json.Number:
						if f, err := n.Float64(); err == nil {
							ratioMap[k] = f
						}
					}
				}
			}
		}
	}
	return
}

// getGroupCol 返回正确引用的 group 列名（group 是保留字）
func (s *ModelStatusService) getGroupCol() string {
	if s.db.IsPG {
		return `"group"`
	}
	return "`group`"
}

// Config management via cache

// GetSelectedModels returns selected model names from cache
func (s *ModelStatusService) GetSelectedModels() []string {
	cm := cache.Get()
	var models []string
	found, _ := cm.GetJSON("model_status:selected_models", &models)
	if found {
		return models
	}
	var config map[string]interface{}
	if loadLocalConfig("model_status:config", &config) {
		if raw, ok := config["selected_models"].([]interface{}); ok {
			models = make([]string, 0, len(raw))
			for _, item := range raw {
				if model := toString(item); model != "" {
					models = append(models, model)
				}
			}
			cm.Set("model_status:selected_models", models, 0)
			return models
		}
	}
	return []string{}
}

// SetSelectedModels saves selected models to cache
func (s *ModelStatusService) SetSelectedModels(models []string) {
	cm := cache.Get()
	cm.Set("model_status:selected_models", models, 0) // no expiry
	config := s.GetConfig()
	config["selected_models"] = models
	_ = saveLocalConfig("model_status:config", config)
}

// GetConfig returns all model status config
func (s *ModelStatusService) GetConfig() map[string]interface{} {
	cm := cache.Get()
	localConfig := map[string]interface{}{}
	loadLocalConfig("model_status:config", &localConfig)

	var timeWindow string
	found, _ := cm.GetJSON("model_status:time_window", &timeWindow)
	if !found {
		timeWindow = toString(localConfig["time_window"])
		if timeWindow == "" {
			timeWindow = DefaultTimeWindow
		}
	}

	var theme string
	found, _ = cm.GetJSON("model_status:theme", &theme)
	if !found {
		theme = toString(localConfig["theme"])
		if theme == "" {
			theme = DefaultTheme
		}
	}
	// Map legacy theme names to valid ones
	if mapped, ok := LegacyThemeMap[theme]; ok {
		theme = mapped
	}

	var refreshInterval int
	found, _ = cm.GetJSON("model_status:refresh_interval", &refreshInterval)
	if !found {
		refreshInterval = int(toInt64(localConfig["refresh_interval"]))
		if refreshInterval == 0 {
			refreshInterval = 60
		}
	}

	var sortMode string
	found, _ = cm.GetJSON("model_status:sort_mode", &sortMode)
	if !found {
		sortMode = toString(localConfig["sort_mode"])
		if sortMode == "" {
			sortMode = "default"
		}
	}

	var customOrder []string
	if found, _ := cm.GetJSON("model_status:custom_order", &customOrder); !found {
		if raw, ok := localConfig["custom_order"].([]interface{}); ok {
			for _, item := range raw {
				if v := toString(item); v != "" {
					customOrder = append(customOrder, v)
				}
			}
		}
	}

	var customGroups []map[string]interface{}
	found, _ = cm.GetJSON("model_status:custom_groups", &customGroups)
	if !found {
		customGroups = []map[string]interface{}{}
		if raw, ok := localConfig["custom_groups"].([]interface{}); ok {
			for _, item := range raw {
				if group, ok := item.(map[string]interface{}); ok {
					customGroups = append(customGroups, group)
				}
			}
		}
	}

	return map[string]interface{}{
		"time_window":      timeWindow,
		"theme":            theme,
		"refresh_interval": refreshInterval,
		"sort_mode":        sortMode,
		"custom_order":     customOrder,
		"selected_models":  s.GetSelectedModels(),
		"custom_groups":    customGroups,
		"site_title":       s.GetSiteTitle(),
	}
}

// SetTimeWindow saves time window to cache
func (s *ModelStatusService) SetTimeWindow(window string) {
	cm := cache.Get()
	cm.Set("model_status:time_window", window, 0)
	config := s.GetConfig()
	config["time_window"] = window
	_ = saveLocalConfig("model_status:config", config)
}

// SetTheme saves theme to cache
func (s *ModelStatusService) SetTheme(theme string) {
	cm := cache.Get()
	cm.Set("model_status:theme", theme, 0)
	config := s.GetConfig()
	config["theme"] = theme
	_ = saveLocalConfig("model_status:config", config)
}

// SetRefreshInterval saves refresh interval to cache
func (s *ModelStatusService) SetRefreshInterval(interval int) {
	cm := cache.Get()
	cm.Set("model_status:refresh_interval", interval, 0)
	config := s.GetConfig()
	config["refresh_interval"] = interval
	_ = saveLocalConfig("model_status:config", config)
}

// SetSortMode saves sort mode to cache
func (s *ModelStatusService) SetSortMode(mode string) {
	cm := cache.Get()
	cm.Set("model_status:sort_mode", mode, 0)
	config := s.GetConfig()
	config["sort_mode"] = mode
	_ = saveLocalConfig("model_status:config", config)
}

// SetCustomOrder saves custom order to cache
func (s *ModelStatusService) SetCustomOrder(order []string) {
	cm := cache.Get()
	cm.Set("model_status:custom_order", order, 0)
	config := s.GetConfig()
	config["custom_order"] = order
	_ = saveLocalConfig("model_status:config", config)
}

// GetCustomGroups returns custom model groups from cache
func (s *ModelStatusService) GetCustomGroups() []map[string]interface{} {
	cm := cache.Get()
	var groups []map[string]interface{}
	found, _ := cm.GetJSON("model_status:custom_groups", &groups)
	if found {
		return groups
	}
	config := map[string]interface{}{}
	if loadLocalConfig("model_status:config", &config) {
		if raw, ok := config["custom_groups"].([]interface{}); ok {
			groups = make([]map[string]interface{}, 0, len(raw))
			for _, item := range raw {
				if group, ok := item.(map[string]interface{}); ok {
					groups = append(groups, group)
				}
			}
			return groups
		}
	}
	return []map[string]interface{}{}
}

// SetCustomGroups saves custom model groups to cache
func (s *ModelStatusService) SetCustomGroups(groups []map[string]interface{}) {
	cm := cache.Get()
	cm.Set("model_status:custom_groups", groups, 0) // no expiry
	config := s.GetConfig()
	config["custom_groups"] = groups
	_ = saveLocalConfig("model_status:config", config)
}

// GetSiteTitle returns the custom site title
func (s *ModelStatusService) GetSiteTitle() string {
	cm := cache.Get()
	var title string
	found, _ := cm.GetJSON("model_status:site_title", &title)
	if found {
		return title
	}
	config := map[string]interface{}{}
	if loadLocalConfig("model_status:config", &config) {
		return toString(config["site_title"])
	}
	return ""
}

// SetSiteTitle saves the custom site title
func (s *ModelStatusService) SetSiteTitle(title string) {
	cm := cache.Get()
	cm.Set("model_status:site_title", title, 0)
	config := s.GetConfig()
	config["site_title"] = title
	_ = saveLocalConfig("model_status:config", config)
}

// GetEmbedConfig returns embed page configuration
func (s *ModelStatusService) GetEmbedConfig() map[string]interface{} {
	config := s.GetConfig()
	config["available_time_windows"] = AvailableTimeWindows
	config["available_themes"] = AvailableThemes
	config["available_refresh_intervals"] = AvailableRefreshIntervals
	config["available_sort_modes"] = AvailableSortModes
	return config
}
