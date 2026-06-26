package frontend

import (
	"embed"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var embeddedFrontend embed.FS

func ServeFrontend(r *gin.Engine) {
	distFS, err := fs.Sub(embeddedFrontend, "dist")
	if err != nil {
		registerFrontendError(r)
		return
	}

	r.StaticFS("/assets", http.FS(distFS))
	r.GET("/", serveIndex(distFS))

	r.NoRoute(func(c *gin.Context) {
		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if isAPIPath(c.Request.URL.Path) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "API endpoint not found",
				"path":    c.Request.URL.Path,
				"message": "请求的 API 端点不存在",
			})
			return
		}

		if path != "" {
			if content, err := fs.ReadFile(distFS, path); err == nil {
				c.Data(http.StatusOK, contentType(path), content)
				return
			}
		}

		serveIndex(distFS)(c)
	})
}

func serveIndex(distFS fs.FS) gin.HandlerFunc {
	return func(c *gin.Context) {
		content, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			c.Data(http.StatusServiceUnavailable, "text/html; charset=utf-8", []byte(frontendErrorPage()))
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	}
}

func registerFrontendError(r *gin.Engine) {
	r.GET("/", func(c *gin.Context) {
		c.Data(http.StatusServiceUnavailable, "text/html; charset=utf-8", []byte(frontendErrorPage()))
	})
}

func isAPIPath(path string) bool {
	return strings.HasPrefix(path, "/api/") || path == "/health" || path == "/openapi.json" || strings.HasPrefix(path, "/swagger/")
}

func contentType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	default:
		return "application/octet-stream"
	}
}

func frontendErrorPage() string {
	return `<!doctype html><html><head><meta charset="utf-8"><title>NewAPI Tools</title></head><body><h1>前端资源未构建</h1><p>请先运行 make embed-frontend 或将 frontend/dist 复制到 backend/frontend/dist。</p></body></html>`
}
