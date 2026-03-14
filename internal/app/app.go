package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/clash-proxyd/proxyd/internal/api"
	"github.com/clash-proxyd/proxyd/internal/auth"
	"github.com/clash-proxyd/proxyd/internal/core"
	"github.com/clash-proxyd/proxyd/internal/health"
	"github.com/clash-proxyd/proxyd/internal/logx"
	"github.com/clash-proxyd/proxyd/internal/renderer"
	"github.com/clash-proxyd/proxyd/internal/scheduler"
	"github.com/clash-proxyd/proxyd/internal/source"
	"github.com/clash-proxyd/proxyd/internal/store"
	"github.com/clash-proxyd/proxyd/internal/types"
	"github.com/clash-proxyd/proxyd/pkg/config"

	"go.uber.org/zap"
)

// App represents the application
type App struct {
	cfg           *config.Config
	db            *store.DB
	handler       *api.Handler
	authManager   *auth.Manager
	sourceStore   *store.SourceStore
	auditStore    *store.AuditStore
	mihomoManager *core.Manager
	updater       *core.Updater
	scheduler     *scheduler.Scheduler
	healthChecker *health.Checker
	server        *http.Server
}

// New creates a new application
func New(cfg *config.Config) (*App, error) {
	// Initialize logger
	if err := logx.Init(&logx.Config{
		Level:      cfg.Logging.Level,
		Output:     cfg.Logging.Output,
		FilePath:   cfg.Logging.FilePath,
		MaxSize:    cfg.Logging.MaxSize,
		MaxBackups: cfg.Logging.MaxBackups,
		MaxAge:     cfg.Logging.MaxAge,
		Compress:   cfg.Logging.Compress,
	}); err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	logx.Info("Starting proxyd",
		zap.String("version", "1.0.0"),
		zap.String("log_level", cfg.Logging.Level))

	// Initialize database
	db, err := store.NewDB(cfg.Database.Path, cfg.Database.ForeignKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	logx.Info("Database initialized", zap.String("path", cfg.Database.Path))

	// Initialize stores
	sourceStore := store.NewSourceStore(db)
	settingStore := store.NewSettingStore(db)
	revisionStore := store.NewRevisionStore(db)
	runtimeStore := store.NewRuntimeStore(db)
	auditStore := store.NewAuditStore(db)

	// Initialize auth manager
	authManager := auth.NewManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.SessionTimeout,
		settingStore,
	)

	// Initialize mihomo manager
	mihomoManager := core.NewManager(
		cfg.Mihomo.BinaryPath,
		cfg.Mihomo.ConfigDir,
		cfg.Mihomo.LogDir,
		cfg.Mihomo.APIPort,
		cfg.Mihomo.APISecret,
	)

	// Initialize updater
	updater := core.NewUpdater(core.UpdaterConfig{
		Enabled:       cfg.Mihomo.AutoUpdateEnabled,
		CheckOnStart:  cfg.Mihomo.AutoUpdateCheckOnStart,
		ReleaseAPI:    cfg.Mihomo.ReleaseAPI,
		DownloadDir:   cfg.Mihomo.DownloadDir,
		TargetVersion: cfg.Mihomo.TargetVersion,
	})

	// Initialize scheduler
	sched := scheduler.NewScheduler(sourceStore, auditStore)

	// Initialize runtime renderer
	runtimeRenderer := renderer.NewRenderer(&cfg.Policy)

	// Initialize health checker
	healthChecker := health.NewChecker()
	healthChecker.Register("database", health.CheckDatabase(db))
	healthChecker.Register("mihomo", health.CheckMihomo(cfg.Mihomo.BinaryPath))
	healthChecker.Register("db_path_writable", health.CheckPathWritable(cfg.Database.Path, true))
	healthChecker.Register("mihomo_config_dir_writable", health.CheckPathWritable(cfg.Mihomo.ConfigDir, false))

	// Initialize API handler
	handler := api.NewHandler(
		authManager,
		sourceStore,
		settingStore,
		revisionStore,
		runtimeStore,
		auditStore,
		mihomoManager,
		updater,
		runtimeRenderer,
		sched,
		cfg.Mihomo.ConfigDir,
		cfg.Mihomo.APIPort,
		cfg.Logging.FilePath,
		cfg.Mihomo.LogDir,
	)

	return &App{
		cfg:           cfg,
		db:            db,
		handler:       handler,
		authManager:   authManager,
		sourceStore:   sourceStore,
		auditStore:    auditStore,
		mihomoManager: mihomoManager,
		updater:       updater,
		scheduler:     sched,
		healthChecker: healthChecker,
	}, nil
}

// InitDB initializes the database schema
func (a *App) InitDB() error {
	schemaSQL, err := os.ReadFile("internal/store/schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}

	if err := a.db.InitSchema(string(schemaSQL)); err != nil {
		return fmt.Errorf("failed to init schema: %w", err)
	}

	logx.Info("Database schema initialized")
	return nil
}

// Start starts the application
func (a *App) Start() error {
	// Perform startup update check
	a.checkAndApplyMihomoUpdate()

	a.logStartupSelfCheck()

	// Perform initial health check
	healthStatus := a.healthChecker.Check()
	logx.Info("Initial health check", zap.String("status", healthStatus.Status), zap.Any("services", healthStatus.Services))

	// Start scheduler if enabled
	if a.cfg.Scheduler.Enabled {
		a.scheduler.Start()
		if err := a.scheduler.StartSourceUpdates(func(sourceID int) error {
			src, getErr := a.sourceStore.GetByID(sourceID)
			if getErr != nil {
				return getErr
			}
			fetcher := source.NewFetcher("clash-proxyd", 30, 3, 5)
			if src.Type == "http" {
				_, getErr = fetcher.Fetch(src.URL)
			} else {
				_, getErr = fetcher.FetchFromFile(src.Path)
			}
			if getErr != nil {
				return getErr
			}
			return a.sourceStore.UpdateLastFetch(sourceID)
		}); err != nil {
			logx.Error("Failed to restore source schedules", zap.Error(err))
		}
		logx.Info("Scheduler started")
	}

	// Setup router
	router := a.handler.SetupRouter(a.authManager, a.cfg.Server.CorsOrigins)

	// Create HTTP server
	a.server = &http.Server{
		Addr:           fmt.Sprintf("%s:%d", a.cfg.Server.Host, a.cfg.Server.Port),
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Start server in goroutine
	go func() {
		logx.Info("Server starting",
			zap.String("address", a.server.Addr))
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logx.Fatal("Server failed", zap.Error(err))
		}
	}()

	logx.Info("Application started successfully")
	return nil
}

func (a *App) checkAndApplyMihomoUpdate() {
	if !a.cfg.Mihomo.AutoUpdateEnabled || !a.cfg.Mihomo.AutoUpdateCheckOnStart {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	binaryPath := a.mihomoManager.GetBinaryPath()
	result, err := a.updater.CheckAndUpdate(ctx, binaryPath)
	if err != nil {
		logx.Error("Mihomo auto-update failed", zap.Error(err))
		a.auditAutoUpdate("mihomo_update_failed", fmt.Sprintf("Auto update failed: %v", err))
		return
	}

	if !result.Updated {
		a.auditAutoUpdate("mihomo_update_skipped", fmt.Sprintf("No update needed. current=%s latest=%s", result.OldVersion, result.NewVersion))
		return
	}

	wasRunning := a.mihomoManager.IsRunning()
	configPath := a.mihomoManager.GetCurrentConfigPath()

	if err := a.mihomoManager.ApplyUpdatedBinary(result.BinaryPath); err != nil {
		logx.Error("Failed to apply updated mihomo binary, rolling back", zap.Error(err))
		auditDetails := fmt.Sprintf("Apply updated binary failed: %v (from=%s to=%s)", err, result.OldVersion, result.NewVersion)

		if rbErr := a.updater.Rollback(result); rbErr != nil {
			logx.Error("Failed to rollback mihomo binary", zap.Error(rbErr))
			a.auditAutoUpdate("mihomo_update_rollback_failed", fmt.Sprintf("%s; rollback failed: %v", auditDetails, rbErr))
			return
		}

		if wasRunning && configPath != "" {
			if restartErr := a.mihomoManager.Start(configPath); restartErr != nil {
				logx.Error("Failed to restart mihomo after rollback", zap.Error(restartErr))
				a.auditAutoUpdate("mihomo_update_rollback_failed", fmt.Sprintf("%s; rollback succeeded but restart failed: %v", auditDetails, restartErr))
				return
			}
		}

		logx.Warn("Mihomo binary rollback completed", zap.String("version", result.OldVersion))
		a.auditAutoUpdate("mihomo_update_rolled_back", fmt.Sprintf("Rollback completed. from=%s to=%s", result.NewVersion, result.OldVersion))
		return
	}

	logx.Info("Mihomo update applied successfully",
		zap.String("from_version", result.OldVersion),
		zap.String("to_version", result.NewVersion),
		zap.String("binary", result.BinaryPath))
	a.auditAutoUpdate("mihomo_update_applied", fmt.Sprintf("Update applied successfully. from=%s to=%s", result.OldVersion, result.NewVersion))
}

func (a *App) auditAutoUpdate(action, details string) {
	if a.auditStore == nil {
		return
	}
	if err := a.auditStore.Create(&types.AuditLog{
		User:      "system",
		Action:    action,
		Resource:  "mihomo",
		Details:   details,
		IPAddress: "127.0.0.1",
	}); err != nil {
		logx.Error("Failed to write auto-update audit log", zap.Error(err))
	}
}

func (a *App) logStartupSelfCheck() {
	binaryCheck := "ok"
	if _, err := os.Stat(a.cfg.Mihomo.BinaryPath); err != nil {
		binaryCheck = "failed: " + err.Error()
	}

	configDirCheck := "ok"
	if err := os.MkdirAll(a.cfg.Mihomo.ConfigDir, 0755); err != nil {
		configDirCheck = "failed: " + err.Error()
	} else if err := health.CheckPathWritable(a.cfg.Mihomo.ConfigDir, false)(); err != nil {
		configDirCheck = "failed: " + err.Error()
	}

	dbPathCheck := "ok"
	if err := health.CheckPathWritable(a.cfg.Database.Path, true)(); err != nil {
		dbPathCheck = "failed: " + err.Error()
	}

	logx.Info("Startup self-check",
		zap.String("mihomo_binary", binaryCheck),
		zap.String("mihomo_config_dir", configDirCheck),
		zap.String("database_path", dbPathCheck),
		zap.String("database_file", filepath.Clean(a.cfg.Database.Path)),
	)

	if a.auditStore != nil {
		details := fmt.Sprintf("mihomo_binary=%s; mihomo_config_dir=%s; database_path=%s", binaryCheck, configDirCheck, dbPathCheck)
		if err := a.auditStore.Create(&types.AuditLog{
			User:      "system",
			Action:    "startup_self_check",
			Resource:  "system",
			Details:   details,
			IPAddress: "127.0.0.1",
		}); err != nil {
			logx.Error("Failed to write startup self-check audit log", zap.Error(err))
		}
	}
}

func (a *App) Stop() error {
	logx.Info("Shutting down application...")

	// Stop mihomo
	if a.mihomoManager.IsRunning() {
		if err := a.mihomoManager.Stop(); err != nil {
			logx.Error("Failed to stop mihomo", zap.Error(err))
		}
	}

	if a.auditStore != nil {
		action := "alert_mihomo_abnormal_stopped"
		details := "Mihomo process is not running"
		if err := a.auditStore.Create(&types.AuditLog{
			User:      "system",
			Action:    action,
			Resource:  "alert",
			Details:   details,
			IPAddress: "127.0.0.1",
		}); err != nil {
			logx.Error("Failed to write shutdown alert", zap.Error(err))
		}
	}

	// Stop scheduler
	a.scheduler.Stop()

	// Shutdown HTTP server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	// Close database
	if err := a.db.Close(); err != nil {
		return fmt.Errorf("database close failed: %w", err)
	}

	// Sync logger
	if err := logx.Sync(); err != nil {
		return fmt.Errorf("logger sync failed: %w", err)
	}

	logx.Info("Application stopped successfully")
	return nil
}

// Wait waits for shutdown signal
func (a *App) Wait() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	if err := a.Stop(); err != nil {
		logx.Error("Shutdown error", zap.Error(err))
		os.Exit(1)
	}
}

// Run starts the application and waits for shutdown
func (a *App) Run() error {
	if err := a.Start(); err != nil {
		return err
	}
	a.Wait()
	return nil
}
