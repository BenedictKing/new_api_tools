package handler

import (
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/BenedictKing/new_api_tools/internal/logger"
	"github.com/BenedictKing/new_api_tools/internal/models"
	"github.com/BenedictKing/new_api_tools/internal/service"
	"github.com/BenedictKing/new_api_tools/pkg/geoip"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Service instances
var (
	redemptionService     = service.NewRedemptionService()
	userService           = service.NewUserService()
	userManagementService = service.NewUserManagementService()
	riskService           = service.NewRiskService()
	ipService             = service.NewIPService()
)

// parseWindowToSeconds 将窗口字符串解析为秒数
func parseWindowToSeconds(window string) int64 {
	switch window {
	case "1h":
		return 3600
	case "3h":
		return 3 * 3600
	case "6h":
		return 6 * 3600
	case "12h":
		return 12 * 3600
	case "24h":
		return 24 * 3600
	case "3d":
		return 3 * 24 * 3600
	case "7d":
		return 7 * 24 * 3600
	case "14d":
		return 14 * 24 * 3600
	default:
		return 24 * 3600 // 默认 24 小时
	}
}

func isSupportedWindow(window string) bool {
	switch window {
	case "1h", "3h", "6h", "12h", "24h", "3d", "7d", "14d":
		return true
	default:
		return false
	}
}

// ==================== Top-Up Handlers ====================

// GetTopUpsHandler 获取充值记录列表
// @Summary     获取充值记录列表
// @Tags        充值记录
// @Produce     json
// @Security    BearerAuth
// @Param       page       query     int     false  "页码"      default(1)
// @Param       page_size  query     int     false  "每页数量"  default(20)
// @Param       start_date query     string  false  "开始日期"
// @Param       end_date   query     string  false  "结束日期"
// @Success     200        {object}  Response{data=object}
// @Router      /top-ups [get]
func GetTopUpsHandler(c *gin.Context) {
	query := &service.ListTopUpParams{}
	if err := c.ShouldBindQuery(query); err != nil {
		Error(c, 400, "参数错误")
		return
	}

	data, err := service.ListTopUpRecords(*query)
	if err != nil {
		logger.Error("获取充值记录失败", zap.Error(err))
		Error(c, 500, "获取充值记录失败")
		return
	}

	Success(c, data)
}

// GetTopUpStatisticsHandler 获取充值统计
// @Summary     获取充值统计
// @Tags        充值记录
// @Produce     json
// @Security    BearerAuth
// @Param       start_date  query     string  false  "开始日期"
// @Param       end_date    query     string  false  "结束日期"
// @Success     200         {object}  Response{data=object}
// @Router      /top-ups/statistics [get]
func GetTopUpStatisticsHandler(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	data, err := service.GetTopUpStatistics(startDate, endDate)
	if err != nil {
		logger.Error("获取充值统计失败", zap.Error(err))
		Error(c, 500, "获取充值统计失败")
		return
	}

	Success(c, data)
}

// GetPaymentMethodsHandler 获取支付方式统计
// @Summary     获取支付方式统计
// @Tags        充值记录
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /top-ups/payment-methods [get]
func GetPaymentMethodsHandler(c *gin.Context) {
	data, err := service.GetPaymentMethods()
	if err != nil {
		logger.Error("获取支付方式统计失败", zap.Error(err))
		Error(c, 500, "获取支付方式统计失败")
		return
	}

	Success(c, data)
}

// RefundTopUpHandler 退款
// @Summary     充值退款
// @Tags        充值记录
// @Produce     json
// @Security    BearerAuth
// @Param       id   path      int  true  "充值记录 ID"
// @Success     200  {object}  Response{data=object}
// @Router      /top-ups/{id}/refund [post]
func RefundTopUpHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		Error(c, 400, "无效的 ID")
		return
	}

	if err := service.RefundTopUp(int64(id)); err != nil {
		logger.Error("退款失败", zap.Error(err))
		Error(c, 500, err.Error())
		return
	}

	Success(c, gin.H{"message": "退款成功"})
}

// ==================== Redemption Handlers ====================

// GetRedemptionsHandler 获取兑换码列表
// @Summary     获取兑换码列表
// @Tags        兑换码
// @Produce     json
// @Security    BearerAuth
// @Param       page       query     int     false  "页码"      default(1)
// @Param       page_size  query     int     false  "每页数量"  default(20)
// @Param       status     query     string  false  "状态过滤"
// @Success     200        {object}  Response{data=object}
// @Router      /redemptions [get]
func GetRedemptionsHandler(c *gin.Context) {
	query := &service.RedemptionQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		Error(c, 400, "参数错误")
		return
	}

	data, err := redemptionService.GetRedemptions(query)
	if err != nil {
		logger.Error("获取兑换码列表失败", zap.Error(err))
		Error(c, 500, "获取兑换码列表失败")
		return
	}

	Success(c, data)
}

// GetRedemptionStatisticsHandler 获取兑换码统计
// @Summary     获取兑换码统计
// @Tags        兑换码
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /redemptions/statistics [get]
func GetRedemptionStatisticsHandler(c *gin.Context) {
	data, err := redemptionService.GetRedemptionStatistics()
	if err != nil {
		logger.Error("获取兑换码统计失败", zap.Error(err))
		Error(c, 500, "获取兑换码统计失败")
		return
	}

	Success(c, data)
}

// GenerateRedemptionsHandler 批量生成兑换码
// @Summary     批量生成兑换码
// @Tags        兑换码
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      service.GenerateConfig  true  "生成配置"
// @Success     200   {object}  Response{data=object}
// @Router      /redemptions/generate [post]
func GenerateRedemptionsHandler(c *gin.Context) {
	var config service.GenerateConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		Error(c, 400, "参数错误")
		return
	}

	keys, err := redemptionService.GenerateRedemptions(&config)
	if err != nil {
		logger.Error("生成兑换码失败", zap.Error(err))
		Error(c, 500, err.Error())
		return
	}

	Success(c, gin.H{
		"count": len(keys),
		"keys":  keys,
	})
}

// DeleteRedemptionHandler 删除兑换码
// @Summary     删除兑换码
// @Tags        兑换码
// @Produce     json
// @Security    BearerAuth
// @Param       id   path      int  true  "兑换码 ID"
// @Success     200  {object}  Response{data=object}
// @Router      /redemptions/{id} [delete]
func DeleteRedemptionHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		Error(c, 400, "无效的 ID")
		return
	}

	if err := redemptionService.DeleteRedemption(id); err != nil {
		logger.Error("删除兑换码失败", zap.Error(err))
		Error(c, 500, err.Error())
		return
	}

	Success(c, gin.H{"message": "删除成功"})
}

// BatchDeleteRedemptionsHandler 批量删除兑换码
// @Summary     批量删除兑换码
// @Tags        兑换码
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      object  true  "兑换码 ID 列表 {ids: [1,2,3]}"
// @Success     200   {object}  Response{data=object}
// @Router      /redemptions/batch [delete]
func BatchDeleteRedemptionsHandler(c *gin.Context) {
	var req struct {
		IDs []int `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "参数错误")
		return
	}

	count, err := redemptionService.BatchDeleteRedemptions(req.IDs)
	if err != nil {
		logger.Error("批量删除兑换码失败", zap.Error(err))
		Error(c, 500, err.Error())
		return
	}

	Success(c, gin.H{"deleted": count})
}

// ==================== User Handlers ====================

// GetUsersHandler 获取用户列表
// @Summary     获取用户列表
// @Tags        用户管理
// @Produce     json
// @Security    BearerAuth
// @Param       page      query     int     false  "页码"      default(1)
// @Param       page_size query     int     false  "每页数量"  default(20)
// @Param       activity  query     string  false  "活跃度过滤"
// @Param       group     query     string  false  "分组过滤"
// @Param       source    query     string  false  "来源过滤"
// @Param       search    query     string  false  "搜索关键词"
// @Param       order_by  query     string  false  "排序字段"
// @Param       order_dir query     string  false  "排序方向 (asc/desc)"
// @Success     200       {object}  Response{data=object}
// @Router      /users [get]
func GetUsersHandler(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	params := service.ListUsersParams{
		Page:           page,
		PageSize:       pageSize,
		ActivityFilter: c.Query("activity"),
		GroupFilter:    c.Query("group"),
		SourceFilter:   c.Query("source"),
		Search:         c.Query("search"),
		OrderBy:        c.Query("order_by"),
		OrderDir:       c.Query("order_dir"),
	}

	data, err := userManagementService.GetUsers(params)
	if err != nil {
		logger.Error("获取用户列表失败", zap.Error(err))
		Error(c, 500, "获取用户列表失败")
		return
	}

	Success(c, data)
}

// GetUserStatsHandler 获取用户统计
// @Summary     获取用户活跃度统计
// @Tags        用户管理
// @Produce     json
// @Security    BearerAuth
// @Param       quick  query     bool  false  "快速模式"  default(false)
// @Success     200    {object}  Response{data=object}
// @Router      /users/stats [get]
func GetUserStatsHandler(c *gin.Context) {
	quick, _ := strconv.ParseBool(c.DefaultQuery("quick", "false"))

	data, err := userService.GetActivityStats(quick)
	if err != nil {
		logger.Error("获取用户统计失败", zap.Error(err))
		Error(c, 500, "获取用户统计失败")
		return
	}

	Success(c, gin.H{
		"total_users":         data.TotalUsers,
		"active_users":        data.ActiveUsers,
		"inactive_users":      data.InactiveUsers,
		"very_inactive_users": data.VeryInactiveUsers,
		"never_requested":     data.NeverRequested,
	})
}

// GetBannedUsersHandler 获取封禁用户列表
// @Summary     获取封禁用户列表
// @Tags        用户管理
// @Produce     json
// @Security    BearerAuth
// @Param       page       query     int     false  "页码"      default(1)
// @Param       page_size  query     int     false  "每页数量"  default(50)
// @Param       search     query     string  false  "搜索关键词"
// @Success     200        {object}  Response{data=object}
// @Router      /users/banned [get]
func GetBannedUsersHandler(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	search := c.Query("search")

	query := &service.UserQuery{
		Page:     page,
		PageSize: pageSize,
		Status:   models.UserStatusBanned,
		Search:   search,
	}

	data, err := userService.GetUsers(query)
	if err != nil {
		logger.Error("获取封禁用户列表失败", zap.Error(err))
		Error(c, 500, "获取封禁用户列表失败")
		return
	}

	Success(c, data)
}

// BanUserHandler 封禁用户
// @Summary     封禁用户
// @Tags        用户管理
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       user_id  path      int     true  "用户 ID"
// @Param       body     body      object  false  "封禁参数 {reason: string, disable_tokens: bool}"
// @Success     200      {object}  Response{data=object}
// @Router      /users/{user_id}/ban [post]
func BanUserHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		Error(c, 400, "无效的用户 ID")
		return
	}

	var req struct {
		Reason        string `json:"reason"`
		DisableTokens *bool  `json:"disable_tokens"`
	}
	c.ShouldBindJSON(&req)

	disableTokens := true
	if req.DisableTokens != nil {
		disableTokens = *req.DisableTokens
	}

	if err := userService.BanUser(id, req.Reason, disableTokens); err != nil {
		logger.Error("封禁用户失败", zap.Error(err))
		Error(c, 500, err.Error())
		return
	}

	Success(c, gin.H{"message": "封禁成功"})
}

// UnbanUserHandler 解封用户
// @Summary     解封用户
// @Tags        用户管理
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       user_id  path      int     true  "用户 ID"
// @Param       body     body      object  false  "解封参数 {enable_tokens: bool}"
// @Success     200      {object}  Response{data=object}
// @Router      /users/{user_id}/unban [post]
func UnbanUserHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		Error(c, 400, "无效的用户 ID")
		return
	}

	var req struct {
		EnableTokens *bool `json:"enable_tokens"`
	}
	c.ShouldBindJSON(&req)

	enableTokens := false
	if req.EnableTokens != nil {
		enableTokens = *req.EnableTokens
	}

	if err := userService.UnbanUser(id, enableTokens); err != nil {
		logger.Error("解封用户失败", zap.Error(err))
		Error(c, 500, err.Error())
		return
	}

	Success(c, gin.H{"message": "解封成功"})
}

// DeleteUserHandler 删除用户
// @Summary     删除用户
// @Tags        用户管理
// @Produce     json
// @Security    BearerAuth
// @Param       user_id  path      int  true  "用户 ID"
// @Success     200      {object}  Response{data=object}
// @Router      /users/{user_id} [delete]
func DeleteUserHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		Error(c, 400, "无效的用户 ID")
		return
	}

	if err := userService.DeleteUser(id); err != nil {
		logger.Error("删除用户失败", zap.Error(err))
		Error(c, 500, err.Error())
		return
	}

	Success(c, gin.H{"message": "删除成功"})
}

// BatchDeleteUsersHandler 批量删除用户（按活跃度级别）
// @Summary     批量删除用户
// @Tags        用户管理
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      object  true  "批量删除参数 {activity_level: string, dry_run: bool, hard_delete: bool}"
// @Success     200   {object}  Response{data=object}
// @Router      /users/batch-delete [post]
func BatchDeleteUsersHandler(c *gin.Context) {
	var req struct {
		ActivityLevel string `json:"activity_level"` // very_inactive, inactive 或 never
		DryRun        bool   `json:"dry_run"`        // 预演模式
		HardDelete    bool   `json:"hard_delete"`    // 彻底删除模式（物理删除）
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "参数错误")
		return
	}

	// 验证活跃度级别
	if req.ActivityLevel != "very_inactive" && req.ActivityLevel != "inactive" && req.ActivityLevel != "never" {
		Error(c, 400, "只能批量删除 very_inactive、inactive 或 never 级别的用户")
		return
	}

	result, err := userService.BatchDeleteUsersByActivity(req.ActivityLevel, req.DryRun, req.HardDelete)
	if err != nil {
		logger.Error("批量删除用户失败", zap.Error(err))
		Error(c, 500, err.Error())
		return
	}

	Success(c, result)
}

// GetSoftDeletedUsersCountHandler 获取已软删除用户的数量
// @Summary     获取已软删除用户数量
// @Tags        用户管理
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /users/soft-deleted/count [get]
func GetSoftDeletedUsersCountHandler(c *gin.Context) {
	result, err := userService.GetSoftDeletedUsersCount()
	if err != nil {
		logger.Error("获取软删除用户数量失败", zap.Error(err))
		Error(c, 500, err.Error())
		return
	}

	Success(c, result)
}

// PurgeSoftDeletedUsersHandler 彻底清理已软删除的用户（物理删除）
// @Summary     彻底清理软删除用户
// @Tags        用户管理
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      object  false  "清理参数 {dry_run: bool}"
// @Success     200   {object}  Response{data=object}
// @Router      /users/soft-deleted/purge [post]
func PurgeSoftDeletedUsersHandler(c *gin.Context) {
	var req struct {
		DryRun bool `json:"dry_run"` // 预览模式
	}
	// 默认为预览模式
	req.DryRun = true
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		Error(c, 400, "参数错误")
		return
	}

	result, err := userService.PurgeSoftDeletedUsers(req.DryRun)
	if err != nil {
		logger.Error("清理软删除用户失败", zap.Error(err))
		Error(c, 500, err.Error())
		return
	}

	Success(c, result)
}

// DisableTokenHandler 禁用令牌
// @Summary     禁用用户令牌
// @Tags        用户管理
// @Produce     json
// @Security    BearerAuth
// @Param       token_id  path      int  true  "令牌 ID"
// @Success     200       {object}  Response{data=object}
// @Router      /users/tokens/{token_id}/disable [post]
func DisableTokenHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("token_id"))
	if err != nil {
		Error(c, 400, "无效的令牌 ID")
		return
	}

	if err := userService.DisableUserToken(id); err != nil {
		logger.Error("禁用令牌失败", zap.Error(err))
		Error(c, 500, err.Error())
		return
	}

	Success(c, gin.H{"message": "禁用成功"})
}

// GetInvitedUsersHandler 获取被邀请用户列表
// @Summary     获取用户的邀请列表
// @Tags        用户管理
// @Produce     json
// @Security    BearerAuth
// @Param       user_id    path      int  true   "用户 ID"
// @Param       page       query     int  false  "页码"      default(1)
// @Param       page_size  query     int  false  "每页数量"  default(20)
// @Success     200        {object}  Response{data=object}
// @Router      /users/{user_id}/invited [get]
func GetInvitedUsersHandler(c *gin.Context) {
	inviterID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		Error(c, 400, "无效的用户 ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	data, err := userService.GetInvitedUsers(inviterID, page, pageSize)
	if err != nil {
		logger.Error("获取被邀请用户失败", zap.Error(err))
		Error(c, 500, "获取被邀请用户失败")
		return
	}

	// 直接返回 data，因为它已经包含了 success 字段
	c.JSON(200, data)
}

// ==================== Risk Monitoring Handlers ====================

// GetLeaderboardsHandler 获取排行榜
// @Summary     获取风控排行榜
// @Tags        风控监控
// @Produce     json
// @Security    BearerAuth
// @Param       windows  query     string  false  "时间窗口列表，逗号分隔"  default(1h,3h,6h,12h,24h)
// @Param       limit    query     int     false  "每个窗口返回数量"        default(10)
// @Param       sort_by  query     string  false  "排序字段 (requests/quota/failure_rate)"  default(requests)
// @Success     200      {object}  Response{data=object}
// @Router      /risk/leaderboards [get]
func GetLeaderboardsHandler(c *gin.Context) {
	windows := c.DefaultQuery("windows", "1h,3h,6h,12h,24h")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	sortBy := c.DefaultQuery("sort_by", "requests")

	if sortBy != "requests" && sortBy != "quota" && sortBy != "failure_rate" {
		Error(c, 400, "无效的 sort_by")
		return
	}

	windowList := []string{}
	for _, w := range strings.Split(windows, ",") {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}
		windowList = append(windowList, w)
	}

	data := map[string]interface{}{}
	for _, w := range windowList {
		if !isSupportedWindow(w) {
			continue
		}
		items, err := riskService.GetRealtimeLeaderboard(parseWindowToSeconds(w), limit, sortBy)
		if err != nil {
			logger.Error("获取排行榜失败", zap.Error(err))
			Error(c, 500, "获取排行榜失败")
			return
		}
		data[w] = items
	}

	Success(c, data)
}

// GetUserRiskAnalysisHandler 获取用户风险分析
// @Summary     获取用户风险分析
// @Tags        风控监控
// @Produce     json
// @Security    BearerAuth
// @Param       user_id  path      int     true   "用户 ID"
// @Param       window   query     string  false  "时间窗口"  default(24h)
// @Success     200      {object}  Response{data=object}
// @Router      /risk/users/{user_id}/analysis [get]
func GetUserRiskAnalysisHandler(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		Error(c, 400, "无效的用户 ID")
		return
	}

	window := c.DefaultQuery("window", "24h")
	if !isSupportedWindow(window) {
		Error(c, 400, "无效的 window")
		return
	}

	data, err := riskService.GetUserAnalysis(userID, parseWindowToSeconds(window))
	if err != nil {
		logger.Error("获取用户风险分析失败", zap.Error(err))
		Error(c, 500, err.Error())
		return
	}

	Success(c, data)
}

// GetBanRecordsHandler 获取封禁记录
// @Summary     获取封禁记录
// @Tags        风控监控
// @Produce     json
// @Security    BearerAuth
// @Param       page       query     int     false  "页码"                      default(1)
// @Param       page_size  query     int     false  "每页数量"                  default(50)
// @Param       action     query     string  false  "操作类型 (ban/unban)"
// @Param       user_id    query     int     false  "过滤用户 ID"
// @Success     200        {object}  Response{data=object}
// @Router      /risk/ban-records [get]
func GetBanRecordsHandler(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	action := c.Query("action")
	userID, _ := strconv.Atoi(c.DefaultQuery("user_id", "0"))

	if action != "" && action != "ban" && action != "unban" {
		Error(c, 400, "无效的 action")
		return
	}

	// 防止除零 panic
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}

	// Go 版仅实现 ban（基于 users.status），unban 记录需要审计日志支持
	if action == "unban" {
		Success(c, gin.H{
			"items":       []interface{}{},
			"total":       0,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": 0,
		})
		return
	}

	data, total, err := riskService.GetBanRecords(page, pageSize, userID)
	if err != nil {
		logger.Error("获取封禁记录失败", zap.Error(err))
		Error(c, 500, "获取封禁记录失败")
		return
	}

	// 计算总页数
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	Success(c, gin.H{
		"items":       data,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}

// GetTokenRotationHandler 获取令牌轮换情况
// @Summary     获取令牌轮换异常用户
// @Tags        风控监控
// @Produce     json
// @Security    BearerAuth
// @Param       window                query     string  false  "时间窗口"            default(24h)
// @Param       min_tokens            query     int     false  "最小令牌数"          default(5)
// @Param       max_requests_per_token query    int     false  "每令牌最大请求数"    default(10)
// @Param       limit                 query     int     false  "返回数量"            default(50)
// @Success     200                   {object}  Response{data=object}
// @Router      /risk/token-rotation [get]
func GetTokenRotationHandler(c *gin.Context) {
	window := c.DefaultQuery("window", "24h")
	if !isSupportedWindow(window) {
		Error(c, 400, "无效的 window")
		return
	}
	minTokens, _ := strconv.Atoi(c.DefaultQuery("min_tokens", "5"))
	maxRequestsPerToken, _ := strconv.Atoi(c.DefaultQuery("max_requests_per_token", "10"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	data, err := riskService.GetTokenRotationUsers(parseWindowToSeconds(window), minTokens, maxRequestsPerToken, limit)
	if err != nil {
		logger.Error("获取令牌轮换失败", zap.Error(err))
		Error(c, 500, "获取令牌轮换失败")
		return
	}

	Success(c, data)
}

// GetAffiliatedAccountsHandler 获取关联账户
// @Summary     获取关联账户（邀请关系）
// @Tags        风控监控
// @Produce     json
// @Security    BearerAuth
// @Param       min_invited       query     int   false  "最小邀请数"        default(3)
// @Param       include_activity  query     bool  false  "包含活跃度数据"    default(true)
// @Param       limit             query     int   false  "返回数量"          default(50)
// @Success     200               {object}  Response{data=object}
// @Router      /risk/affiliated-accounts [get]
func GetAffiliatedAccountsHandler(c *gin.Context) {
	minInvited, _ := strconv.Atoi(c.DefaultQuery("min_invited", "3"))
	includeActivity, _ := strconv.ParseBool(c.DefaultQuery("include_activity", "true"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	data, err := riskService.GetAffiliatedAccounts(minInvited, includeActivity, limit)
	if err != nil {
		logger.Error("获取关联账户失败", zap.Error(err))
		Error(c, 500, "获取关联账户失败")
		return
	}

	Success(c, data)
}

// GetSameIPRegistrationsHandler 获取同 IP 注册用户
// @Summary     获取同 IP 注册用户
// @Tags        风控监控
// @Produce     json
// @Security    BearerAuth
// @Param       window     query     string  false  "时间窗口"  default(7d)
// @Param       min_users  query     int     false  "最小用户数"  default(3)
// @Param       limit      query     int     false  "返回数量"    default(50)
// @Success     200        {object}  Response{data=object}
// @Router      /risk/same-ip-registrations [get]
func GetSameIPRegistrationsHandler(c *gin.Context) {
	window := c.DefaultQuery("window", "7d")
	if !isSupportedWindow(window) {
		Error(c, 400, "无效的 window")
		return
	}
	minUsers, _ := strconv.Atoi(c.DefaultQuery("min_users", "3"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	data, err := riskService.GetSameIPRegistrations(parseWindowToSeconds(window), minUsers, limit)
	if err != nil {
		logger.Error("获取同 IP 注册用户失败", zap.Error(err))
		Error(c, 500, "获取同 IP 注册用户失败")
		return
	}

	Success(c, data)
}

// ==================== IP Monitoring Handlers ====================

// GetIPStatsHandler 获取 IP 统计
// @Summary     获取 IP 统计信息
// @Tags        IP监控
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /ip-monitoring/stats [get]
func GetIPStatsHandler(c *gin.Context) {
	data, err := ipService.GetIPStats()
	if err != nil {
		logger.Error("获取 IP 统计失败", zap.Error(err))
		Error(c, 500, "获取 IP 统计失败")
		return
	}

	Success(c, data)
}

// GetSharedIPsHandler 获取共享 IP
// @Summary     获取共享 IP 列表
// @Tags        IP监控
// @Produce     json
// @Security    BearerAuth
// @Param       min_tokens  query     int     false  "最小令牌/用户数"  default(2)
// @Param       limit       query     int     false  "返回数量"         default(50)
// @Param       window      query     string  false  "时间窗口"         default(24h)
// @Success     200         {object}  Response{data=object}
// @Router      /ip-monitoring/shared-ips [get]
func GetSharedIPsHandler(c *gin.Context) {
	minTokensStr := c.Query("min_tokens")
	if minTokensStr == "" {
		minTokensStr = c.DefaultQuery("min_users", "2")
	}
	minTokens, _ := strconv.Atoi(minTokensStr)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	window := c.DefaultQuery("window", "24h")
	windowSeconds := parseWindowToSeconds(window)

	data, err := ipService.GetSharedIPs(minTokens, limit, windowSeconds)
	if err != nil {
		logger.Error("获取共享 IP 失败", zap.Error(err))
		Error(c, 500, "获取共享 IP 失败")
		return
	}

	Success(c, data)
}

// GetMultiIPTokensHandler 获取多 IP 令牌
// @Summary     获取使用多个 IP 的令牌
// @Tags        IP监控
// @Produce     json
// @Security    BearerAuth
// @Param       min_ips  query     int     false  "最小 IP 数"  default(5)
// @Param       limit    query     int     false  "返回数量"     default(50)
// @Param       window   query     string  false  "时间窗口"     default(24h)
// @Success     200      {object}  Response{data=object}
// @Router      /ip-monitoring/multi-ip-tokens [get]
func GetMultiIPTokensHandler(c *gin.Context) {
	minIPs, _ := strconv.Atoi(c.DefaultQuery("min_ips", "5"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	window := c.DefaultQuery("window", "24h")
	windowSeconds := parseWindowToSeconds(window)

	data, err := ipService.GetMultiIPTokens(minIPs, limit, windowSeconds)
	if err != nil {
		logger.Error("获取多 IP 令牌失败", zap.Error(err))
		Error(c, 500, "获取多 IP 令牌失败")
		return
	}

	Success(c, data)
}

// GetMultiIPUsersHandler 获取多 IP 用户
// @Summary     获取使用多个 IP 的用户
// @Tags        IP监控
// @Produce     json
// @Security    BearerAuth
// @Param       min_ips  query     int     false  "最小 IP 数"  default(10)
// @Param       limit    query     int     false  "返回数量"     default(50)
// @Param       window   query     string  false  "时间窗口"     default(24h)
// @Success     200      {object}  Response{data=object}
// @Router      /ip-monitoring/multi-ip-users [get]
func GetMultiIPUsersHandler(c *gin.Context) {
	minIPs, _ := strconv.Atoi(c.DefaultQuery("min_ips", "10"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	window := c.DefaultQuery("window", "24h")
	windowSeconds := parseWindowToSeconds(window)

	data, err := ipService.GetMultiIPUsers(minIPs, limit, windowSeconds)
	if err != nil {
		logger.Error("获取多 IP 用户失败", zap.Error(err))
		Error(c, 500, "获取多 IP 用户失败")
		return
	}

	Success(c, data)
}

// GetIPGeoHandler 获取单个 IP 地理信息
// @Summary     获取 IP 地理信息
// @Tags        IP监控
// @Produce     json
// @Security    BearerAuth
// @Param       ip   path      string  true  "IP 地址"
// @Success     200  {object}  Response{data=object}
// @Router      /ip-monitoring/geo/{ip} [get]
func GetIPGeoHandler(c *gin.Context) {
	ip := c.Param("ip")
	if ip == "" {
		ip = c.Query("ip")
	}
	if ip == "" {
		Error(c, 400, "缺少 IP 参数")
		return
	}

	data := ipService.GetIPGeo(ip)
	Success(c, data)
}

// BatchGetIPGeoHandler 批量获取 IP 地理信息
// @Summary     批量获取 IP 地理信息
// @Tags        IP监控
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      object  true  "IP 列表 {ips: [\"1.2.3.4\", ...]}"
// @Success     200   {object}  Response{data=object}
// @Router      /ip-monitoring/geo/batch [post]
func BatchGetIPGeoHandler(c *gin.Context) {
	var req struct {
		IPs []string `json:"ips"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "参数错误")
		return
	}

	data := ipService.BatchGetIPGeo(req.IPs)
	Success(c, data)
}

// GetIPAccessHistoryHandler 获取 IP 访问历史
func GetIPAccessHistoryHandler(c *gin.Context) {
	ip := c.Query("ip")
	if ip == "" {
		Error(c, 400, "缺少 IP 参数")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	data, err := ipService.GetIPAccessHistory(ip, limit)
	if err != nil {
		logger.Error("获取 IP 访问历史失败", zap.Error(err))
		Error(c, 500, "获取 IP 访问历史失败")
		return
	}

	Success(c, data)
}

// GetSuspiciousIPsHandler 获取可疑 IP
func GetSuspiciousIPsHandler(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	data, err := ipService.GetSuspiciousIPs(limit)
	if err != nil {
		logger.Error("获取可疑 IP 失败", zap.Error(err))
		Error(c, 500, "获取可疑 IP 失败")
		return
	}

	Success(c, data)
}

// GetIPDistributionHandler 获取 IP 分布统计
func GetIPDistributionHandler(c *gin.Context) {
	data, err := ipService.GetIPStats()
	if err != nil {
		logger.Error("获取 IP 分布失败", zap.Error(err))
		Error(c, 500, "获取 IP 分布失败")
		return
	}

	Success(c, gin.H{
		"countries":  data.TopCountries,
		"continents": data.TopContinents,
	})
}

// GetGeoIPStatusHandler 获取 GeoIP 服务状态
func GetGeoIPStatusHandler(c *gin.Context) {
	available := geoip.IsAvailable()
	Success(c, gin.H{
		"available": available,
		"message":   map[bool]string{true: "GeoIP 服务正常", false: "GeoIP 服务不可用"}[available],
	})
}

// GetUserIPsHandler 获取用户的 IP 列表
// @Summary     获取用户的 IP 访问列表
// @Tags        IP监控
// @Produce     json
// @Security    BearerAuth
// @Param       user_id  path      int     true   "用户 ID"
// @Param       limit    query     int     false  "返回数量"  default(100)
// @Param       window   query     string  false  "时间窗口"  default(24h)
// @Success     200      {object}  Response{data=object}
// @Router      /ip-monitoring/users/{user_id}/ips [get]
func GetUserIPsHandler(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		Error(c, 400, "无效的用户 ID")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	window := c.DefaultQuery("window", "24h")
	windowSeconds := parseWindowToSeconds(window)

	ips, err := ipService.GetUserIPs(userID, limit, windowSeconds)
	if err != nil {
		logger.Error("获取用户 IP 失败", zap.Error(err))
		Error(c, 500, "获取用户 IP 失败")
		return
	}

	Success(c, gin.H{"items": ips})
}

// GetIPIndexStatusHandler 获取 IP 索引状态
// @Summary     获取 IP 索引状态
// @Tags        IP监控
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /ip-monitoring/index-status [get]
func GetIPIndexStatusHandler(c *gin.Context) {
	status, err := ipService.GetIndexStatus()
	if err != nil {
		logger.Error("获取 IP 索引状态失败", zap.Error(err))
		Error(c, 500, "获取 IP 索引状态失败")
		return
	}
	Success(c, status)
}

// EnsureIPIndexesHandler 确保 IP 索引
// @Summary     确保 IP 相关数据库索引存在
// @Tags        IP监控
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /ip-monitoring/ensure-indexes [post]
func EnsureIPIndexesHandler(c *gin.Context) {
	results, created, existing, err := ipService.EnsureIndexes()
	if err != nil {
		logger.Error("确保 IP 索引失败", zap.Error(err))
		Error(c, 500, "确保 IP 索引失败")
		return
	}

	c.JSON(200, Response{
		Success: true,
		Message: "索引处理完成",
		Data: gin.H{
			"indexes":  results,
			"created":  created,
			"existing": existing,
		},
	})
}
