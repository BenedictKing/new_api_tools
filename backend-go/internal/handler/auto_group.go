package handler

import (
	"net/http"
	"strconv"

	"github.com/BenedictKing/new_api_tools/internal/service"
	"github.com/gin-gonic/gin"
)

var autoGroupService = service.NewAutoGroupService()

// RegisterAutoGroupRoutes 注册自动分组路由
func RegisterAutoGroupRoutes(r *gin.RouterGroup) {
	g := r.Group("/auto-group")
	{
		g.GET("/config", GetAutoGroupConfig)
		g.POST("/config", SaveAutoGroupConfig)
		g.GET("/groups", GetAutoGroupAvailableGroups)
		g.GET("/stats", GetAutoGroupStats)
		g.GET("/preview", GetAutoGroupPreview)
		g.GET("/logs", GetAutoGroupLogs)
		g.POST("/scan", RunAutoGroupScan)
		g.POST("/revert", RevertAutoGroupUser)
		g.POST("/batch-move", BatchMoveAutoGroupUsers)
	}
}

// GetAutoGroupConfig 获取自动分组配置
// @Summary     获取自动分组配置
// @Tags        自动分组
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /auto-group/config [get]
func GetAutoGroupConfig(c *gin.Context) {
	Success(c, autoGroupService.GetConfig())
}

// SaveAutoGroupConfig 保存自动分组配置
// @Summary     保存自动分组配置
// @Tags        自动分组
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      object  true  "配置项"
// @Success     200   {object}  Response{data=object}
// @Router      /auto-group/config [post]
func SaveAutoGroupConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "参数错误")
		return
	}
	if len(req) == 0 {
		Error(c, 400, "没有要保存的配置")
		return
	}

	if mode, ok := req["mode"].(string); ok {
		if mode != "simple" && mode != "by_source" {
			Error(c, 400, "无效的分组模式")
			return
		}
	}

	if interval, ok := req["scan_interval_minutes"]; ok {
		minutes := int64(0)
		switch v := interval.(type) {
		case float64:
			minutes = int64(v)
		case int:
			minutes = int64(v)
		case int64:
			minutes = v
		case string:
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				minutes = parsed
			}
		}
		if minutes < 1 || minutes > 1440 {
			Error(c, 400, "扫描间隔必须在 1-1440 分钟之间")
			return
		}
	}

	if !autoGroupService.SaveConfig(req) {
		Error(c, 500, "保存配置失败")
		return
	}

	Success(c, autoGroupService.GetConfig())
}

// GetAutoGroupAvailableGroups 获取可用分组
// @Summary     获取所有可用分组
// @Tags        自动分组
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /auto-group/groups [get]
func GetAutoGroupAvailableGroups(c *gin.Context) {
	groups := autoGroupService.GetAvailableGroups()
	Success(c, gin.H{
		"items": groups,
		"total": len(groups),
	})
}

// GetAutoGroupStats 获取自动分组统计
// @Summary     获取自动分组统计信息
// @Tags        自动分组
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /auto-group/stats [get]
func GetAutoGroupStats(c *gin.Context) {
	Success(c, autoGroupService.GetStats())
}

// GetAutoGroupPreview 获取待分组用户预览
// @Summary     获取待分组用户预览
// @Tags        自动分组
// @Produce     json
// @Security    BearerAuth
// @Param       page       query     int  false  "页码"      default(1)
// @Param       page_size  query     int  false  "每页数量"  default(20)
// @Success     200        {object}  Response{data=object}
// @Router      /auto-group/preview [get]
func GetAutoGroupPreview(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	Success(c, autoGroupService.GetPendingUsers(page, pageSize))
}

// GetAutoGroupLogs 获取自动分组日志
// @Summary     获取自动分组操作日志
// @Tags        自动分组
// @Produce     json
// @Security    BearerAuth
// @Param       page       query     int     false  "页码"                       default(1)
// @Param       page_size  query     int     false  "每页数量"                   default(20)
// @Param       action     query     string  false  "操作类型 (assign/revert)"
// @Param       user_id    query     int     false  "过滤用户 ID"
// @Success     200        {object}  Response{data=object}
// @Router      /auto-group/logs [get]
func GetAutoGroupLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	action := c.Query("action")
	if action != "" && action != "assign" && action != "revert" {
		Error(c, 400, "无效的 action")
		return
	}

	var userID *int64
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		parsed, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			Error(c, 400, "无效的用户 ID")
			return
		}
		userID = &parsed
	}

	Success(c, autoGroupService.GetLogs(page, pageSize, action, userID))
}

// RunAutoGroupScan 执行自动分组扫描
// @Summary     执行自动分组扫描
// @Tags        自动分组
// @Produce     json
// @Security    BearerAuth
// @Param       dry_run  query     bool  false  "预演模式"  default(true)
// @Success     200      {object}  Response{data=object}
// @Router      /auto-group/scan [post]
func RunAutoGroupScan(c *gin.Context) {
	dryRun, err := strconv.ParseBool(c.DefaultQuery("dry_run", "true"))
	if err != nil {
		Error(c, 400, "无效的 dry_run 参数")
		return
	}

	if !autoGroupService.IsEnabled() {
		Error(c, 400, "自动分组功能未启用")
		return
	}

	result := autoGroupService.RunScan(dryRun)
	success, _ := result["success"].(bool)
	if success {
		Success(c, result)
		return
	}

	message, _ := result["message"].(string)
	if message == "" {
		message = "扫描失败"
	}
	c.JSON(http.StatusOK, Response{
		Success: false,
		Message: message,
		Data:    result,
	})
}

// RevertAutoGroupUser 恢复用户分组
// @Summary     恢复用户分组到变更前状态
// @Tags        自动分组
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      object  true  "恢复参数 {log_id: int}"
// @Success     200   {object}  Response{data=object}
// @Router      /auto-group/revert [post]
func RevertAutoGroupUser(c *gin.Context) {
	var req struct {
		LogID int `json:"log_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "参数错误")
		return
	}
	if req.LogID <= 0 {
		Error(c, 400, "无效的日志 ID")
		return
	}

	result := autoGroupService.RevertUser(req.LogID)
	success, _ := result["success"].(bool)
	message, _ := result["message"].(string)
	c.JSON(http.StatusOK, Response{
		Success: success,
		Message: message,
		Data:    result,
	})
}

// BatchMoveAutoGroupUsers 批量移动用户分组
// @Summary     批量移动用户到指定分组
// @Tags        自动分组
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      object  true  "批量移动参数 {user_ids: [1,2,3], target_group: string}"
// @Success     200   {object}  Response{data=object}
// @Router      /auto-group/batch-move [post]
func BatchMoveAutoGroupUsers(c *gin.Context) {
	var req struct {
		UserIDs     []int64 `json:"user_ids"`
		TargetGroup string  `json:"target_group"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "参数错误")
		return
	}
	if len(req.UserIDs) == 0 {
		Error(c, 400, "未选择用户")
		return
	}
	if req.TargetGroup == "" {
		Error(c, 400, "未指定目标分组")
		return
	}

	result := autoGroupService.BatchMoveUsers(req.UserIDs, req.TargetGroup)
	success, _ := result["success"].(bool)
	message, _ := result["message"].(string)
	c.JSON(http.StatusOK, Response{
		Success: success,
		Message: message,
		Data:    result,
	})
}
