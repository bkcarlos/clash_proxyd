package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/clash-proxyd/proxyd/internal/logx"
	"github.com/clash-proxyd/proxyd/internal/store"
	"github.com/clash-proxyd/proxyd/internal/types"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// Scheduler manages scheduled tasks
type Scheduler struct {
	cron        *cron.Cron
	sourceStore *store.SourceStore
	auditStore  *store.AuditStore
	mu          sync.RWMutex
	jobs        map[string]cron.EntryID
}

// JobFunc represents a scheduled job function
type JobFunc func() error

// NewScheduler creates a new scheduler
func NewScheduler(sourceStore *store.SourceStore, auditStore *store.AuditStore) *Scheduler {
	return &Scheduler{
		cron:        cron.New(cron.WithSeconds()),
		sourceStore: sourceStore,
		auditStore:  auditStore,
		jobs:        make(map[string]cron.EntryID),
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	s.cron.Start()
	logx.Info("Scheduler started")
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.cron.Stop()
	logx.Info("Scheduler stopped")
}

// AddJob adds a scheduled job
func (s *Scheduler) AddJob(name string, schedule string, job JobFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[name]; exists {
		return fmt.Errorf("job already exists: %s", name)
	}

	wrappedJob := func() {
		if err := job(); err != nil {
			logx.Warn("Scheduled job failed",
				zap.String("job", name),
				zap.Error(err))
			s.writeAudit("alert_source_update_failed", "alert", fmt.Sprintf("job=%s failed: %v", name, err))
		} else {
			logx.Info("Scheduled job completed",
				zap.String("job", name))
			s.writeAudit("source_update_ok", "source", fmt.Sprintf("job=%s completed", name))
		}
	}

	id, err := s.cron.AddFunc(schedule, wrappedJob)
	if err != nil {
		return fmt.Errorf("failed to add job: %w", err)
	}

	s.jobs[name] = id
	logx.Info("Job added to scheduler",
		zap.String("job", name),
		zap.String("schedule", schedule))

	return nil
}

// RemoveJob removes a scheduled job
func (s *Scheduler) RemoveJob(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, exists := s.jobs[name]
	if !exists {
		return fmt.Errorf("job not found: %s", name)
	}

	s.cron.Remove(id)
	delete(s.jobs, name)

	logx.Info("Job removed from scheduler", zap.String("job", name))

	return nil
}

// UpdateJob updates a scheduled job
func (s *Scheduler) UpdateJob(name string, schedule string, job JobFunc) error {
	if err := s.RemoveJob(name); err != nil {
		return err
	}
	return s.AddJob(name, schedule, job)
}

// StartSourceUpdates starts update jobs for all enabled sources
func (s *Scheduler) StartSourceUpdates(updateFunc func(sourceID int) error) error {
	sources, err := s.sourceStore.GetEnabled()
	if err != nil {
		return fmt.Errorf("failed to get sources: %w", err)
	}

	for _, source := range sources {
		if err := s.AddSourceJob(source.ID, source.UpdateCron, updateFunc); err != nil {
			logx.Error("Failed to add source job",
				zap.Int("source_id", source.ID),
				zap.Error(err))
		}
	}

	return nil
}

// AddSourceJob adds an update job for a source
func (s *Scheduler) AddSourceJob(sourceID int, cronExpr string, updateFunc func(sourceID int) error) error {
	name := fmt.Sprintf("source-%d", sourceID)

	// Default to hourly if no cron expression
	schedule := cronExpr
	if schedule == "" {
		schedule = "0 0 * * * *" // Every hour
	}

	job := func() error {
		return updateFunc(sourceID)
	}

	return s.AddJob(name, schedule, job)
}

// RemoveSourceJob removes an update job for a source
func (s *Scheduler) RemoveSourceJob(sourceID int) error {
	name := fmt.Sprintf("source-%d", sourceID)
	return s.RemoveJob(name)
}

// RunJob runs a job immediately
func (s *Scheduler) RunJob(name string) error {
	s.mu.RLock()
	id, exists := s.jobs[name]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("job not found: %s", name)
	}

	// Trigger the job immediately
	s.cron.Entry(id).Job.Run()

	return nil
}

// GetJobs returns all scheduled jobs
func (s *Scheduler) GetJobs() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make(map[string]string)
	for name, id := range s.jobs {
		entry := s.cron.Entry(id)
		jobs[name] = fmt.Sprintf("next: %s", entry.Next.Format(time.RFC3339))
	}

	return jobs
}

// Running returns whether the scheduler is running
func (s *Scheduler) Running() bool {
	return s.cron != nil
}

func (s *Scheduler) writeAudit(action, resource, details string) {
	if s.auditStore == nil {
		return
	}
	if err := s.auditStore.Create(&types.AuditLog{
		User:      "system",
		Action:    action,
		Resource:  resource,
		Details:   details,
		IPAddress: "127.0.0.1",
	}); err != nil {
		logx.Error("Failed to write scheduler audit", zap.Error(err))
	}
}

// RunPeriodic runs a task at a fixed interval until context cancellation.
func (s *Scheduler) RunPeriodic(ctx context.Context, interval time.Duration, task func() error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := task(); err != nil {
				logx.Error("Periodic task failed", zap.Error(err))
			}
		}
	}
}

// RunOnce runs a task once after a delay
func (s *Scheduler) RunOnce(delay time.Duration, task func()) {
	time.AfterFunc(delay, func() {
		defer func() {
			if r := recover(); r != nil {
				logx.Error("Delayed task panicked", zap.Any("panic", r))
			}
		}()
		task()
	})
}
