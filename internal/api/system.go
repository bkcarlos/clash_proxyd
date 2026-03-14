package api

import (
	"fmt"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/clash-proxyd/proxyd/internal/types"
	"github.com/gin-gonic/gin"
)

// SystemInfo represents system information
type SystemInfo struct {
	Version               string     `json:"version"`
	GoVersion             string     `json:"go_version"`
	Uptime                int64      `json:"uptime"`
	MihomoStatus          string     `json:"mihomo_status"`
	Database              string     `json:"database"`
	StartTime             time.Time  `json:"start_time"`
	RuntimeConfigPath     string     `json:"runtime_config_path"`
	LastAutoUpdateAction  string     `json:"last_auto_update_action,omitempty"`
	LastAutoUpdateDetails string     `json:"last_auto_update_details,omitempty"`
	LastAutoUpdateAt      *time.Time `json:"last_auto_update_at,omitempty"`
	LastAlertAction       string     `json:"last_alert_action,omitempty"`
	LastAlertDetails      string     `json:"last_alert_details,omitempty"`
	LastAlertAt           *time.Time `json:"last_alert_at,omitempty"`
}

var startTime = time.Now()

// GetSystemInfo returns system information
func (h *Handler) GetSystemInfo(c *gin.Context) {
	uptime := time.Since(startTime).Seconds()

	// Get mihomo status
	mihomoStatus := "stopped"
	if runtime, err := h.runtimeStore.Get(); err == nil {
		mihomoStatus = runtime.Status
	}

	// Use the configured default path — this is where new configs should be written.
	runtimeConfigPath := h.defaultRuntimeConfigPath()

	info := SystemInfo{
		Version:           "1.0.0",
		GoVersion:         runtime.Version(),
		Uptime:            int64(uptime),
		MihomoStatus:      mihomoStatus,
		Database:          "sqlite",
		StartTime:         startTime,
		RuntimeConfigPath: runtimeConfigPath,
	}

	hydrateLastAutoUpdateSummary(&info, h.auditStore)
	hydrateLastAlertSummary(&info, h.auditStore)
	h.respondJSON(c, http.StatusOK, info)
}

// GetSystemStatus returns detailed system status
func (h *Handler) GetSystemStatus(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	status := gin.H{
		"uptime":        time.Since(startTime).String(),
		"goroutines":    runtime.NumGoroutine(),
		"memory_alloc":  m.Alloc,
		"memory_sys":    m.Sys,
		"heap_alloc":    m.HeapAlloc,
		"heap_sys":      m.HeapSys,
		"heap_objects":  m.HeapObjects,
		"gc_cycles":     m.NumGC,
		"mihomo_status": "unknown",
	}

	// Get mihomo status
	if runtime, err := h.runtimeStore.Get(); err == nil {
		status["mihomo_status"] = runtime.Status
		status["mihomo_pid"] = runtime.PID
		status["mihomo_port"] = runtime.Port
		status["mihomo_uptime"] = runtime.Uptime
		status["mihomo_memory"] = runtime.MemoryUsage
	}

	if action, details, ts, ok := lastAutoUpdateSummary(h.auditStore); ok {
		status["last_auto_update"] = gin.H{
			"action":  action,
			"details": details,
			"at":      ts,
		}
	}

	if action, details, ts, ok := lastAlertSummary(h.auditStore); ok {
		status["last_alert"] = gin.H{
			"action":  action,
			"details": details,
			"at":      ts,
		}
	}

	h.respondJSON(c, http.StatusOK, status)
}

// GetSettings returns all system settings
func (h *Handler) GetSettings(c *gin.Context) {
	settings, err := h.settingStore.GetAll()
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, "Failed to get settings")
		return
	}

	h.respondJSON(c, http.StatusOK, settings)
}

// UpdateSetting updates a system setting
func (h *Handler) UpdateSetting(c *gin.Context) {
	var req struct {
		Key         string `json:"key" binding:"required"`
		Value       string `json:"value" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if err := h.settingStore.Set(req.Key, req.Value, req.Description); err != nil {
		h.respondError(c, http.StatusInternalServerError, "Failed to update setting")
		return
	}

	h.auditLog(c, "update_setting", "setting", "Updated setting: "+req.Key)
	h.respondSuccess(c, "Setting updated successfully", nil)
}

// UpdateSettingsBatch updates multiple system settings
func (h *Handler) UpdateSettingsBatch(c *gin.Context) {
	var req struct {
		Settings []struct {
			Key         string `json:"key" binding:"required"`
			Value       string `json:"value" binding:"required"`
			Description string `json:"description"`
		} `json:"settings" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	items := make([]types.Setting, 0, len(req.Settings))
	for _, s := range req.Settings {
		items = append(items, types.Setting{
			Key:         s.Key,
			Value:       s.Value,
			Description: s.Description,
		})
	}

	if err := h.settingStore.SetBatch(items); err != nil {
		h.respondError(c, http.StatusInternalServerError, "Failed to update settings")
		return
	}

	h.auditLog(c, "update_settings_batch", "setting", "Updated settings batch")
	h.respondSuccess(c, "Settings updated successfully", nil)
}

// GetAuditLogs returns audit logs
func (h *Handler) GetAuditLogs(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "50")

	// Parse pagination
	pageInt := 1
	limitInt := 50
	if p, err := parseInt(page); err == nil && p > 0 {
		pageInt = p
	}
	if l, err := parseInt(limit); err == nil && l > 0 && l <= 100 {
		limitInt = l
	}

	offset := (pageInt - 1) * limitInt

	logs, err := h.auditStore.List(limitInt, offset)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, "Failed to get audit logs")
		return
	}

	total, err := h.auditStore.Count()
	if err != nil {
		total = 0
	}

	h.respondJSON(c, http.StatusOK, gin.H{
		"logs":  logs,
		"total": total,
		"page":  pageInt,
		"limit": limitInt,
	})
}

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

func hydrateLastAutoUpdateSummary(info *SystemInfo, auditStore interface {
	List(limit, offset int) ([]types.AuditLog, error)
}) {
	action, details, ts, ok := lastAutoUpdateSummary(auditStore)
	if !ok {
		return
	}
	info.LastAutoUpdateAction = action
	info.LastAutoUpdateDetails = details
	info.LastAutoUpdateAt = &ts
}

func hydrateLastAlertSummary(info *SystemInfo, auditStore interface {
	List(limit, offset int) ([]types.AuditLog, error)
}) {
	action, details, ts, ok := lastAlertSummary(auditStore)
	if !ok {
		return
	}
	info.LastAlertAction = action
	info.LastAlertDetails = details
	info.LastAlertAt = &ts
}

func lastAutoUpdateSummary(auditStore interface {
	List(limit, offset int) ([]types.AuditLog, error)
}) (action, details string, at time.Time, ok bool) {
	if auditStore == nil {
		return "", "", time.Time{}, false
	}

	logs, err := auditStore.List(100, 0)
	if err != nil {
		return "", "", time.Time{}, false
	}

	for _, log := range logs {
		if log.Resource != "mihomo" {
			continue
		}
		if !strings.HasPrefix(log.Action, "mihomo_update_") {
			continue
		}
		return log.Action, log.Details, log.CreatedAt, true
	}

	return "", "", time.Time{}, false
}

func lastAlertSummary(auditStore interface {
	List(limit, offset int) ([]types.AuditLog, error)
}) (action, details string, at time.Time, ok bool) {
	if auditStore == nil {
		return "", "", time.Time{}, false
	}

	logs, err := auditStore.List(200, 0)
	if err != nil {
		return "", "", time.Time{}, false
	}

	for _, log := range logs {
		if log.Resource != "alert" {
			continue
		}
		if !strings.HasPrefix(log.Action, "alert_") {
			continue
		}
		return log.Action, log.Details, log.CreatedAt, true
	}

	return "", "", time.Time{}, false
}

// GetNetworkInterfaces returns local IPv4 addresses for proxy host selection.
func (h *Handler) GetNetworkInterfaces(c *gin.Context) {
	addrs := []string{"127.0.0.1"}

	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			ifAddrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, a := range ifAddrs {
				var ip net.IP
				switch v := a.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				if ip == nil || ip.IsLoopback() || ip.To4() == nil {
					continue
				}
				addrs = append(addrs, ip.String())
			}
		}
	}

	h.respondJSON(c, http.StatusOK, gin.H{"addresses": addrs})
}
