package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/clash-proxyd/proxyd/internal/parser"
	"github.com/clash-proxyd/proxyd/internal/source"
	"github.com/clash-proxyd/proxyd/internal/types"
	"github.com/gin-gonic/gin"
)

// GenerateConfigRequest represents config generation request
type GenerateConfigRequest struct {
	SourceIDs []int `json:"source_ids"`
}

// ApplyConfigRequest represents config apply request
type ApplyConfigRequest struct {
	Config string `json:"config"`
	Path   string `json:"path,omitempty"`
}

func (h *Handler) loadSourceConfigs(sourceIDs []int) ([]types.Source, []map[string]interface{}, error) {
	sources := make([]types.Source, 0, len(sourceIDs))
	configs := make([]map[string]interface{}, 0, len(sourceIDs))
	yamlParser := parser.NewParser()
	fetcher := source.NewFetcher("clash-proxyd", 30, 3, 5)

	for _, id := range sourceIDs {
		src, err := h.sourceStore.GetByID(id)
		if err != nil {
			return nil, nil, fmt.Errorf("source not found: ID %d", id)
		}
		if !src.Enabled {
			return nil, nil, fmt.Errorf("source is disabled: %s", src.Name)
		}

		var content []byte
		if src.Type == "http" {
			content, err = fetcher.Fetch(src.URL)
		} else {
			content, err = fetcher.FetchFromFile(src.Path)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("failed to fetch source %s: %w", src.Name, err)
		}

		cfg, err := yamlParser.Parse(content)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse source %s: %w", src.Name, err)
		}

		_ = h.sourceStore.UpdateLastFetch(src.ID)
		sources = append(sources, *src)
		configs = append(configs, cfg)
	}

	return sources, configs, nil
}

func (h *Handler) nextRevisionVersion() string {
	latest, err := h.revisionStore.GetLatest()
	if err != nil || latest == nil {
		return "1"
	}
	if n, convErr := strconv.Atoi(latest.Version); convErr == nil {
		return strconv.Itoa(n + 1)
	}
	return latest.Version + "-next"
}

func (h *Handler) ensurePathInConfigDir(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("path is required")
	}
	clean := filepath.Clean(path)
	absPath, err := filepath.Abs(clean)
	if err != nil {
		return "", fmt.Errorf("invalid path")
	}
	configDirAbs, err := filepath.Abs(filepath.Clean(h.mihomoConfigDir))
	if err != nil {
		return "", fmt.Errorf("invalid config dir")
	}
	if absPath != configDirAbs && !strings.HasPrefix(absPath, configDirAbs+string(os.PathSeparator)) {
		return "", fmt.Errorf("path must be within mihomo config dir")
	}
	return absPath, nil
}

func (h *Handler) saveAndApplyConfig(c *gin.Context, yamlStr string, targetPath string, action string) error {
	absPath, err := h.ensurePathInConfigDir(targetPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	if err := os.WriteFile(absPath, []byte(yamlStr), 0644); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	if h.mihomoManager.IsRunning() {
		if err := h.mihomoManager.Restart(absPath); err != nil {
			h.upsertRuntime("error", 0, absPath)
			h.auditLog(c, action+"_failed", "config", "Reload failed with config "+absPath+": "+err.Error())
			return fmt.Errorf("failed to reload mihomo with new config: %w", err)
		}
		status, pid, runtimePath := h.runtimeStatusFromManager(absPath)
		h.upsertRuntime(status, pid, runtimePath)
	} else {
		h.upsertRuntime("stopped", 0, absPath)
	}

	h.auditLog(c, action, "config", "Applied config to: "+absPath)
	return nil
}

// GenerateConfig generates mihomo configuration from sources
func (h *Handler) GenerateConfig(c *gin.Context) {
	var req GenerateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if len(req.SourceIDs) == 0 {
		h.respondError(c, http.StatusBadRequest, "At least one source ID is required")
		return
	}

	_, configs, err := h.loadSourceConfigs(req.SourceIDs)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	rendered, err := h.renderer.Render(configs)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, "Failed to render config")
		return
	}

	yamlParser := parser.NewParser()
	yamlStr, err := yamlParser.ToYAMLString(rendered)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, "Failed to generate YAML")
		return
	}

	hash := source.Hash([]byte(yamlStr))
	revision := &types.Revision{
		Version:    h.nextRevisionVersion(),
		Content:    yamlStr,
		SourceHash: hash,
		CreatedBy:  h.getUser(c),
	}
	if err := h.revisionStore.Create(revision); err != nil {
		h.respondError(c, http.StatusInternalServerError, "Failed to save revision")
		return
	}

	h.auditLog(c, "generate_config", "config", "Generated config from sources")
	h.respondJSON(c, http.StatusOK, gin.H{
		"config":   yamlStr,
		"hash":     hash,
		"revision": revision,
	})
}

// GetConfig returns the current mihomo configuration
func (h *Handler) GetConfig(c *gin.Context) {
	path := h.defaultRuntimeConfigPath()
	if runtimeState, err := h.runtimeStore.Get(); err == nil && runtimeState.ConfigPath != "" {
		path = runtimeState.ConfigPath
	}

	content, err := os.ReadFile(path)
	if err != nil {
		h.respondError(c, http.StatusNotFound, "Runtime config not found")
		return
	}

	h.respondJSON(c, http.StatusOK, gin.H{
		"config": string(content),
		"path":   path,
	})
}

// SaveConfig saves configuration to a file
func (h *Handler) SaveConfig(c *gin.Context) {
	var req struct {
		Config string `json:"config" binding:"required"`
		Path   string `json:"path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if _, err := h.ensurePathInConfigDir(req.Path); err != nil {
		h.respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	hash := source.Hash([]byte(req.Config))
	revision := &types.Revision{
		Version:    h.nextRevisionVersion(),
		Content:    req.Config,
		SourceHash: hash,
		CreatedBy:  h.getUser(c),
	}
	if err := h.revisionStore.Create(revision); err != nil {
		h.respondError(c, http.StatusInternalServerError, "Failed to create revision")
		return
	}

	if err := h.saveAndApplyConfig(c, req.Config, req.Path, "save_config"); err != nil {
		h.respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondSuccess(c, "Configuration saved successfully", gin.H{
		"revision": revision,
		"hash":     hash,
	})
}

// ApplyConfig applies config and reloads runtime
func (h *Handler) ApplyConfig(c *gin.Context) {
	var req ApplyConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	configContent := req.Config
	if strings.TrimSpace(configContent) == "" {
		latest, err := h.revisionStore.GetLatest()
		if err != nil || latest == nil {
			h.respondError(c, http.StatusBadRequest, "No config content provided and no latest revision")
			return
		}
		configContent = latest.Content
	}

	path := req.Path
	if strings.TrimSpace(path) == "" {
		path = h.defaultRuntimeConfigPath()
	}

	if err := h.saveAndApplyConfig(c, configContent, path, "apply_config"); err != nil {
		h.respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondSuccess(c, "Configuration applied successfully", nil)
}

// ListRevisions returns configuration revisions
func (h *Handler) ListRevisions(c *gin.Context) {
	limit := 50
	if l, err := strconv.Atoi(c.DefaultQuery("limit", "50")); err == nil && l > 0 {
		limit = l
	}

	revisions, err := h.revisionStore.List(limit)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, "Failed to get revisions")
		return
	}

	h.respondJSON(c, http.StatusOK, revisions)
}

// GetRevision returns a specific revision
func (h *Handler) GetRevision(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid revision ID")
		return
	}

	revision, err := h.revisionStore.GetByID(id)
	if err != nil {
		h.respondError(c, http.StatusNotFound, "Revision not found")
		return
	}

	h.respondJSON(c, http.StatusOK, revision)
}

// RollbackRevision rolls back to a revision and applies it immediately
func (h *Handler) RollbackRevision(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid revision ID")
		return
	}

	revision, err := h.revisionStore.GetByID(id)
	if err != nil {
		h.respondError(c, http.StatusNotFound, "Revision not found")
		return
	}

	path := h.defaultRuntimeConfigPath()
	if runtimeState, getErr := h.runtimeStore.Get(); getErr == nil && runtimeState.ConfigPath != "" {
		path = runtimeState.ConfigPath
	}

	if err := h.saveAndApplyConfig(c, revision.Content, path, "rollback_revision"); err != nil {
		h.respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondSuccess(c, "Revision rolled back successfully", gin.H{
		"revision": revision,
		"path":     path,
	})
}

// DeleteRevision deletes a specific revision
func (h *Handler) DeleteRevision(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid revision ID")
		return
	}

	if err := h.revisionStore.Delete(id); err != nil {
		h.respondError(c, http.StatusInternalServerError, "Failed to delete revision")
		return
	}

	h.auditLog(c, "delete_revision", "revision", "Deleted revision: "+idStr)
	h.respondSuccess(c, "Revision deleted successfully", nil)
}
