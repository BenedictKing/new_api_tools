package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/BenedictKing/new_api_tools/internal/service"
)

// RegisterLinuxDoLookupRoutes registers LinuxDo lookup endpoints.
func RegisterLinuxDoLookupRoutes(rg *gin.RouterGroup) {
	lookup := rg.Group("/linuxdo")
	{
		lookup.GET("/lookup/:linux_do_id", LinuxDoLookupHandler)
	}
}

// LinuxDoLookupHandler handles GET /api/linuxdo/lookup/:linux_do_id
func LinuxDoLookupHandler(c *gin.Context) {
	linuxDoID := c.Param("linux_do_id")
	if linuxDoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "linux_do_id 不能为空"})
		return
	}

	svc := service.NewLinuxDoLookupService()
	res, lerr := svc.LookupUsername(linuxDoID)
	if lerr != nil {
		c.JSON(lerr.StatusCode, gin.H{"success": false, "message": lerr.Message, "error_type": lerr.ErrorType, "wait_seconds": lerr.WaitSeconds})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": res})
}
