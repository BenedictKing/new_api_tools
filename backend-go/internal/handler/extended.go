package handler

import (
	"encoding/json"
	"strconv"
	"time"

	appcache "github.com/BenedictKing/new_api_tools/internal/cache"
	"github.com/BenedictKing/new_api_tools/internal/logger"
	"github.com/BenedictKing/new_api_tools/internal/service"
	"github.com/BenedictKing/new_api_tools/internal/tasks"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Service instances for new modules
var (
	aiBanService       = service.NewAIAutoBanService()
	analyticsService   = service.NewAnalyticsService()
	modelStatusService = service.NewModelStatusService()
	systemService      = service.NewSystemService()
	storageService     = service.NewStorageService()
)

const (
	modelStatusSelectedModelsKey    = "model_status:selected_models"
	modelStatusTimeWindowKey        = "model_status:time_window"
	modelStatusThemeKey             = "model_status:theme"
	modelStatusRefreshIntervalKey   = "model_status:refresh_interval"
	defaultModelStatusTimeWindow    = "24h"
	defaultModelStatusTheme         = "daylight"
	defaultModelStatusRefreshSecond = 60
)

var (
	modelStatusAvailableThemes = []string{
		"daylight", "obsidian", "minimal", "neon", "forest", "ocean", "terminal", "cupertino", "material",
		"openai", "anthropic", "vercel", "linear", "stripe", "github", "discord", "tesla",
	}
	modelStatusAvailableRefreshIntervals = []int{0, 30, 60, 120, 300}
	modelStatusAvailableTimeWindows      = []string{"1h", "6h", "12h", "24h"}
)

func stringInSlice(v string, list []string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func toInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case float64:
		return int64(val)
	case string:
		i, _ := strconv.ParseInt(val, 10, 64)
		return i
	}
	return 0
}

func intInSlice(v int, list []int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// ==================== AI Ban Handlers ====================

// GetAIBanConfigHandler 获取 AI 封禁配置
// @Summary     获取 AI 自动封禁配置
// @Tags        AI封禁
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /ai-ban/config [get]
func GetAIBanConfigHandler(c *gin.Context) {
	data := aiBanService.GetConfig()
	Success(c, data)
}

// UpdateAIBanConfigHandler 更新 AI 封禁配置
// @Summary     更新 AI 自动封禁配置
// @Tags        AI封禁
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      object  true  "配置项键值对"
// @Success     200   {object}  Response{data=object}
// @Router      /ai-ban/config [post]
func UpdateAIBanConfigHandler(c *gin.Context) {
	var config map[string]interface{}
	if err := c.ShouldBindJSON(&config); err != nil {
		Error(c, 400, "参数错误")
		return
	}

	if err := aiBanService.SaveConfig(config); err != nil {
		logger.Error("更新 AI 封禁配置失败", zap.Error(err))
		Error(c, 500, err.Error())
		return
	}

	Success(c, gin.H{"message": "更新成功"})
}

// TestAIModelHandler 测试 AI 模型
// @Summary     测试 AI 模型连接
// @Tags        AI封禁
// @Produce     json
// @Security    BearerAuth
// @Param       api_key   query     string  true  "API Key"
// @Param       base_url  query     string  true  "Base URL"
// @Param       model     query     string  true  "模型名称"
// @Success     200       {object}  Response{data=object}
// @Router      /ai-ban/test-model [post]
func TestAIModelHandler(c *gin.Context) {
	// TestModel 需要参数
	apiKey := c.Query("api_key")
	baseURL := c.Query("base_url")
	model := c.Query("model")

	if apiKey == "" || baseURL == "" || model == "" {
		Error(c, 400, "缺少必要参数: api_key, base_url, model")
		return
	}

	data := aiBanService.TestModel(apiKey, baseURL, model)
	Success(c, data)
}

// GetSuspiciousUsersHandler 获取可疑用户
// @Summary     获取可疑用户列表
// @Tags        AI封禁
// @Produce     json
// @Security    BearerAuth
// @Param       window  query     string  false  "时间窗口"  default(24h)
// @Param       limit   query     int     false  "返回数量"  default(50)
// @Success     200     {object}  Response{data=object}
// @Router      /ai-ban/suspicious-users [get]
func GetSuspiciousUsersHandler(c *gin.Context) {
	window := c.DefaultQuery("window", "24h")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	data, err := aiBanService.GetSuspiciousUsers(window, limit)
	if err != nil {
		logger.Error("获取可疑用户失败", zap.Error(err))
		Error(c, 500, "获取可疑用户失败")
		return
	}

	Success(c, data)
}

// AssessUserRiskHandler 评估用户风险
// @Summary     AI 评估用户风险
// @Tags        AI封禁
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      object  true  "评估参数 {user_id: int, window: string}"
// @Success     200   {object}  Response{data=object}
// @Router      /ai-ban/assess [post]
func AssessUserRiskHandler(c *gin.Context) {
	var req struct {
		UserID int    `json:"user_id"`
		Window string `json:"window"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "参数错误")
		return
	}

	if req.UserID <= 0 {
		Error(c, 400, "无效的用户 ID")
		return
	}

	if req.Window == "" {
		req.Window = "1h"
	}

	data := aiBanService.ManualAssess(int64(req.UserID), req.Window)
	if errMsg, ok := data["error"]; ok {
		logger.Error("评估用户风险失败", zap.String("error", errMsg.(string)))
		Error(c, 500, errMsg.(string))
		return
	}

	Success(c, data)
}

// ScanUsersHandler 扫描用户
// @Summary     扫描可疑用户
// @Tags        AI封禁
// @Produce     json
// @Security    BearerAuth
// @Param       window  query     string  false  "时间窗口"  default(24h)
// @Param       limit   query     int     false  "扫描数量"  default(100)
// @Success     200     {object}  Response{data=object}
// @Router      /ai-ban/scan [post]
func ScanUsersHandler(c *gin.Context) {
	window := c.DefaultQuery("window", "24h")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	data, err := aiBanService.GetSuspiciousUsers(window, limit)
	if err != nil {
		logger.Error("扫描用户失败", zap.Error(err))
		Error(c, 500, "扫描失败")
		return
	}

	Success(c, data)
}

// GetWhitelistHandler 获取白名单
// @Summary     获取 AI 封禁白名单
// @Tags        AI封禁
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /ai-ban/whitelist [get]
func GetWhitelistHandler(c *gin.Context) {
	data := aiBanService.GetWhitelist()
	Success(c, data)
}

// AddToWhitelistHandler 添加到白名单
// @Summary     添加用户到白名单
// @Tags        AI封禁
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      object  true  "白名单参数 {user_id: int, reason: string}"
// @Success     200   {object}  Response{data=object}
// @Router      /ai-ban/whitelist/add [post]
func AddToWhitelistHandler(c *gin.Context) {
	var req struct {
		UserID int    `json:"user_id"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "参数错误")
		return
	}

	result := aiBanService.AddToWhitelist(int64(req.UserID))
	if errMsg, ok := result["error"]; ok {
		logger.Error("添加白名单失败", zap.String("error", errMsg.(string)))
		Error(c, 500, errMsg.(string))
		return
	}

	Success(c, gin.H{"message": "添加成功"})
}

// RemoveFromWhitelistHandler 从白名单移除
// @Summary     从白名单移除用户
// @Tags        AI封禁
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      object  true  "移除参数 {user_id: int}"
// @Success     200   {object}  Response{data=object}
// @Router      /ai-ban/whitelist/remove [post]
func RemoveFromWhitelistHandler(c *gin.Context) {
	var req struct {
		UserID int `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "参数错误")
		return
	}

	result := aiBanService.RemoveFromWhitelist(int64(req.UserID))
	if errMsg, ok := result["error"]; ok {
		logger.Error("移除白名单失败", zap.String("error", errMsg.(string)))
		Error(c, 500, errMsg.(string))
		return
	}

	Success(c, gin.H{"message": "移除成功"})
}

// SearchWhitelistHandler 搜索白名单
// @Summary     搜索白名单用户
// @Tags        AI封禁
// @Produce     json
// @Security    BearerAuth
// @Param       q  query     string  false  "搜索关键词"
// @Success     200  {object}  Response{data=object}
// @Router      /ai-ban/whitelist/search [get]
func SearchWhitelistHandler(c *gin.Context) {
	// 支持 q 和 keyword 两种参数名，优先使用 q（与前端一致）
	keyword := c.Query("q")
	if keyword == "" {
		keyword = c.Query("keyword")
	}
	data, err := aiBanService.SearchUserForWhitelist(keyword)
	if err != nil {
		Error(c, 500, "搜索失败")
		return
	}
	Success(c, data)
}

// GetAuditLogsHandler 获取审计日志
// @Summary     获取 AI 封禁审计日志
// @Tags        AI封禁
// @Produce     json
// @Security    BearerAuth
// @Param       page       query     int     false  "页码"      default(1)
// @Param       page_size  query     int     false  "每页数量"  default(20)
// @Param       status     query     string  false  "状态过滤"
// @Success     200        {object}  Response{data=object}
// @Router      /ai-ban/audit-logs [get]
func GetAuditLogsHandler(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.DefaultQuery("status", "")
	data := aiBanService.GetAuditLogs(page, pageSize, status)
	Success(c, data)
}

// DeleteAuditLogsHandler 删除审计日志
// @Summary     清空 AI 封禁审计日志
// @Tags        AI封禁
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /ai-ban/audit-logs [delete]
func DeleteAuditLogsHandler(c *gin.Context) {
	data := aiBanService.ClearAuditLogs()
	if errMsg, ok := data["error"]; ok {
		Error(c, 500, errMsg.(string))
		return
	}
	Success(c, gin.H{"message": "删除成功"})
}

// TestConnectionHandler 测试 AI 连接
// @Summary     测试 AI 服务连接
// @Tags        AI封禁
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /ai-ban/test-connection [post]
func TestConnectionHandler(c *gin.Context) {
	data := aiBanService.TestConnection()
	if errMsg, ok := data["error"]; ok {
		Error(c, 500, errMsg.(string))
		return
	}
	Success(c, data)
}

// ResetAPIHealthHandler 重置 API 健康状态
// @Summary     重置 AI API 健康状态
// @Tags        AI封禁
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /ai-ban/reset-api-health [post]
func ResetAPIHealthHandler(c *gin.Context) {
	data := aiBanService.ResetAPIHealth()
	if errMsg, ok := data["error"]; ok {
		Error(c, 500, errMsg.(string))
		return
	}
	Success(c, gin.H{"message": "重置成功"})
}

// UpdateAIModelsHandler 更新 AI 模型列表
// @Summary     更新 AI 封禁可用模型列表
// @Tags        AI封禁
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      object  true  "模型列表 {models: [\"gpt-4\", ...]}"
// @Success     200   {object}  Response{data=object}
// @Router      /ai-ban/models [post]
func UpdateAIModelsHandler(c *gin.Context) {
	var req struct {
		Models []string `json:"models"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "参数错误")
		return
	}
	Success(c, gin.H{"message": "更新成功", "models": req.Models})
}

// ==================== Analytics Handlers ====================

// GetAnalyticsStateHandler 获取分析状态
// @Summary     获取日志分析状态
// @Tags        日志分析
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /analytics/state [get]
func GetAnalyticsStateHandler(c *gin.Context) {
	data, err := analyticsService.GetState()
	if err != nil {
		logger.Error("获取分析状态失败", zap.Error(err))
		Error(c, 500, "获取状态失败")
		return
	}

	Success(c, data)
}

// ProcessLogsHandler 处理日志
// @Summary     处理待分析日志
// @Tags        日志分析
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  object
// @Router      /analytics/process [post]
func ProcessLogsHandler(c *gin.Context) {
	processed, lastLogID, err := analyticsService.ProcessLegacy()
	if err != nil {
		logger.Error("处理日志失败", zap.Error(err))
		c.JSON(200, gin.H{"success": false, "message": "处理失败: " + err.Error()})
		return
	}

	if processed == 0 {
		c.JSON(200, gin.H{
			"success":     true,
			"processed":   0,
			"message":     "No new logs to process",
			"last_log_id": lastLogID,
		})
		return
	}

	c.JSON(200, gin.H{
		"success":     true,
		"processed":   processed,
		"message":     "Processed new logs",
		"last_log_id": lastLogID,
	})
}

// GetUserRequestRankingHandler 获取用户请求排行
// @Summary     获取用户请求量排行
// @Tags        日志分析
// @Produce     json
// @Security    BearerAuth
// @Param       period  query     string  false  "时间周期"  default(today)
// @Param       limit   query     int     false  "返回数量"  default(20)
// @Success     200     {object}  Response{data=object}
// @Router      /analytics/users/requests [get]
func GetUserRequestRankingHandler(c *gin.Context) {
	period := c.DefaultQuery("period", "today")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	data, err := analyticsService.GetUserRequestRanking(period, limit)
	if err != nil {
		logger.Error("获取用户请求排行失败", zap.Error(err))
		Error(c, 500, "获取排行失败")
		return
	}

	Success(c, data)
}

// GetUserQuotaRankingHandler 获取用户额度排行
// @Summary     获取用户额度消耗排行
// @Tags        日志分析
// @Produce     json
// @Security    BearerAuth
// @Param       period  query     string  false  "时间周期"  default(today)
// @Param       limit   query     int     false  "返回数量"  default(20)
// @Success     200     {object}  Response{data=object}
// @Router      /analytics/users/quota [get]
func GetUserQuotaRankingHandler(c *gin.Context) {
	period := c.DefaultQuery("period", "today")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	data, err := analyticsService.GetUserQuotaRanking(period, limit)
	if err != nil {
		logger.Error("获取用户额度排行失败", zap.Error(err))
		Error(c, 500, "获取排行失败")
		return
	}

	Success(c, data)
}

// GetModelStatsHandler 获取模型统计
// @Summary     获取模型使用统计
// @Tags        日志分析
// @Produce     json
// @Security    BearerAuth
// @Param       period  query     string  false  "时间周期"  default(today)
// @Param       limit   query     int     false  "返回数量"  default(20)
// @Success     200     {object}  Response{data=object}
// @Router      /analytics/models [get]
func GetModelStatsHandler(c *gin.Context) {
	period := c.DefaultQuery("period", "today")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	data, err := analyticsService.GetModelStats(period, limit)
	if err != nil {
		logger.Error("获取模型统计失败", zap.Error(err))
		Error(c, 500, "获取统计失败")
		return
	}

	Success(c, data)
}

// GetAnalyticsSummaryHandler 获取分析摘要
// @Summary     获取日志分析完整摘要
// @Tags        日志分析
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /analytics/summary [get]
func GetAnalyticsSummaryHandler(c *gin.Context) {
	// 使用 GetFullSummary 返回前端期望的完整数据结构
	data, err := analyticsService.GetFullSummary()
	if err != nil {
		logger.Error("获取分析摘要失败", zap.Error(err))
		Error(c, 500, "获取摘要失败")
		return
	}

	Success(c, data)
}

// ResetAnalyticsHandler 重置分析
// @Summary     重置日志分析数据
// @Tags        日志分析
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  object
// @Router      /analytics/reset [post]
func ResetAnalyticsHandler(c *gin.Context) {
	if err := analyticsService.ResetLegacy(); err != nil {
		logger.Error("重置分析失败", zap.Error(err))
		c.JSON(200, gin.H{"success": false, "message": "重置失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true, "message": "重置成功"})
}

// ==================== Model Status Handlers ====================

// GetAvailableModelsHandler 获取可用模型
// @Summary     获取可用模型列表
// @Tags        模型状态
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  object
// @Router      /model-status/models [get]
func GetAvailableModelsHandler(c *gin.Context) {
	data, err := modelStatusService.GetAvailableModels()
	if err != nil {
		logger.Error("获取可用模型失败", zap.Error(err))
		Error(c, 500, "获取模型失败")
		return
	}
	c.JSON(200, gin.H{"success": true, "data": data})
}

// GetModelStatusHandler 获取模型状态
// @Summary     获取单个模型状态
// @Tags        模型状态
// @Produce     json
// @Security    BearerAuth
// @Param       model_name  path      string  true   "模型名称"
// @Param       window      query     string  false  "时间窗口 (1h/6h/12h/24h)"  default(24h)
// @Success     200         {object}  object
// @Router      /model-status/status/{model_name} [get]
func GetModelStatusHandler(c *gin.Context) {
	modelName := c.Param("model_name")
	if modelName == "" {
		c.JSON(200, gin.H{"success": false, "message": "缺少模型名称"})
		return
	}

	window := c.DefaultQuery("window", defaultModelStatusTimeWindow)
	if !stringInSlice(window, modelStatusAvailableTimeWindows) {
		c.JSON(200, gin.H{"success": false, "message": "无效的时间窗口"})
		return
	}

	data, err := modelStatusService.GetModelStatus(modelName, window)
	if err != nil {
		logger.Error("获取模型状态失败", zap.Error(err))
		c.JSON(200, gin.H{"success": false, "message": err.Error()})
		return
	}

	totalRequests := int64(0)
	if tr, ok := data["total_requests"]; ok {
		totalRequests = toInt64(tr)
	}

	if data == nil || totalRequests == 0 {
		c.JSON(200, gin.H{
			"success": false,
			"message": "Model not found or has no recent logs",
		})
		return
	}

	c.JSON(200, gin.H{"success": true, "data": data})
}

// BatchGetModelStatusHandler 批量获取模型状态
// @Summary     批量获取模型状态
// @Tags        模型状态
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       window  query     string  false  "时间窗口 (1h/6h/12h/24h)"  default(24h)
// @Param       body    body      object  true   "模型名称列表 {models: [\"gpt-4\", ...]} 或直接数组"
// @Success     200     {object}  object
// @Router      /model-status/status/batch [post]
func BatchGetModelStatusHandler(c *gin.Context) {
	window := c.DefaultQuery("window", defaultModelStatusTimeWindow)
	if !stringInSlice(window, modelStatusAvailableTimeWindows) {
		c.JSON(200, gin.H{"success": false, "message": "无效的时间窗口"})
		return
	}

	raw, err := c.GetRawData()
	if err != nil {
		Error(c, 400, "参数错误")
		return
	}

	var reqObj struct {
		Models []string `json:"models"`
	}
	var modelNames []string
	if json.Unmarshal(raw, &reqObj) == nil && len(reqObj.Models) > 0 {
		modelNames = reqObj.Models
	} else {
		var arr []string
		if err := json.Unmarshal(raw, &arr); err != nil {
			Error(c, 400, "参数错误")
			return
		}
		modelNames = arr
	}

	items, err := modelStatusService.GetMultipleModelsStatus(modelNames, window)
	if err != nil {
		logger.Error("批量获取模型状态失败", zap.Error(err))
		Error(c, 500, "获取状态失败")
		return
	}

	c.JSON(200, gin.H{
		"success":     true,
		"data":        items,
		"time_window": window,
		"cache_ttl":   60,
	})
}

// GetSelectedModelsHandler 获取选中的模型
// @Summary     获取已选中模型及配置
// @Tags        模型状态
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  object
// @Router      /model-status/config/selected [get]
func GetSelectedModelsHandler(c *gin.Context) {
	mgr := appcache.GetCacheManager()
	var selected []string
	if err := mgr.Get(modelStatusSelectedModelsKey, &selected); err != nil || len(selected) == 0 {
		available, _ := modelStatusService.GetAvailableModels()
		for _, m := range available {
			modelName := toString(m["model_name"])
			selected = append(selected, modelName)
		}
		_ = mgr.Set(modelStatusSelectedModelsKey, selected, 365*24*time.Hour)
	}

	var timeWindow string
	if err := mgr.Get(modelStatusTimeWindowKey, &timeWindow); err != nil || timeWindow == "" {
		timeWindow = defaultModelStatusTimeWindow
		_ = mgr.Set(modelStatusTimeWindowKey, timeWindow, 365*24*time.Hour)
	}

	var theme string
	if err := mgr.Get(modelStatusThemeKey, &theme); err != nil || theme == "" {
		theme = defaultModelStatusTheme
		_ = mgr.Set(modelStatusThemeKey, theme, 365*24*time.Hour)
	}

	var refreshInterval int
	if err := mgr.Get(modelStatusRefreshIntervalKey, &refreshInterval); err != nil || !intInSlice(refreshInterval, modelStatusAvailableRefreshIntervals) {
		refreshInterval = defaultModelStatusRefreshSecond
		_ = mgr.Set(modelStatusRefreshIntervalKey, refreshInterval, 365*24*time.Hour)
	}

	c.JSON(200, gin.H{
		"success":          true,
		"data":             selected,
		"time_window":      timeWindow,
		"theme":            theme,
		"refresh_interval": refreshInterval,
	})
}

// UpdateSelectedModelsHandler 更新选中的模型
// @Summary     更新已选中模型列表
// @Tags        模型状态
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      object  true  "模型列表 {models: [\"gpt-4\", ...]}"
// @Success     200   {object}  object
// @Router      /model-status/config/selected [post]
func UpdateSelectedModelsHandler(c *gin.Context) {
	var req struct {
		Models []string `json:"models"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "参数错误")
		return
	}
	mgr := appcache.GetCacheManager()
	_ = mgr.Set(modelStatusSelectedModelsKey, req.Models, 365*24*time.Hour)

	// 返回与 GetSelectedModelsHandler 一致的结构
	GetSelectedModelsHandler(c)
}

// GetTimeWindowHandler 获取时间窗口
// @Summary     获取当前时间窗口配置
// @Tags        模型状态
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  object
// @Router      /model-status/config/window [get]
func GetTimeWindowHandler(c *gin.Context) {
	mgr := appcache.GetCacheManager()
	var timeWindow string
	if err := mgr.Get(modelStatusTimeWindowKey, &timeWindow); err != nil || timeWindow == "" {
		timeWindow = defaultModelStatusTimeWindow
		_ = mgr.Set(modelStatusTimeWindowKey, timeWindow, 365*24*time.Hour)
	}
	c.JSON(200, gin.H{"success": true, "time_window": timeWindow})
}

// UpdateTimeWindowHandler 更新时间窗口
// @Summary     更新时间窗口配置
// @Tags        模型状态
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      object  true  "时间窗口 {time_window: \"24h\"}"
// @Success     200   {object}  object
// @Router      /model-status/config/window [post]
func UpdateTimeWindowHandler(c *gin.Context) {
	var req struct {
		TimeWindow string `json:"time_window"`
		Window     string `json:"window"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "参数错误")
		return
	}
	timeWindow := req.TimeWindow
	if timeWindow == "" {
		timeWindow = req.Window
	}
	if !stringInSlice(timeWindow, modelStatusAvailableTimeWindows) {
		mgr := appcache.GetCacheManager()
		var current string
		if err := mgr.Get(modelStatusTimeWindowKey, &current); err != nil || current == "" {
			current = defaultModelStatusTimeWindow
		}
		c.JSON(200, gin.H{
			"success":     false,
			"time_window": current,
			"message":     "无效的时间窗口: " + timeWindow,
		})
		return
	}

	mgr := appcache.GetCacheManager()
	_ = mgr.Set(modelStatusTimeWindowKey, timeWindow, 365*24*time.Hour)
	c.JSON(200, gin.H{
		"success":     true,
		"time_window": timeWindow,
		"message":     "已保存时间窗口: " + timeWindow,
	})
}

// GetTimeWindowsHandler 获取所有时间窗口选项
// @Summary     获取所有可用时间窗口
// @Tags        模型状态
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  object
// @Router      /model-status/windows [get]
func GetTimeWindowsHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"success": true,
		"data":    modelStatusAvailableTimeWindows,
		"default": defaultModelStatusTimeWindow,
	})
}

// GetAllModelStatusHandler 获取所有模型状态
// @Summary     获取所有模型状态
// @Tags        模型状态
// @Produce     json
// @Security    BearerAuth
// @Param       window  query     string  false  "时间窗口 (1h/6h/12h/24h)"  default(24h)
// @Success     200     {object}  object
// @Router      /model-status/status [get]
func GetAllModelStatusHandler(c *gin.Context) {
	window := c.DefaultQuery("window", defaultModelStatusTimeWindow)
	if !stringInSlice(window, modelStatusAvailableTimeWindows) {
		c.JSON(200, gin.H{"success": false, "message": "无效的时间窗口"})
		return
	}

	available, err := modelStatusService.GetAvailableModels()
	if err != nil {
		Error(c, 500, "获取失败")
		return
	}

	modelNames := make([]string, 0, len(available))
	for _, m := range available {
		modelName := toString(m["model_name"])
		modelNames = append(modelNames, modelName)
	}

	items, err := modelStatusService.GetMultipleModelsStatus(modelNames, window)
	if err != nil {
		Error(c, 500, "获取失败")
		return
	}

	c.JSON(200, gin.H{
		"success":     true,
		"data":        items,
		"time_window": window,
		"cache_ttl":   60,
	})
}

// GetThemeConfigHandler 获取主题配置
// @Summary     获取模型状态页主题配置
// @Tags        模型状态
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  object
// @Router      /model-status/config/theme [get]
func GetThemeConfigHandler(c *gin.Context) {
	mgr := appcache.GetCacheManager()
	var theme string
	if err := mgr.Get(modelStatusThemeKey, &theme); err != nil || theme == "" {
		theme = defaultModelStatusTheme
		_ = mgr.Set(modelStatusThemeKey, theme, 365*24*time.Hour)
	}
	c.JSON(200, gin.H{
		"success":          true,
		"theme":            theme,
		"available_themes": modelStatusAvailableThemes,
	})
}

// UpdateThemeConfigHandler 更新主题配置
// @Summary     更新模型状态页主题
// @Tags        模型状态
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      object  true  "主题配置 {theme: \"daylight\"}"
// @Success     200   {object}  object
// @Router      /model-status/config/theme [post]
func UpdateThemeConfigHandler(c *gin.Context) {
	var req struct {
		Theme string `json:"theme"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "参数错误")
		return
	}
	if !stringInSlice(req.Theme, modelStatusAvailableThemes) {
		mgr := appcache.GetCacheManager()
		var current string
		if err := mgr.Get(modelStatusThemeKey, &current); err != nil || current == "" {
			current = defaultModelStatusTheme
		}
		c.JSON(200, gin.H{
			"success":          false,
			"theme":            current,
			"available_themes": modelStatusAvailableThemes,
			"message":          "无效的主题: " + req.Theme,
		})
		return
	}
	mgr := appcache.GetCacheManager()
	_ = mgr.Set(modelStatusThemeKey, req.Theme, 365*24*time.Hour)
	c.JSON(200, gin.H{
		"success":          true,
		"theme":            req.Theme,
		"available_themes": modelStatusAvailableThemes,
		"message":          "已保存主题: " + req.Theme,
	})
}

// GetRefreshIntervalHandler 获取刷新间隔
// @Summary     获取模型状态刷新间隔配置
// @Tags        模型状态
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  object
// @Router      /model-status/config/refresh [get]
func GetRefreshIntervalHandler(c *gin.Context) {
	mgr := appcache.GetCacheManager()
	var interval int
	if err := mgr.Get(modelStatusRefreshIntervalKey, &interval); err != nil || !intInSlice(interval, modelStatusAvailableRefreshIntervals) {
		interval = defaultModelStatusRefreshSecond
		_ = mgr.Set(modelStatusRefreshIntervalKey, interval, 365*24*time.Hour)
	}
	c.JSON(200, gin.H{
		"success":             true,
		"refresh_interval":    interval,
		"available_intervals": modelStatusAvailableRefreshIntervals,
	})
}

// GetTokenGroupsHandler 获取模型监控可用令牌分组
// @Summary     获取模型监控令牌分组
// @Tags        模型状态
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  object
// @Router      /model-status/token-groups [get]
func GetTokenGroupsHandler(c *gin.Context) {
	data, err := modelStatusService.GetTokenGroups()
	if err != nil {
		logger.Error("获取模型监控令牌分组失败", zap.Error(err))
		c.JSON(200, gin.H{"success": false, "message": "获取令牌分组失败"})
		return
	}
	c.JSON(200, gin.H{"success": true, "data": data})
}

// GetSortConfigHandler 获取排序配置
// @Summary     获取模型状态排序配置
// @Tags        模型状态
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  object
// @Router      /model-status/config/sort [get]
func GetSortConfigHandler(c *gin.Context) {
	config := modelStatusService.GetConfig()
	c.JSON(200, gin.H{
		"success":              true,
		"sort_mode":            config["sort_mode"],
		"custom_order":         config["custom_order"],
		"available_sort_modes": service.AvailableSortModes,
	})
}

// UpdateSortConfigHandler 更新排序配置
// @Summary     更新模型状态排序配置
// @Tags        模型状态
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      object  true  "排序配置"
// @Success     200   {object}  object
// @Router      /model-status/config/sort [post]
func UpdateSortConfigHandler(c *gin.Context) {
	var raw map[string]json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		Error(c, 400, "参数错误")
		return
	}

	var req struct {
		SortMode    string   `json:"sort_mode"`
		CustomOrder []string `json:"custom_order"`
	}
	if data, ok := raw["sort_mode"]; ok {
		if err := json.Unmarshal(data, &req.SortMode); err != nil {
			Error(c, 400, "参数错误")
			return
		}
	}
	customOrderExists := false
	if data, ok := raw["custom_order"]; ok {
		customOrderExists = true
		if err := json.Unmarshal(data, &req.CustomOrder); err != nil {
			Error(c, 400, "参数错误")
			return
		}
	}
	if req.SortMode == "" {
		req.SortMode = "default"
	}
	config := modelStatusService.GetConfig()
	if !stringInSlice(req.SortMode, service.AvailableSortModes) {
		c.JSON(200, gin.H{
			"success":              false,
			"sort_mode":            config["sort_mode"],
			"custom_order":         config["custom_order"],
			"available_sort_modes": service.AvailableSortModes,
			"message":              "无效的排序模式: " + req.SortMode,
		})
		return
	}

	modelStatusService.SetSortMode(req.SortMode)
	if req.SortMode == "custom" && customOrderExists {
		modelStatusService.SetCustomOrder(req.CustomOrder)
	}

	updatedConfig := modelStatusService.GetConfig()
	c.JSON(200, gin.H{
		"success":              true,
		"sort_mode":            updatedConfig["sort_mode"],
		"custom_order":         updatedConfig["custom_order"],
		"available_sort_modes": service.AvailableSortModes,
		"message":              "排序配置已保存",
	})
}

// UpdateRefreshIntervalHandler 更新刷新间隔
// @Summary     更新模型状态刷新间隔
// @Tags        模型状态
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      object  true  "刷新间隔 {refresh_interval: 60}"
// @Success     200   {object}  object
// @Router      /model-status/config/refresh [post]
func UpdateRefreshIntervalHandler(c *gin.Context) {
	var req struct {
		RefreshInterval int `json:"refresh_interval"`
		Interval        int `json:"interval"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "参数错误")
		return
	}
	interval := req.RefreshInterval
	if interval == 0 && req.Interval != 0 {
		interval = req.Interval
	}
	if !intInSlice(interval, modelStatusAvailableRefreshIntervals) {
		mgr := appcache.GetCacheManager()
		var current int
		if err := mgr.Get(modelStatusRefreshIntervalKey, &current); err != nil || !intInSlice(current, modelStatusAvailableRefreshIntervals) {
			current = defaultModelStatusRefreshSecond
		}
		c.JSON(200, gin.H{
			"success":             false,
			"refresh_interval":    current,
			"available_intervals": modelStatusAvailableRefreshIntervals,
			"message":             "无效的刷新间隔: " + strconv.Itoa(interval),
		})
		return
	}
	mgr := appcache.GetCacheManager()
	_ = mgr.Set(modelStatusRefreshIntervalKey, interval, 365*24*time.Hour)
	c.JSON(200, gin.H{
		"success":             true,
		"refresh_interval":    interval,
		"available_intervals": modelStatusAvailableRefreshIntervals,
		"message":             "已保存刷新间隔: " + strconv.Itoa(interval) + "秒",
	})
}

// ==================== System Handlers ====================

// GetSystemScaleHandler 获取系统规模
// @Summary     获取系统规模检测结果
// @Tags        系统管理
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /system/scale [get]
func GetSystemScaleHandler(c *gin.Context) {
	// 使用新的 DetectScale 方法
	data, err := systemService.DetectScale(false)
	if err != nil {
		logger.Error("获取系统规模失败", zap.Error(err))
		Error(c, 500, "获取失败")
		return
	}

	Success(c, data)
}

// RefreshSystemScaleHandler 刷新系统规模
// @Summary     强制刷新系统规模检测
// @Tags        系统管理
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /system/scale/refresh [post]
func RefreshSystemScaleHandler(c *gin.Context) {
	// 强制刷新
	data, err := systemService.DetectScale(true)
	if err != nil {
		logger.Error("刷新系统规模失败", zap.Error(err))
		Error(c, 500, "刷新失败")
		return
	}

	Success(c, data)
}

// GetWarmupStatusHandler 获取预热状态
// 直接使用 tasks.WarmupStatus 中维护的状态，避免重复计算
// @Summary     获取系统预热状态
// @Tags        系统管理
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /system/warmup-status [get]
func GetWarmupStatusHandler(c *gin.Context) {
	// 从 tasks 包获取预热状态（已包含完整的 8 阶段信息）
	warmupStatus := tasks.GetWarmupStatus()

	// 直接使用 WarmupStatus 中的 Steps 构建响应
	steps := make([]gin.H, len(warmupStatus.Steps))
	for i, step := range warmupStatus.Steps {
		steps[i] = gin.H{
			"name":   step.Name,
			"status": step.Status,
		}
	}

	// 返回前端期望的格式
	data := gin.H{
		"status":       warmupStatus.Status,
		"progress":     warmupStatus.Progress,
		"message":      warmupStatus.Message,
		"steps":        steps,
		"started_at":   nil,
		"completed_at": nil,
	}

	if !warmupStatus.StartTime.IsZero() {
		data["started_at"] = warmupStatus.StartTime.Unix()
	}
	if warmupStatus.CompletedAt != nil {
		data["completed_at"] = warmupStatus.CompletedAt.Unix()
	}

	Success(c, data)
}

// GetIndexesHandler 获取索引
// @Summary     获取数据库索引列表
// @Tags        系统管理
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /system/indexes [get]
func GetIndexesHandler(c *gin.Context) {
	data, err := systemService.GetIndexes()
	if err != nil {
		logger.Error("获取索引失败", zap.Error(err))
		Error(c, 500, "获取失败")
		return
	}

	Success(c, data)
}

// EnsureIndexesHandler 确保索引
// @Summary     确保数据库索引存在
// @Tags        系统管理
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /system/indexes/ensure [post]
func EnsureIndexesHandler(c *gin.Context) {
	data, err := systemService.EnsureIndexes()
	if err != nil {
		logger.Error("确保索引失败", zap.Error(err))
		Error(c, 500, "操作失败")
		return
	}

	Success(c, data)
}

// ==================== Storage Handlers ====================

// GetStorageConfigHandler 获取存储配置
// @Summary     获取存储配置列表
// @Tags        存储管理
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /storage/config [get]
func GetStorageConfigHandler(c *gin.Context) {
	data, err := storageService.GetConfig()
	if err != nil {
		logger.Error("获取存储配置失败", zap.Error(err))
		Error(c, 500, "获取失败")
		return
	}

	Success(c, data)
}

// UpdateStorageConfigHandler 更新存储配置
// @Summary     更新存储配置
// @Tags        存储管理
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      service.StorageConfig  true  "存储配置"
// @Success     200   {object}  Response{data=object}
// @Router      /storage/config [post]
func UpdateStorageConfigHandler(c *gin.Context) {
	var config service.StorageConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		Error(c, 400, "参数错误")
		return
	}

	if err := storageService.UpdateConfig(&config); err != nil {
		logger.Error("更新存储配置失败", zap.Error(err))
		Error(c, 500, err.Error())
		return
	}

	Success(c, gin.H{"message": "更新成功"})
}

// CleanupCacheHandler 清理缓存
// @Summary     清理过期缓存
// @Tags        存储管理
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /storage/cache/cleanup [post]
func CleanupCacheHandler(c *gin.Context) {
	data, err := storageService.CleanupCache()
	if err != nil {
		logger.Error("清理缓存失败", zap.Error(err))
		Error(c, 500, "清理失败")
		return
	}

	Success(c, data)
}

// GetStorageConfigByKeyHandler 获取单个配置项
// @Summary     获取单个存储配置项
// @Tags        存储管理
// @Produce     json
// @Security    BearerAuth
// @Param       key  path      string  true  "配置键"
// @Success     200  {object}  Response{data=object}
// @Router      /storage/config/{key} [get]
func GetStorageConfigByKeyHandler(c *gin.Context) {
	key := c.Param("key")
	data, err := storageService.GetConfigByKey(key)
	if err != nil {
		Error(c, 500, "获取失败")
		return
	}
	Success(c, data)
}

// DeleteStorageConfigHandler 删除配置项
// @Summary     删除存储配置项
// @Tags        存储管理
// @Produce     json
// @Security    BearerAuth
// @Param       key  path      string  true  "配置键"
// @Success     200  {object}  Response{data=object}
// @Router      /storage/config/{key} [delete]
func DeleteStorageConfigHandler(c *gin.Context) {
	key := c.Param("key")
	if err := storageService.DeleteConfig(key); err != nil {
		Error(c, 500, "删除失败")
		return
	}
	Success(c, gin.H{"message": "删除成功"})
}

// GetCacheInfoHandler 获取缓存信息
// @Summary     获取缓存详细信息
// @Tags        存储管理
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /storage/cache/info [get]
func GetCacheInfoHandler(c *gin.Context) {
	data, err := storageService.GetCacheInfo()
	if err != nil {
		Error(c, 500, "获取失败")
		return
	}
	Success(c, data)
}

// ClearAllCacheHandler 清空所有缓存
// @Summary     清空所有缓存
// @Tags        存储管理
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /storage/cache [delete]
func ClearAllCacheHandler(c *gin.Context) {
	if err := storageService.ClearAllCache(); err != nil {
		Error(c, 500, "清空失败")
		return
	}
	Success(c, gin.H{"message": "缓存已清空"})
}

// ClearDashboardCacheHandler 清空仪表板缓存
// @Summary     清空仪表板缓存
// @Tags        存储管理
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /storage/cache/dashboard [delete]
func ClearDashboardCacheHandler(c *gin.Context) {
	if err := storageService.ClearDashboardCache(); err != nil {
		Error(c, 500, "清空失败")
		return
	}
	Success(c, gin.H{"message": "仪表板缓存已清空"})
}

// GetCacheStatsHandler 获取缓存统计
// @Summary     获取缓存统计信息
// @Tags        存储管理
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /storage/cache/stats [get]
func GetCacheStatsHandler(c *gin.Context) {
	data, err := storageService.GetCacheStats()
	if err != nil {
		Error(c, 500, "获取失败")
		return
	}
	Success(c, data)
}

// CleanupExpiredCacheHandler 清理过期缓存
// @Summary     清理过期缓存条目
// @Tags        存储管理
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /storage/cache/cleanup-expired [post]
func CleanupExpiredCacheHandler(c *gin.Context) {
	data, err := storageService.CleanupExpiredCache()
	if err != nil {
		Error(c, 500, "清理失败")
		return
	}
	Success(c, data)
}

// GetStorageInfoHandler 获取存储信息
// @Summary     获取存储空间信息
// @Tags        存储管理
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /storage/info [get]
func GetStorageInfoHandler(c *gin.Context) {
	data, err := storageService.GetStorageInfo()
	if err != nil {
		Error(c, 500, "获取失败")
		return
	}
	Success(c, data)
}

// ==================== Analytics Extended Handlers ====================

// BatchProcessLogsHandler 批量处理日志
// @Summary     批量处理日志（多轮迭代）
// @Tags        日志分析
// @Produce     json
// @Security    BearerAuth
// @Param       max_iterations  query     int  false  "最大迭代次数"  default(100)
// @Success     200             {object}  object
// @Router      /analytics/batch [post]
func BatchProcessLogsHandler(c *gin.Context) {
	maxIterations, _ := strconv.Atoi(c.DefaultQuery("max_iterations", "100"))
	data, err := analyticsService.BatchProcessLegacy(maxIterations)
	if err != nil {
		c.JSON(200, gin.H{"success": false, "message": "处理失败: " + err.Error()})
		return
	}
	c.JSON(200, data)
}

// GetSyncStatusHandler 获取同步状态
// @Summary     获取日志同步状态
// @Tags        日志分析
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /analytics/sync-status [get]
func GetSyncStatusHandler(c *gin.Context) {
	data, err := analyticsService.GetLegacySyncStatus()
	if err != nil {
		Error(c, 500, "获取失败")
		return
	}
	Success(c, data)
}

// CheckConsistencyHandler 检查数据一致性
// @Summary     检查日志数据一致性
// @Tags        日志分析
// @Produce     json
// @Security    BearerAuth
// @Param       auto_reset  query     bool  false  "是否自动重置"  default(false)
// @Success     200         {object}  object
// @Router      /analytics/check-consistency [post]
func CheckConsistencyHandler(c *gin.Context) {
	autoReset, _ := strconv.ParseBool(c.DefaultQuery("auto_reset", "false"))

	if autoReset {
		result, err := analyticsService.CheckAndAutoResetLegacy(true)
		if err != nil {
			c.JSON(200, gin.H{"success": false, "message": "检查失败: " + err.Error()})
			return
		}
		c.JSON(200, gin.H{
			"success":            true,
			"reset":              result.Reset,
			"reason":             result.Reason,
			"old_last_log_id":    result.OldLastLogID,
			"current_max_log_id": result.CurrentMaxLogID,
		})
		return
	}

	status, err := analyticsService.GetLegacySyncStatus()
	if err != nil {
		c.JSON(200, gin.H{"success": false, "message": "检查失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"success":           true,
		"data_inconsistent": status.DataInconsistent,
		"needs_reset":       status.NeedsReset,
		"last_log_id":       status.LastLogID,
		"max_log_id":        status.MaxLogID,
	})
}

// ==================== TopUp Extended Handlers ====================

// GetTopUpByIDHandler 获取单个充值记录
// @Summary     根据 ID 获取充值记录
// @Tags        充值记录
// @Produce     json
// @Security    BearerAuth
// @Param       id   path      int  true  "充值记录 ID"
// @Success     200  {object}  Response{data=object}
// @Router      /top-ups/{id} [get]
func GetTopUpByIDHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		Error(c, 400, "无效的 ID")
		return
	}
	data, err := service.GetTopUpByID(int64(id))
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, data)
}

// ==================== Model Status Embed Handlers (Public) ====================

// GetEmbedTimeWindowsHandler [公开] 获取时间窗口
// @Summary     [公开] 获取可用时间窗口
// @Tags        模型状态(公开)
// @Produce     json
// @Success     200  {object}  object
// @Router      /model-status/embed/windows [get]
func GetEmbedTimeWindowsHandler(c *gin.Context) {
	GetTimeWindowsHandler(c)
}

// GetEmbedAvailableModelsHandler [公开] 获取可用模型
// @Summary     [公开] 获取可用模型列表
// @Tags        模型状态(公开)
// @Produce     json
// @Success     200  {object}  object
// @Router      /model-status/embed/models [get]
func GetEmbedAvailableModelsHandler(c *gin.Context) {
	GetAvailableModelsHandler(c)
}

// GetEmbedModelStatusHandler [公开] 获取模型状态
// @Summary     [公开] 获取单个模型状态
// @Tags        模型状态(公开)
// @Produce     json
// @Param       model_name  path      string  true   "模型名称"
// @Param       window      query     string  false  "时间窗口"  default(24h)
// @Success     200         {object}  object
// @Router      /model-status/embed/status/{model_name} [get]
func GetEmbedModelStatusHandler(c *gin.Context) {
	GetModelStatusHandler(c)
}

// BatchGetEmbedModelStatusHandler [公开] 批量获取模型状态
// @Summary     [公开] 批量获取模型状态
// @Tags        模型状态(公开)
// @Accept      json
// @Produce     json
// @Param       window  query     string  false  "时间窗口"  default(24h)
// @Param       body    body      object  true   "模型名称列表"
// @Success     200     {object}  object
// @Router      /model-status/embed/status/batch [post]
func BatchGetEmbedModelStatusHandler(c *gin.Context) {
	BatchGetModelStatusHandler(c)
}

// GetEmbedAllModelStatusHandler [公开] 获取所有模型状态
// @Summary     [公开] 获取所有模型状态
// @Tags        模型状态(公开)
// @Produce     json
// @Param       window  query     string  false  "时间窗口"  default(24h)
// @Success     200     {object}  object
// @Router      /model-status/embed/status [get]
func GetEmbedAllModelStatusHandler(c *gin.Context) {
	GetAllModelStatusHandler(c)
}

// GetEmbedSelectedModelsHandler [公开] 获取选中模型配置
// @Summary     [公开] 获取已选中模型及配置
// @Tags        模型状态(公开)
// @Produce     json
// @Success     200  {object}  object
// @Router      /model-status/embed/config/selected [get]
func GetEmbedSelectedModelsHandler(c *gin.Context) {
	GetSelectedModelsHandler(c)
}

// GetEmbedTokenGroupsHandler [公开] 获取令牌分组
// @Summary     [公开] 获取模型监控令牌分组
// @Tags        模型状态(公开)
// @Produce     json
// @Success     200  {object}  object
// @Router      /model-status/embed/token-groups [get]
func GetEmbedTokenGroupsHandler(c *gin.Context) {
	GetTokenGroupsHandler(c)
}
