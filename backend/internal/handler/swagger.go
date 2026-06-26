package handler

import (
	"embed"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

//go:embed openapi.json
var openAPIFS embed.FS

func RegisterSwaggerRoutes(r *gin.Engine) {
	r.GET("/openapi.json", func(c *gin.Context) {
		data, err := os.ReadFile("openapi.json")
		if err != nil {
			data, err = openAPIFS.ReadFile("openapi.json")
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "OpenAPI document not available",
			})
			return
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", data)
	})

	r.GET("/swagger/*any", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerHTML))
	})
}

const swaggerHTML = `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>NewAPI Tools API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>body{margin:0;background:#fff}#swagger-ui{min-height:100vh}</style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: "/openapi.json",
      dom_id: "#swagger-ui",
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis],
      layout: "BaseLayout"
    })
  </script>
</body>
</html>`
