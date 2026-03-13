package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/BenedictKing/new_api_tools/internal/logger"
	"github.com/BenedictKing/new_api_tools/internal/service"
	"go.uber.org/zap"
)

var tokenService = service.NewTokenService()

// RegisterTokenRoutes 注册令牌路由
func RegisterTokenRoutes(r *gin.RouterGroup) {
	g := r.Group("/tokens")
	{
		g.GET("", GetTokens)
		g.GET("/statistics", GetTokenStatistics)
	}
}

// GetTokens 获取令牌列表
func GetTokens(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	name := c.Query("name")

	if status != "" && status != "active" && status != "disabled" && status != "expired" {
		Error(c, 400, "无效的状态过滤")
		return
	}

	params := service.TokenListParams{
		Page:     page,
		PageSize: pageSize,
		Status:   status,
		Name:     name,
	}

	data, err := tokenService.ListTokens(params)
	if err != nil {
		logger.Error("获取令牌列表失败", zap.Error(err))
		Error(c, 500, "获取令牌列表失败")
		return
	}

	Success(c, data)
}

// GetTokenStatistics 获取令牌统计
func GetTokenStatistics(c *gin.Context) {
	data, err := tokenService.GetTokenStatistics()
	if err != nil {
		logger.Error("获取令牌统计失败", zap.Error(err))
		Error(c, 500, "获取令牌统计失败")
		return
	}

	Success(c, data)
}
