package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/maharanidw/translator-pipeline/internal/transport/http/handlers"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*") // Izinkan dari semua URL/Domain
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		// Jika request adalah OPTIONS (Preflight dari Browser Chrome Extension), berikan 204
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// SetupRouter mendaftarkan semua endpoint yang ada
func SetupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 1. TAMBAHKAN MIDDLEWARE CORS (WAJIB AGAR BISA DIAKSES OLEH EXTENSION SECARA ONLINE)
	r.Use(CORSMiddleware())

	if err := r.SetTrustedProxies(nil); err != nil {
		// handle fail silently
	}

	r.GET("/api/v1/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "pong",
			"status":  "Translator Pipeline Backend Running!",
		})
	})

	api := r.Group("/api/v1")
	{
		// Catcher core app untuk ekstensi Chrome
		api.POST("/extract", handlers.ExtractText)

		// Dashboard web handler
		api.GET("/novels", handlers.GetNovels)
		api.DELETE("/novels/:id", handlers.DeleteNovel) // <-- Tambahan route DELETE
		api.GET("/novels/:id/chapters", handlers.GetChaptersByNovelID)
		api.GET("/chapters/:id", handlers.GetChapterByID)
		api.GET("/status", handlers.GetChapterStatusByURL)

		// System Logs Handler
		api.GET("/logs", handlers.GetSystemLogs)
		api.DELETE("/logs", handlers.ClearSystemLogs)
	}

	// Membuka akses untuk melayani file Web Dashboard (Frontend)
	r.Static("/dashboard", "./dashboard")

	return r
}
