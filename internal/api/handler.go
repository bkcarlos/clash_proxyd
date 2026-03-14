package api

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/clash-proxyd/proxyd/internal/auth"
	"github.com/clash-proxyd/proxyd/internal/core"
	"github.com/clash-proxyd/proxyd/internal/logx"
	"github.com/clash-proxyd/proxyd/internal/renderer"
	"github.com/clash-proxyd/proxyd/internal/scheduler"
	"github.com/clash-proxyd/proxyd/internal/store"
	"github.com/clash-proxyd/proxyd/internal/types"
	"github.com/gin-gonic/gin"

	"go.uber.org/zap"
)

// Handler handles HTTP requests
type Handler struct {
	authManager     *auth.Manager
	sourceStore     *store.SourceStore
	settingStore    *store.SettingStore
	revisionStore   *store.RevisionStore
	runtimeStore    *store.RuntimeStore
	auditStore      *store.AuditStore
	mihomoManager   *core.Manager
	mihomoUpdater   *core.Updater
	renderer        *renderer.Renderer
	scheduler       *scheduler.Scheduler
	mihomoConfigDir string
	mihomoAPIPort   int
	proxydLogFile   string
	mihomoLogDir    string
	installJob      *InstallJob

	proxyDelayTTL   time.Duration
	proxyDelayCache map[string]proxyDelayCacheItem
	cacheMu         sync.Mutex
}

type proxyDelayCacheItem struct {
	Delay     int
	ExpiresAt time.Time
}

// NewHandler creates a new API handler
func NewHandler(
	authManager *auth.Manager,
	sourceStore *store.SourceStore,
	settingStore *store.SettingStore,
	revisionStore *store.RevisionStore,
	runtimeStore *store.RuntimeStore,
	auditStore *store.AuditStore,
	mihomoManager *core.Manager,
	mihomoUpdater *core.Updater,
	runtimeRenderer *renderer.Renderer,
	scheduler *scheduler.Scheduler,
	mihomoConfigDir string,
	mihomoAPIPort int,
	proxydLogFile string,
	mihomoLogDir string,
) *Handler {
	return &Handler{
		authManager:     authManager,
		sourceStore:     sourceStore,
		settingStore:    settingStore,
		revisionStore:   revisionStore,
		runtimeStore:    runtimeStore,
		auditStore:      auditStore,
		mihomoManager:   mihomoManager,
		mihomoUpdater:   mihomoUpdater,
		renderer:        runtimeRenderer,
		scheduler:       scheduler,
		mihomoConfigDir: mihomoConfigDir,
		mihomoAPIPort:   mihomoAPIPort,
		proxydLogFile:   proxydLogFile,
		mihomoLogDir:    mihomoLogDir,
		proxyDelayTTL:   15 * time.Second,
		proxyDelayCache: make(map[string]proxyDelayCacheItem),
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// respondError sends an error response
func (h *Handler) respondError(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, ErrorResponse{Error: message})
}

// respondSuccess sends a success response
func (h *Handler) respondSuccess(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, SuccessResponse{Message: message, Data: data})
}

// respondJSON sends a JSON response
func (h *Handler) respondJSON(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}

// getUser returns the authenticated user from context
func (h *Handler) getUser(c *gin.Context) string {
	if user, exists := c.Get("user"); exists {
		return user.(string)
	}
	return "system"
}

// auditLog logs an audit event
func (h *Handler) auditLog(c *gin.Context, action, resource string, details string) {
	log := &types.AuditLog{
		User:      h.getUser(c),
		Action:    action,
		Resource:  resource,
		Details:   details,
		IPAddress: c.ClientIP(),
	}
	if err := h.auditStore.Create(log); err != nil {
		// Log error but don't fail request
		logx.Error("Failed to create audit log", zap.Error(err))
	}
}

func (h *Handler) auditSystem(action, resource, details string) {
	if h.auditStore == nil {
		return
	}
	if err := h.auditStore.Create(&types.AuditLog{
		User:      "system",
		Action:    action,
		Resource:  resource,
		Details:   details,
		IPAddress: "127.0.0.1",
	}); err != nil {
		logx.Error("Failed to create system audit log", zap.Error(err))
	}
}

func (h *Handler) respondAndAuditFailure(c *gin.Context, statusCode int, action, resource string, err error) {
	message := fmt.Sprintf("%s failed: %v", action, err)
	h.auditLog(c, action+"_failed", resource, message)
	h.respondError(c, statusCode, message)
}

func (h *Handler) runtimeStatusFromManager(configPath string) (string, int, string) {
	if h.mihomoManager.IsRunning() {
		return "running", h.mihomoManager.GetPID(), configPath
	}
	return "stopped", 0, configPath
}

func (h *Handler) defaultRuntimeConfigPath() string {
	return filepath.Join(h.mihomoConfigDir, "runtime.yaml")
}

func (h *Handler) upsertRuntime(status string, pid int, configPath string) {
	runtimeState, err := h.runtimeStore.Get()
	if err != nil {
		_ = h.runtimeStore.Create(&types.Runtime{
			PID:         pid,
			Port:        h.mihomoAPIPort,
			ConfigPath:  configPath,
			Status:      status,
			Uptime:      0,
			MemoryUsage: 0,
		})
		return
	}

	runtimeState.PID = pid
	if configPath != "" {
		runtimeState.ConfigPath = configPath
	}
	runtimeState.Status = status
	_ = h.runtimeStore.Update(runtimeState)
}

func (h *Handler) getProxyDelayCache(key string) (int, bool) {
	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()
	item, ok := h.proxyDelayCache[key]
	if !ok {
		return 0, false
	}
	if time.Now().After(item.ExpiresAt) {
		delete(h.proxyDelayCache, key)
		return 0, false
	}
	return item.Delay, true
}

func (h *Handler) setProxyDelayCache(key string, delay int) {
	h.cacheMu.Lock()
	h.proxyDelayCache[key] = proxyDelayCacheItem{
		Delay:     delay,
		ExpiresAt: time.Now().Add(h.proxyDelayTTL),
	}
	h.cacheMu.Unlock()
}
