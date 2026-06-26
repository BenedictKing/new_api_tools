package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/new-api-tools/backend/internal/models"
	"github.com/new-api-tools/backend/internal/service"
)

const defaultUptimeKumaWindow = "24h"

func RegisterStatusPageRoutes(r *gin.RouterGroup) {
	g := r.Group("/status-page")
	{
		g.GET("/heartbeat/:slug", GetStatusPageHeartbeat)
		g.GET("/:slug", GetStatusPageConfig)
		g.GET("/:slug/badge", GetStatusPageBadge)
		g.GET("/:slug/summary", GetStatusPageSummary)
	}
}

func GetStatusPageConfig(c *gin.Context) {
	data, err := service.NewUptimeKumaService().GetStatusPageConfig(c.Param("slug"), c.DefaultQuery("window", defaultUptimeKumaWindow))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("STATUS_PAGE_ERROR", "获取状态页配置失败", err.Error()))
		return
	}
	c.JSON(http.StatusOK, data)
}

func GetStatusPageHeartbeat(c *gin.Context) {
	data, err := service.NewUptimeKumaService().GetHeartbeatData(c.Param("slug"), c.DefaultQuery("window", defaultUptimeKumaWindow))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("STATUS_PAGE_ERROR", "获取心跳数据失败", err.Error()))
		return
	}
	c.JSON(http.StatusOK, data)
}

func GetStatusPageBadge(c *gin.Context) {
	data, err := service.NewUptimeKumaService().GetBadgeData(c.Param("slug"), c.DefaultQuery("window", defaultUptimeKumaWindow), c.DefaultQuery("label", ""))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("STATUS_PAGE_ERROR", "获取徽章数据失败", err.Error()))
		return
	}
	c.JSON(http.StatusOK, data)
}

func GetStatusPageSummary(c *gin.Context) {
	data, err := service.NewUptimeKumaService().GetSummaryData(c.Param("slug"), c.DefaultQuery("window", defaultUptimeKumaWindow))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResp("STATUS_PAGE_ERROR", "获取摘要数据失败", err.Error()))
		return
	}
	c.JSON(http.StatusOK, data)
}
