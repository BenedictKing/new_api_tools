package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/new-api-tools/backend/frontend"
	"github.com/new-api-tools/backend/internal/auth"
	"github.com/new-api-tools/backend/internal/cache"
	"github.com/new-api-tools/backend/internal/config"
	"github.com/new-api-tools/backend/internal/database"
	"github.com/new-api-tools/backend/internal/handler"
	"github.com/new-api-tools/backend/internal/logger"
	"github.com/new-api-tools/backend/internal/middleware"
	"github.com/new-api-tools/backend/internal/tasks"
)

func main() {
	// ========== 1. Load configuration ==========
	cfg := config.Load()

	// ========== 2. Initialize logger ==========
	logger.Init(cfg.LogLevel, cfg.LogFile)
	logger.L.Banner("🚀 NewAPI Middleware Tool - Go Backend")
	logger.L.System(fmt.Sprintf("服务器地址: %s", cfg.ServerAddr()))
	logger.L.System(fmt.Sprintf("数据库引擎: %s", cfg.DatabaseEngine))
	logger.L.System(fmt.Sprintf("时区: %s", cfg.TimeZone))

	// ========== 3. Initialize database ==========
	_, err := database.Init(cfg)
	if err != nil {
		logger.L.Fatal("数据库初始化失败: " + err.Error())
	}
	defer database.Close()

	// ========== 4. Initialize Redis cache ==========
	if cfg.RedisConnString != "" {
		_, err := cache.Init(cfg.RedisConnString)
		if err != nil {
			logger.L.Warn("Redis 连接失败，将使用无缓存模式: " + err.Error())
		}
	} else {
		logger.L.Warn("REDIS_CONN_STRING 未配置，缓存功能不可用")
	}
	defer cache.Close()

	// ========== 5. Setup Gin router ==========
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Global middleware
	r.Use(middleware.ErrorHandlerMiddleware())  // Panic recovery
	r.Use(middleware.CORSMiddleware())          // CORS
	r.Use(middleware.RequestLoggerMiddleware()) // Request logging

	// ========== 6. Register routes ==========

	// Health check (no auth required)
	handler.RegisterHealthRoutes(r)
	handler.RegisterSwaggerRoutes(r)

	// API group with authentication
	api := r.Group("/api")
	api.Use(auth.AuthMiddleware())
	{
		// Auth routes (login/logout are whitelisted in middleware)
		handler.RegisterAuthRoutes(api)

		// Phase 2.1: Basic modules
		handler.RegisterRedemptionRoutes(api)
		handler.RegisterTopUpRoutes(api)
		handler.RegisterTopUpAnalyticsRoutes(api)
		handler.RegisterStorageRoutes(api)
		handler.RegisterSystemRoutes(api)

		// Phase 2.2: Dashboard, UserManagement, LogAnalytics
		handler.RegisterDashboardRoutes(api)
		handler.RegisterUserManagementRoutes(api)
		handler.RegisterAffiliateStatsRoutes(api)
		handler.RegisterLogAnalyticsRoutes(api)

		// Phase 2.3: IP Monitoring, Risk Monitoring, Model Status
		handler.RegisterIPMonitoringRoutes(api)
		handler.RegisterRiskMonitoringRoutes(api)
		handler.RegisterModelStatusRoutes(api)
		handler.RegisterAbuseBroadcastRoutes(api)

		// Phase 2.4: Token Management
		handler.RegisterTokenRoutes(api)

		// Phase 3: AI AutoBan, AutoGroup, LinuxDo Lookup
		handler.RegisterAIAutoBanRoutes(api)
		handler.RegisterAutoGroupRoutes(api)
		handler.RegisterLinuxDoRoutes(api)
	}

	// Public embed routes (no auth)
	handler.RegisterModelStatusEmbedRoutes(r)
	publicAPI := r.Group("/api")
	handler.RegisterStatusPageRoutes(publicAPI)
	frontend.ServeFrontend(r)

	// ========== 7. Background tasks ==========
	taskManager, err := tasks.InitDefaultTasks()
	if err != nil {
		logger.L.Fatal("后台任务初始化失败: " + err.Error())
	}
	taskManager.Start()
	defer taskManager.Stop()

	// ========== 8. Start server with graceful shutdown ==========
	srv := &http.Server{
		Addr:         cfg.ServerAddr(),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.L.Success(fmt.Sprintf("服务已启动: http://%s", cfg.ServerAddr()))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.L.Fatal("服务启动失败: " + err.Error())
		}
	}()

	// ========== 9. Wait for interrupt signal ==========
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.L.System("正在优雅关闭服务...")

	// Stop background tasks
	taskManager.Stop()

	// Give the server 10 seconds to finish processing requests
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.L.Error("服务关闭异常: " + err.Error())
	}

	logger.L.Success("服务已关闭")
}

// backgroundEnforceIPRecording periodically checks and enforces IP recording for all users.
