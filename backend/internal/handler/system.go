package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/new-api-tools/backend/internal/database"
	"github.com/new-api-tools/backend/internal/service"
	"github.com/new-api-tools/backend/internal/tasks"
)

// RegisterSystemRoutes registers /api/system endpoints
func RegisterSystemRoutes(r *gin.RouterGroup) {
	g := r.Group("/system")
	{
		g.GET("/scale", GetSystemScale)
		g.POST("/scale/refresh", RefreshSystemScale)
		g.GET("/warmup-status", GetWarmupStatus)
		g.GET("/indexes", GetIndexStatus)
		g.POST("/indexes/ensure", EnsureIndexes)
	}
}

// GET /api/system/scale
func GetSystemScale(c *gin.Context) {
	data, err := service.NewSystemService().DetectScale(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

// POST /api/system/scale/refresh
func RefreshSystemScale(c *gin.Context) {
	data, err := service.NewSystemService().DetectScale(true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

// GET /api/system/warmup-status
func GetWarmupStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": tasks.GetManager().GetWarmupStatus()})
}

// GET /api/system/indexes
func GetIndexStatus(c *gin.Context) {
	db := database.Get()

	// Check existing indexes
	var indexes []struct {
		IndexName string `db:"indexname"`
	}

	var indexResults []gin.H
	total := 0
	existing := 0

	if db.IsPG {
		db.DB.Select(&indexes, "SELECT indexname FROM pg_indexes WHERE schemaname = 'public'")
	}

	// Build response matching Python format
	recommendedIndexes := []string{
		"idx_users_status",
		"idx_tokens_user_status",
		"idx_logs_created_type_user",
		"idx_logs_model_created",
		"idx_logs_token_created",
		"idx_logs_channel_created",
		"idx_redemptions_key",
		"idx_redemptions_status",
		"idx_top_ups_user",
		"idx_top_ups_status",
	}

	existingSet := make(map[string]bool)
	for _, idx := range indexes {
		existingSet[idx.IndexName] = true
	}

	for _, name := range recommendedIndexes {
		total++
		exists := existingSet[name]
		if exists {
			existing++
		}
		indexResults = append(indexResults, gin.H{
			"name":   name,
			"exists": exists,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"indexes":   indexResults,
			"total":     total,
			"existing":  existing,
			"missing":   total - existing,
			"all_ready": existing == total,
		},
	})
}

// POST /api/system/indexes/ensure
func EnsureIndexes(c *gin.Context) {
	db := database.Get()

	// Run index creation
	db.EnsureIndexes(true, 500*time.Millisecond)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Index creation completed",
		},
	})
}
