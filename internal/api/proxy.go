package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/clash-proxyd/proxyd/internal/core"
	"github.com/gin-gonic/gin"
)

func (h *Handler) mihomoUnavailable(c *gin.Context, err error) {
	h.respondError(c, http.StatusServiceUnavailable, "Mihomo unavailable: "+err.Error())
}

// GetProxies returns all proxies from mihomo
func (h *Handler) GetProxies(c *gin.Context) {
	proxies, err := h.mihomoManager.GetProxies()
	if err != nil {
		h.mihomoUnavailable(c, err)
		return
	}

	h.respondJSON(c, http.StatusOK, proxies)
}

// GetProxy returns a specific proxy
func (h *Handler) GetProxy(c *gin.Context) {
	name := c.Param("name")
	proxy, err := h.mihomoManager.GetProxy(name)
	if err != nil {
		h.mihomoUnavailable(c, err)
		return
	}

	h.respondJSON(c, http.StatusOK, proxy)
}

// TestProxy tests a proxy delay
func (h *Handler) TestProxy(c *gin.Context) {
	name := c.Param("name")

	var req struct {
		URL     string `json:"url"`
		Timeout int    `json:"timeout"`
	}
	_ = c.ShouldBindJSON(&req)

	testURL := c.DefaultQuery("url", req.URL)
	if testURL == "" {
		testURL = "http://www.gstatic.com/generate_204"
	}

	timeoutInt := req.Timeout
	if timeoutInt <= 0 {
		timeoutStr := c.DefaultQuery("timeout", "3000")
		if parsed, err := strconv.Atoi(timeoutStr); err == nil && parsed > 0 {
			timeoutInt = parsed
		} else {
			timeoutInt = 3000
		}
	}

	cacheKey := fmt.Sprintf("%s|%s|%d", name, testURL, timeoutInt)
	if cachedDelay, ok := h.getProxyDelayCache(cacheKey); ok {
		h.respondJSON(c, http.StatusOK, gin.H{
			"name":      name,
			"delay":     cachedDelay,
			"url":       testURL,
			"timeout":   timeoutInt,
			"from_cache": true,
		})
		return
	}

	delay, err := h.mihomoManager.GetProxyDelay(name, testURL, timeoutInt)
	if err != nil {
		h.mihomoUnavailable(c, err)
		return
	}

	h.setProxyDelayCache(cacheKey, delay)

	h.respondJSON(c, http.StatusOK, gin.H{
		"name":      name,
		"delay":     delay,
		"url":       testURL,
		"timeout":   timeoutInt,
		"from_cache": false,
	})
}

// SwitchProxy switches proxy in a group
func (h *Handler) SwitchProxy(c *gin.Context) {
	group := c.Param("group")

	var req struct {
		Proxy string `json:"proxy" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if err := h.mihomoManager.SwitchProxy(group, req.Proxy); err != nil {
		h.mihomoUnavailable(c, err)
		return
	}

	h.auditLog(c, "switch_proxy", "proxy", "Switched group "+group+" to "+req.Proxy)
	h.respondSuccess(c, "Proxy switched successfully", gin.H{
		"group": group,
		"proxy": req.Proxy,
	})
}

// GetProxyGroups returns proxy groups
func (h *Handler) GetProxyGroups(c *gin.Context) {
	proxies, err := h.mihomoManager.GetProxies()
	if err != nil {
		h.mihomoUnavailable(c, err)
		return
	}

	result := make([]gin.H, 0)
	if raw, ok := proxies["proxies"].(map[string]interface{}); ok {
		for name, item := range raw {
			proxyMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			all, hasAll := proxyMap["all"]
			if !hasAll {
				continue
			}

			group := gin.H{
				"name": name,
				"type": proxyMap["type"],
			}
			if proxiesList, ok := all.([]interface{}); ok {
				group["proxies"] = proxiesList
			}
			if now, ok := proxyMap["now"]; ok {
				group["now"] = now
			}
			result = append(result, group)
		}
	}

	h.respondJSON(c, http.StatusOK, gin.H{"groups": result})
}

// GetConnections returns all active connections from mihomo
func (h *Handler) GetConnections(c *gin.Context) {
	conns, err := h.mihomoManager.GetConnections()
	if err != nil {
		h.mihomoUnavailable(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, conns)
}

// CloseConnection closes a specific connection by ID
func (h *Handler) CloseConnection(c *gin.Context) {
	id := c.Param("id")
	if err := h.mihomoManager.CloseConnection(id); err != nil {
		h.mihomoUnavailable(c, err)
		return
	}
	h.respondSuccess(c, "Connection closed", nil)
}

// CloseAllConnections closes all active connections
func (h *Handler) CloseAllConnections(c *gin.Context) {
	if err := h.mihomoManager.CloseAllConnections(); err != nil {
		h.mihomoUnavailable(c, err)
		return
	}
	h.respondSuccess(c, "All connections closed", nil)
}

// GetRules returns active rules
func (h *Handler) GetRules(c *gin.Context) {
	rules, err := h.mihomoManager.GetRules()
	if err != nil {
		h.mihomoUnavailable(c, err)
		return
	}

	h.respondJSON(c, http.StatusOK, rules)
}

// GetTraffic returns traffic statistics
func (h *Handler) GetTraffic(c *gin.Context) {
	traffic, err := h.mihomoManager.GetTraffic()
	if err != nil {
		h.mihomoUnavailable(c, err)
		return
	}

	h.respondJSON(c, http.StatusOK, traffic)
}

// GetMemory returns memory usage
func (h *Handler) GetMemory(c *gin.Context) {
	memory, err := h.mihomoManager.GetMemory()
	if err != nil {
		h.mihomoUnavailable(c, err)
		return
	}

	h.respondJSON(c, http.StatusOK, memory)
}

// MihomoControl controls mihomo process
func (h *Handler) MihomoControl(c *gin.Context) {
	action := c.Param("action")
	configPath := h.defaultRuntimeConfigPath()

	if runtimeState, err := h.runtimeStore.Get(); err == nil && runtimeState.ConfigPath != "" {
		configPath = runtimeState.ConfigPath
	}

	switch action {
	case "start":
		if err := h.mihomoManager.Start(configPath); err != nil {
			h.upsertRuntime("error", 0, configPath)
			h.respondAndAuditFailure(c, http.StatusInternalServerError, "mihomo_start", "mihomo", err)
			return
		}
		status, pid, runtimePath := h.runtimeStatusFromManager(configPath)
		h.upsertRuntime(status, pid, runtimePath)
		h.auditLog(c, "mihomo_start", "mihomo", "Started mihomo")
		h.respondSuccess(c, "Mihomo started successfully", nil)
	case "stop":
		if err := h.mihomoManager.Stop(); err != nil {
			h.upsertRuntime("error", 0, configPath)
			h.respondAndAuditFailure(c, http.StatusInternalServerError, "mihomo_stop", "mihomo", err)
			return
		}
		status, pid, runtimePath := h.runtimeStatusFromManager(configPath)
		h.upsertRuntime(status, pid, runtimePath)
		h.auditLog(c, "mihomo_stop", "mihomo", "Stopped mihomo")
		h.respondSuccess(c, "Mihomo stopped successfully", nil)
	case "restart":
		if err := h.mihomoManager.Restart(configPath); err != nil {
			h.upsertRuntime("error", 0, configPath)
			h.respondAndAuditFailure(c, http.StatusInternalServerError, "mihomo_restart", "mihomo", err)
			return
		}
		status, pid, runtimePath := h.runtimeStatusFromManager(configPath)
		h.upsertRuntime(status, pid, runtimePath)
		h.auditLog(c, "mihomo_restart", "mihomo", "Restarted mihomo")
		h.respondSuccess(c, "Mihomo restarted successfully", nil)
	case "status":
		running := h.mihomoManager.IsRunning()
		pid := h.mihomoManager.GetPID()
		status := "stopped"
		if running {
			status = "running"
		}
		h.respondSuccess(c, "Mihomo status", gin.H{
			"running": running,
			"status":  status,
			"pid":     pid,
		})
	default:
		h.respondError(c, http.StatusBadRequest, "Invalid action: "+action)
	}
}

// MihomoVersionList returns a list of available mihomo release versions from GitHub.
func (h *Handler) MihomoVersionList(c *gin.Context) {
	if h.mihomoUpdater == nil {
		h.respondError(c, http.StatusServiceUnavailable, "Updater not configured")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	versions, err := h.mihomoUpdater.FetchVersionList(ctx, 30)
	if err != nil {
		h.respondError(c, http.StatusBadGateway, "Failed to fetch version list: "+err.Error())
		return
	}
	h.respondJSON(c, http.StatusOK, gin.H{"versions": versions})
}

// MihomoInstallStatus returns a consolidated installation status: binary existence,
// current version, latest upstream version, update availability, and process state.
func (h *Handler) MihomoInstallStatus(c *gin.Context) {
	binaryPath := h.mihomoManager.GetBinaryPath()

	// Check if binary exists.
	_, statErr := os.Stat(binaryPath)
	installed := statErr == nil

	currentVersion := ""
	if installed {
		currentVersion, _ = core.DetectBinaryVersion(binaryPath)
	}

	// Fetch latest version with a short timeout; non-fatal on error.
	latestVersion := ""
	needsUpdate := false
	if h.mihomoUpdater != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		if v, err := h.mihomoUpdater.FetchLatestVersion(ctx); err == nil {
			latestVersion = v
			if !installed || currentVersion == "" {
				needsUpdate = true
			} else {
				needsUpdate = h.mihomoUpdater.NeedsUpdate(currentVersion, latestVersion)
			}
		}
	}

	h.respondJSON(c, http.StatusOK, gin.H{
		"installed":       installed,
		"current_version": currentVersion,
		"latest_version":  latestVersion,
		"needs_update":    needsUpdate,
		"binary_path":     binaryPath,
		"is_running":      h.mihomoManager.IsRunning(),
		"pid":             h.mihomoManager.GetPID(),
	})
}

// MihomoVersion returns the current mihomo binary version by executing the binary.
func (h *Handler) MihomoVersion(c *gin.Context) {
	binaryPath := h.mihomoManager.GetBinaryPath()
	version, err := core.DetectBinaryVersion(binaryPath)
	if err != nil {
		h.respondJSON(c, http.StatusOK, gin.H{
			"version":      "",
			"installed":    false,
			"binary_path":  binaryPath,
		})
		return
	}
	h.respondJSON(c, http.StatusOK, gin.H{
		"version":     version,
		"installed":   true,
		"binary_path": binaryPath,
	})
}

// MihomoReleases returns the latest available mihomo version from GitHub.
func (h *Handler) MihomoReleases(c *gin.Context) {
	if h.mihomoUpdater == nil {
		h.respondError(c, http.StatusServiceUnavailable, "Updater not configured")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	latestVersion, err := h.mihomoUpdater.FetchLatestVersion(ctx)
	if err != nil {
		h.respondError(c, http.StatusBadGateway, "Failed to fetch release info: "+err.Error())
		return
	}
	h.respondJSON(c, http.StatusOK, gin.H{"latest_version": latestVersion})
}

// MihomoUpdate triggers a manual mihomo install or update.
// Body (optional JSON): {"version": "v1.2.3", "force": true}
// - version: specific version to install; empty = latest
// - force: if true, install even when current == latest
func (h *Handler) MihomoUpdate(c *gin.Context) {
	if h.mihomoUpdater == nil {
		h.respondError(c, http.StatusServiceUnavailable, "Updater not configured")
		return
	}

	var req struct {
		Version string `json:"version"`
		Force   bool   `json:"force"`
	}
	_ = c.ShouldBindJSON(&req)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Minute)
	defer cancel()

	binaryPath := h.mihomoManager.GetBinaryPath()

	var result core.UpdateResult
	var err error

	if req.Force || req.Version != "" {
		// Forced install or specific version requested.
		result, err = h.mihomoUpdater.Install(ctx, binaryPath, req.Version)
	} else {
		result, err = h.mihomoUpdater.CheckAndUpdate(ctx, binaryPath)
	}

	if err != nil {
		h.respondAndAuditFailure(c, http.StatusInternalServerError, "mihomo_update", "mihomo", err)
		return
	}

	if !result.Updated {
		h.respondSuccess(c, "Already up to date", gin.H{
			"updated":         false,
			"current_version": result.OldVersion,
			"latest_version":  result.NewVersion,
		})
		return
	}

	// Reload mihomo if it is currently running.
	if err := h.mihomoManager.ApplyUpdatedBinary(result.BinaryPath); err != nil {
		if rbErr := h.mihomoUpdater.Rollback(result); rbErr != nil {
			h.auditLog(c, "mihomo_update_rollback_failed", "mihomo",
				fmt.Sprintf("Apply failed: %v; rollback failed: %v", err, rbErr))
			h.respondError(c, http.StatusInternalServerError,
				fmt.Sprintf("Update apply failed and rollback also failed: %v", err))
			return
		}
		h.auditLog(c, "mihomo_update_rolled_back", "mihomo",
			fmt.Sprintf("Apply failed (%v), rolled back to %s", err, result.OldVersion))
		h.respondError(c, http.StatusInternalServerError,
			fmt.Sprintf("Update apply failed, rolled back to %s: %v", result.OldVersion, err))
		return
	}

	action := "mihomo_update_applied"
	if result.OldVersion == "" {
		action = "mihomo_installed"
	}
	h.auditLog(c, action, "mihomo",
		fmt.Sprintf("Installed %s (was: %q)", result.NewVersion, result.OldVersion))
	h.respondSuccess(c, "Mihomo installed successfully", gin.H{
		"updated":         true,
		"old_version":     result.OldVersion,
		"new_version":     result.NewVersion,
		"downloaded_from": result.DownloadedFrom,
	})
}
