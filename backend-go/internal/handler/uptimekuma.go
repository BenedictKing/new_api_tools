package handler

import (
	"github.com/BenedictKing/new_api_tools/internal/logger"
	"github.com/BenedictKing/new_api_tools/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var uptimeKumaService = service.NewUptimeKumaService()

const defaultUptimeKumaWindow = "24h"

// GetStatusPageConfigHandler 获取状态页配置（uptime-kuma 格式）
// GET /api/status-page/:slug
// @Summary     获取状态页配置
// @Tags        状态页
// @Produce     json
// @Param       slug    path      string  true   "状态页标识"
// @Param       window  query     string  false  "时间窗口"  default(24h)
// @Success     200     {object}  object
// @Router      /status-page/{slug} [get]
func GetStatusPageConfigHandler(c *gin.Context) {
	slug := c.Param("slug")
	window := c.DefaultQuery("window", defaultUptimeKumaWindow)

	data, err := uptimeKumaService.GetStatusPageConfig(slug, window)
	if err != nil {
		logger.Error("获取状态页配置失败", zap.Error(err), zap.String("slug", slug))
		Error(c, 500, "获取状态页配置失败")
		return
	}

	c.JSON(200, data)
}

// GetStatusPageHeartbeatHandler 获取心跳数据（uptime-kuma 格式）
// GET /api/status-page/heartbeat/:slug
// @Summary     获取状态页心跳数据
// @Tags        状态页
// @Produce     json
// @Param       slug    path      string  true   "状态页标识"
// @Param       window  query     string  false  "时间窗口"  default(24h)
// @Success     200     {object}  object
// @Router      /status-page/heartbeat/{slug} [get]
func GetStatusPageHeartbeatHandler(c *gin.Context) {
	slug := c.Param("slug")
	window := c.DefaultQuery("window", defaultUptimeKumaWindow)

	data, err := uptimeKumaService.GetHeartbeatData(slug, window)
	if err != nil {
		logger.Error("获取心跳数据失败", zap.Error(err), zap.String("slug", slug))
		Error(c, 500, "获取心跳数据失败")
		return
	}

	c.JSON(200, data)
}

// GetStatusPageBadgeHandler 获取徽章数据
// GET /api/status-page/:slug/badge
// @Summary     获取状态页徽章
// @Tags        状态页
// @Produce     json
// @Param       slug    path      string  true   "状态页标识"
// @Param       window  query     string  false  "时间窗口"  default(24h)
// @Param       label   query     string  false  "徽章标签"
// @Success     200     {object}  object
// @Router      /status-page/{slug}/badge [get]
func GetStatusPageBadgeHandler(c *gin.Context) {
	slug := c.Param("slug")
	window := c.DefaultQuery("window", defaultUptimeKumaWindow)
	label := c.DefaultQuery("label", "")

	data, err := uptimeKumaService.GetBadgeData(slug, window, label)
	if err != nil {
		logger.Error("获取徽章数据失败", zap.Error(err), zap.String("slug", slug))
		Error(c, 500, "获取徽章数据失败")
		return
	}

	c.JSON(200, data)
}

// GetStatusPageSummaryHandler 获取摘要数据
// GET /api/status-page/:slug/summary
// @Summary     获取状态页摘要
// @Tags        状态页
// @Produce     json
// @Param       slug    path      string  true   "状态页标识"
// @Param       window  query     string  false  "时间窗口"  default(24h)
// @Success     200     {object}  object
// @Router      /status-page/{slug}/summary [get]
func GetStatusPageSummaryHandler(c *gin.Context) {
	slug := c.Param("slug")
	window := c.DefaultQuery("window", defaultUptimeKumaWindow)

	data, err := uptimeKumaService.GetSummaryData(slug, window)
	if err != nil {
		logger.Error("获取摘要数据失败", zap.Error(err), zap.String("slug", slug))
		Error(c, 500, "获取摘要数据失败")
		return
	}

	c.JSON(200, data)
}
