package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/clash-proxyd/proxyd/internal/auth"
	"github.com/gin-gonic/gin"
)

// SetupRouter sets up the API router.
// If webFS is non-nil the embedded Vue SPA is served for all non-API paths.
func (h *Handler) SetupRouter(authManager *auth.Manager, corsOrigins []string, webFS fs.FS) *gin.Engine {
	router := gin.New()

	// Middleware
	router.Use(auth.RecoveryMiddleware())
	router.Use(auth.RequestIDMiddleware())
	router.Use(auth.CORSMiddleware(corsOrigins))

	// Health check (no auth required)
	router.GET("/health", h.HealthCheck)
	router.GET("/ping", h.Ping)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public routes
		public := v1.Group("")
		{
			public.POST("/auth/login", h.Login)
			public.GET("/system/ws", h.SystemWS)
			public.GET("/proxy/mihomo/log-stream", h.MihomoLogStream)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(authManager.AuthMiddleware())
		{
			// Auth
			auth := protected.Group("/auth")
			{
				auth.POST("/logout", h.Logout)
				auth.POST("/refresh", h.RefreshToken)
				auth.GET("/profile", h.GetProfile)
				auth.PUT("/password", h.UpdatePassword)
			}

			// System
			system := protected.Group("/system")
			{
				system.GET("/info", h.GetSystemInfo)
				system.GET("/status", h.GetSystemStatus)
				system.GET("/settings", h.GetSettings)
				system.PUT("/settings", h.UpdateSetting)
				system.PUT("/settings/batch", h.UpdateSettingsBatch)
				system.GET("/audit-logs", h.GetAuditLogs)
			system.GET("/logs", h.GetLogs)
			system.GET("/network-interfaces", h.GetNetworkInterfaces)
			system.GET("/logs/download", h.DownloadLog)
			}

			// Sources
			sources := protected.Group("/sources")
			{
				sources.GET("", h.ListSources)
				sources.POST("", h.CreateSource)
				sources.GET("/:id", h.GetSource)
				sources.PUT("/:id", h.UpdateSource)
				sources.DELETE("/:id", h.DeleteSource)
				sources.POST("/:id/test", h.TestSource)
				sources.POST("/:id/fetch", h.FetchSource)
			}

			// Config
			config := protected.Group("/config")
			{
				config.POST("/generate", h.GenerateConfig)
			config.POST("/quick-apply", h.QuickApply)
				config.GET("", h.GetConfig)
				config.POST("/save", h.SaveConfig)
				config.POST("/apply", h.ApplyConfig)
				config.GET("/revisions", h.ListRevisions)
				config.GET("/revisions/:id", h.GetRevision)
				config.POST("/revisions/:id/rollback", h.RollbackRevision)
				config.DELETE("/revisions/:id", h.DeleteRevision)
			}

			// Policy
			policy := protected.Group("/policy")
			{
				policy.POST("/groups", h.GenerateGroups)
				policy.POST("/rules", h.GenerateRules)
				policy.POST("/validate-rule", h.ValidateRule)
				policy.POST("/custom-group", h.CreateCustomGroup)
			}

			// Proxy
			proxy := protected.Group("/proxy")
			{
				proxy.GET("/proxies", h.GetProxies)
				proxy.GET("/proxies/:name", h.GetProxy)
				proxy.POST("/proxies/:name/test", h.TestProxy)
				proxy.PUT("/groups/:group", h.SwitchProxy)
				proxy.GET("/groups", h.GetProxyGroups)
				proxy.GET("/rules", h.GetRules)
			proxy.GET("/connections", h.GetConnections)
			proxy.DELETE("/connections", h.CloseAllConnections)
			proxy.DELETE("/connections/:id", h.CloseConnection)
				proxy.GET("/traffic", h.GetTraffic)
				proxy.GET("/memory", h.GetMemory)
				proxy.GET("/mihomo/version", h.MihomoVersion)
				proxy.GET("/mihomo/releases", h.MihomoReleases)
				proxy.GET("/mihomo/versions", h.MihomoVersionList)
				proxy.GET("/mihomo/install-status", h.MihomoInstallStatus)
			proxy.GET("/mihomo/mmdb", h.MihomoMMDBStatus)
			proxy.POST("/mihomo/mmdb/download", h.MihomoMMDBDownload)
			proxy.POST("/mihomo/mmdb/upload", h.MihomoMMDBUpload)
				proxy.POST("/mihomo/update", h.MihomoUpdate)
			proxy.POST("/mihomo/install-job", h.StartInstallJob)
			proxy.GET("/mihomo/install-progress", h.GetInstallProgress)
				proxy.POST("/mihomo/:action", h.MihomoControl)
			}
		}
	}

	// Serve embedded Vue SPA for all non-API paths when webFS is provided.
	if webFS != nil {
		fileServer := http.FileServer(http.FS(webFS))
		router.NoRoute(func(c *gin.Context) {
			urlPath := strings.TrimPrefix(c.Request.URL.Path, "/")
			// Try to open the exact file from the embedded FS.
			f, err := webFS.Open(urlPath)
			if err == nil {
				f.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
			// Fall back to index.html so Vue Router can handle the path.
			c.Request.URL.Path = "/"
			fileServer.ServeHTTP(c.Writer, c.Request)
		})
	}

	return router
}

// HealthCheck returns health status
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "ok",
		"service": "proxyd",
	})
}

// Ping returns pong
func (h *Handler) Ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}
