package api

import (
	"net/http"
	"os"
	"strconv"

	"github.com/clash-proxyd/proxyd/internal/source"
	"github.com/clash-proxyd/proxyd/internal/types"
	"github.com/gin-gonic/gin"
)

func (h *Handler) syncSourceSchedule(source *types.Source) {
	if h.scheduler == nil {
		return
	}
	if source.Enabled {
		_ = h.scheduler.AddSourceJob(source.ID, source.UpdateCron, h.updateSourceNow)
		return
	}
	_ = h.scheduler.RemoveSourceJob(source.ID)
}

func (h *Handler) updateSourceNow(sourceID int) error {
	src, err := h.sourceStore.GetByID(sourceID)
	if err != nil {
		return err
	}

	fetcher := source.NewFetcher("clash-proxyd", 30, 3, 5)
	if src.Type == "http" {
		_, err = fetcher.Fetch(src.URL)
	} else {
		_, err = fetcher.FetchFromFile(src.Path)
	}
	if err != nil {
		return err
	}

	return h.sourceStore.UpdateLastFetch(sourceID)
}

// ListSources returns all sources
func (h *Handler) ListSources(c *gin.Context) {
	sources, err := h.sourceStore.List()
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, "Failed to get sources")
		return
	}

	h.respondJSON(c, http.StatusOK, sources)
}

// GetSource returns a source by ID
func (h *Handler) GetSource(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid source ID")
		return
	}

	source, err := h.sourceStore.GetByID(id)
	if err != nil {
		h.respondError(c, http.StatusNotFound, "Source not found")
		return
	}

	h.respondJSON(c, http.StatusOK, source)
}

// CreateSourceRequest represents create source request
type CreateSourceRequest struct {
	Name           string `json:"name" binding:"required"`
	Type           string `json:"type" binding:"required,oneof=http file local"`
	URL            string `json:"url,omitempty"`
	Path           string `json:"path,omitempty"`
	UpdateInterval int    `json:"update_interval"`
	UpdateCron     string `json:"update_cron,omitempty"`
	Enabled        bool   `json:"enabled"`
	Priority       int    `json:"priority"`
	ConfigOverride string `json:"config_override,omitempty"`
}

// CreateSource creates a new source
func (h *Handler) CreateSource(c *gin.Context) {
	var req CreateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Validate URL or path based on type
	if req.Type == "http" && req.URL == "" {
		h.respondError(c, http.StatusBadRequest, "URL is required for HTTP sources")
		return
	}
	if (req.Type == "file" || req.Type == "local") && req.Path == "" {
		h.respondError(c, http.StatusBadRequest, "Path is required for file sources")
		return
	}

	source := &types.Source{
		Name:           req.Name,
		Type:           req.Type,
		URL:            req.URL,
		Path:           req.Path,
		UpdateInterval: req.UpdateInterval,
		UpdateCron:     req.UpdateCron,
		Enabled:        req.Enabled,
		Priority:       req.Priority,
		ConfigOverride: req.ConfigOverride,
	}

	// Set default interval
	if source.UpdateInterval == 0 {
		source.UpdateInterval = 3600 // 1 hour
	}

	if err := h.sourceStore.Create(source); err != nil {
		h.respondError(c, http.StatusInternalServerError, "Failed to create source")
		return
	}

	h.syncSourceSchedule(source)
	h.auditLog(c, "create_source", "source", "Created source: "+source.Name)
	h.respondJSON(c, http.StatusCreated, source)
}

// UpdateSource updates a source
func (h *Handler) UpdateSource(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid source ID")
		return
	}

	var req CreateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	source, err := h.sourceStore.GetByID(id)
	if err != nil {
		h.respondError(c, http.StatusNotFound, "Source not found")
		return
	}

	// Update fields
	source.Name = req.Name
	source.Type = req.Type
	source.URL = req.URL
	source.Path = req.Path
	source.UpdateInterval = req.UpdateInterval
	source.UpdateCron = req.UpdateCron
	source.Enabled = req.Enabled
	source.Priority = req.Priority
	source.ConfigOverride = req.ConfigOverride

	if err := h.sourceStore.Update(source); err != nil {
		h.respondError(c, http.StatusInternalServerError, "Failed to update source")
		return
	}

	h.syncSourceSchedule(source)
	h.auditLog(c, "update_source", "source", "Updated source: "+source.Name)
	h.respondJSON(c, http.StatusOK, source)
}

// DeleteSource deletes a source
func (h *Handler) DeleteSource(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid source ID")
		return
	}

	source, err := h.sourceStore.GetByID(id)
	if err != nil {
		h.respondError(c, http.StatusNotFound, "Source not found")
		return
	}

	if err := h.sourceStore.Delete(id); err != nil {
		h.respondError(c, http.StatusInternalServerError, "Failed to delete source")
		return
	}

	if h.scheduler != nil {
		_ = h.scheduler.RemoveSourceJob(id)
	}

	h.auditLog(c, "delete_source", "source", "Deleted source: "+source.Name)
	h.respondSuccess(c, "Source deleted successfully", nil)
}

// TestSource tests a source connection
func (h *Handler) TestSource(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid source ID")
		return
	}

	src, err := h.sourceStore.GetByID(id)
	if err != nil {
		h.respondError(c, http.StatusNotFound, "Source not found")
		return
	}

	// Test the source
	var result types.TestResult
	if src.Type == "http" {
		success, latency, err := source.TestSubscription(src.URL, 10)
		result.Success = success
		result.Latency = latency
		if err != nil {
			result.Error = err.Error()
		}
	} else if src.Type == "file" || src.Type == "local" {
		// Test file access
		if _, err := os.Stat(src.Path); err == nil {
			result.Success = true
		} else {
			result.Success = false
			result.Error = "File not found"
		}
	}

	h.respondJSON(c, http.StatusOK, result)
}

// FetchSource fetches a source immediately
func (h *Handler) FetchSource(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "Invalid source ID")
		return
	}

	src, err := h.sourceStore.GetByID(id)
	if err != nil {
		h.respondError(c, http.StatusNotFound, "Source not found")
		return
	}

	// Fetch source content
	var content []byte
	if src.Type == "http" {
		fetcher := source.NewFetcher("clash-proxyd", 30, 3, 5)
		content, err = fetcher.Fetch(src.URL)
	} else {
		fetcher := source.NewFetcher("clash-proxyd", 30, 3, 5)
		content, err = fetcher.FetchFromFile(src.Path)
	}

	if err != nil {
		h.respondError(c, http.StatusInternalServerError, "Failed to fetch source: "+err.Error())
		return
	}

	// Update last fetch time
	h.sourceStore.UpdateLastFetch(id)

	h.auditLog(c, "fetch_source", "source", "Fetched source: "+src.Name)
	h.respondJSON(c, http.StatusOK, gin.H{
		"content": string(content),
		"size":    len(content),
		"hash":    source.Hash(content),
	})
}
