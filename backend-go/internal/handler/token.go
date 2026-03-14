package handler

import (
	"strconv"

	"github.com/BenedictKing/new_api_tools/internal/logger"
	"github.com/BenedictKing/new_api_tools/internal/service"
	"github.com/gin-gonic/gin"
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
// @Summary     获取令牌列表
// @Tags        令牌管理
// @Produce     json
// @Security    BearerAuth
// @Param       page       query     int     false  "页码"                              default(1)
// @Param       page_size  query     int     false  "每页数量"                          default(20)
// @Param       status     query     string  false  "状态过滤 (active/disabled/expired)"
// @Param       name       query     string  false  "名称过滤"
// @Success     200        {object}  Response{data=object}
// @Router      /tokens [get]
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
// @Summary     获取令牌统计信息
// @Tags        令牌管理
// @Produce     json
// @Security    BearerAuth
// @Success     200  {object}  Response{data=object}
// @Router      /tokens/statistics [get]
func GetTokenStatistics(c *gin.Context) {
	data, err := tokenService.GetTokenStatistics()
	if err != nil {
		logger.Error("获取令牌统计失败", zap.Error(err))
		Error(c, 500, "获取令牌统计失败")
		return
	}

	Success(c, data)
}
