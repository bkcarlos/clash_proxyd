package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/clash-proxyd/proxyd/internal/core"
	"github.com/clash-proxyd/proxyd/internal/logx"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// InstallJob holds the state of a single background install operation.
type InstallJob struct {
	mu         sync.RWMutex
	running    bool
	stage      string
	percent    int
	message    string
	errMsg     string
	startedAt  time.Time
	finishedAt time.Time
	result     *core.UpdateResult
}

func (j *InstallJob) update(stage, message string, percent int) {
	j.mu.Lock()
	j.stage = stage
	j.message = message
	j.percent = percent
	j.mu.Unlock()
}

func (j *InstallJob) snapshot() gin.H {
	j.mu.RLock()
	defer j.mu.RUnlock()
	h := gin.H{
		"running":     j.running,
		"stage":       j.stage,
		"percent":     j.percent,
		"message":     j.message,
		"error":       j.errMsg,
		"started_at":  j.startedAt,
	}
	if !j.finishedAt.IsZero() {
		h["finished_at"] = j.finishedAt
	}
	if j.result != nil {
		h["old_version"] = j.result.OldVersion
		h["new_version"] = j.result.NewVersion
		h["downloaded_from"] = j.result.DownloadedFrom
	}
	return h
}

// jobMu guards currentJob replacement.
var jobMu sync.Mutex

// StartInstallJob starts a background install and returns immediately.
// Only one job may run at a time; returns 409 if already running.
// Body (optional JSON): {"version": "v1.2.3", "force": true}
func (h *Handler) StartInstallJob(c *gin.Context) {
	if h.mihomoUpdater == nil {
		h.respondError(c, http.StatusServiceUnavailable, "Updater not configured")
		return
	}

	var req struct {
		Version string `json:"version"`
		Force   bool   `json:"force"`
	}
	_ = c.ShouldBindJSON(&req)

	jobMu.Lock()
	if h.installJob != nil && h.installJob.running {
		jobMu.Unlock()
		h.respondError(c, http.StatusConflict, "An install job is already running")
		return
	}
	job := &InstallJob{
		running:   true,
		stage:     core.StageFetchRelease,
		message:   "Starting…",
		startedAt: time.Now(),
	}
	h.installJob = job
	jobMu.Unlock()

	binaryPath := h.mihomoManager.GetBinaryPath()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		progress := func(e core.ProgressEvent) {
			job.update(e.Stage, e.Message, e.Percent)
		}

		var result core.UpdateResult
		var err error

		if req.Force || req.Version != "" {
			result, err = h.mihomoUpdater.InstallWithProgress(ctx, binaryPath, req.Version, progress)
		} else {
			// Check-and-update path: fetch latest, compare, install if needed.
			job.update(core.StageFetchRelease, "Fetching release information…", 0)
			result, err = h.mihomoUpdater.CheckAndUpdateWithProgress(ctx, binaryPath, progress)
		}

		job.mu.Lock()
		job.running = false
		job.finishedAt = time.Now()

		if err != nil {
			job.stage = core.StageError
			job.message = err.Error()
			job.errMsg = err.Error()
			job.mu.Unlock()
			logx.Error("Install job failed", zap.Error(err))
			return
		}

		if !result.Updated {
			job.stage = core.StageDone
			job.message = fmt.Sprintf("Already up to date (%s)", result.OldVersion)
			job.percent = 100
			job.result = &result
			job.mu.Unlock()
			return
		}

		job.result = &result
		job.mu.Unlock()

		// Apply binary (restart if running).
		job.update(core.StageRestart, "Applying updated binary…", 100)
		if applyErr := h.mihomoManager.ApplyUpdatedBinary(result.BinaryPath); applyErr != nil {
			logx.Error("Failed to apply updated mihomo binary, rolling back", zap.Error(applyErr))
			_ = h.mihomoUpdater.Rollback(result)
			job.mu.Lock()
			job.stage = core.StageError
			job.message = fmt.Sprintf("Apply failed, rolled back: %v", applyErr)
			job.errMsg = applyErr.Error()
			job.mu.Unlock()
			return
		}

		job.update(core.StageDone, fmt.Sprintf("Installed %s", result.NewVersion), 100)
		logx.Info("Install job completed",
			zap.String("old", result.OldVersion),
			zap.String("new", result.NewVersion))

		if h.auditStore != nil {
			action := "mihomo_update_applied"
			if result.OldVersion == "" {
				action = "mihomo_installed"
			}
			h.auditSystem(action, "mihomo",
				fmt.Sprintf("job: installed %s (was %q)", result.NewVersion, result.OldVersion))
		}
	}()

	h.respondSuccess(c, "Install job started", gin.H{
		"stage":      job.stage,
		"started_at": job.startedAt,
	})
}

// GetInstallProgress returns the current or last install job status.
func (h *Handler) GetInstallProgress(c *gin.Context) {
	jobMu.Lock()
	job := h.installJob
	jobMu.Unlock()

	if job == nil {
		h.respondJSON(c, http.StatusOK, gin.H{
			"running": false,
			"stage":   "",
			"message": "No install job has been started",
		})
		return
	}

	h.respondJSON(c, http.StatusOK, job.snapshot())
}
