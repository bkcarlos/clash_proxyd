package app

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/clash-proxyd/proxyd/internal/api"
	"github.com/clash-proxyd/proxyd/internal/assets"
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
	runtimeStore  *store.RuntimeStore
	mihomoManager *core.Manager
	updater       *core.Updater
	scheduler     *scheduler.Scheduler
	healthChecker *health.Checker
	server        *http.Server
	webServer     *http.Server // non-nil when web UI runs on a dedicated port
	webFS         fs.FS        // non-nil when the embedded web UI should be served
}

// EnableWebUI registers the embedded filesystem to be served at "/".
// Must be called before Start().
func (a *App) EnableWebUI(webFS fs.FS) {
	a.webFS = webFS
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

	// Extract bundled assets (mihomo binary + Country.mmdb) if absent on disk.
	if assets.Bundled() {
		mmdbPath := filepath.Join(cfg.Mihomo.ConfigDir, "Country.mmdb")
		if err := assets.Extract(cfg.Mihomo.BinaryPath, mmdbPath); err != nil {
			logx.Warn("Failed to extract bundled assets", zap.Error(err))
		} else {
			logx.Info("Bundled assets ready",
				zap.String("binary", cfg.Mihomo.BinaryPath),
				zap.String("mmdb", mmdbPath))
		}
	}

	// Initialize database
	db, err := store.NewDB(cfg.Database.Path, cfg.Database.ForeignKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	logx.Info("Database initialized", zap.String("path", cfg.Database.Path))

	// Migrate schema: add content cache columns to sources if not present.
	if err := migrateSourcesTable(db); err != nil {
		return nil, fmt.Errorf("failed to migrate sources table: %w", err)
	}

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
		cfg.Subscription.UserAgent,
		cfg.Subscription.Timeout,
		cfg.Subscription.MaxRetries,
		cfg.Subscription.RetryDelay,
	)

	return &App{
		cfg:           cfg,
		db:            db,
		handler:       handler,
		authManager:   authManager,
		sourceStore:   sourceStore,
		auditStore:    auditStore,
		runtimeStore:  runtimeStore,
		mihomoManager: mihomoManager,
		updater:       updater,
		scheduler:     sched,
		healthChecker: healthChecker,
	}, nil
}

// InitDB re-applies the embedded schema (idempotent). Kept for the -init-db CLI flag.
func (a *App) InitDB() error {
	if err := a.db.InitSchema(); err != nil {
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

	// Auto-restore mihomo if it was running before proxyd stopped
	a.autoStartMihomo()

	// Start scheduler if enabled
	if a.cfg.Scheduler.Enabled {
		a.scheduler.Start()
		if err := a.scheduler.StartSourceUpdates(func(sourceID int) error {
			src, getErr := a.sourceStore.GetByID(sourceID)
			if getErr != nil {
				return getErr
			}
			fetcher := source.NewFetcher(a.cfg.Subscription.UserAgent, a.cfg.Subscription.Timeout, a.cfg.Subscription.MaxRetries, a.cfg.Subscription.RetryDelay)
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

	// When a dedicated web_port is configured, the API server carries no web
	// routes and the web UI gets its own lightweight server.
	webFSForAPI := a.webFS
	if a.webFS != nil && a.cfg.Server.WebPort > 0 {
		webFSForAPI = nil // API server: no SPA routes
	}

	// Setup API router
	router := a.handler.SetupRouter(a.authManager, a.cfg.Server.CorsOrigins, webFSForAPI)

	// Create API HTTP server
	a.server = &http.Server{
		Addr:           fmt.Sprintf("%s:%d", a.cfg.Server.Host, a.cfg.Server.Port),
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Start API server in goroutine
	go func() {
		logx.Info("API server starting", zap.String("address", a.server.Addr))
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logx.Fatal("API server failed", zap.Error(err))
		}
	}()

	// Start dedicated web UI server when web_port is configured.
	// The web server also reverse-proxies /api, /health and /ping to the API
	// server so that the browser can use relative URLs from any origin.
	if a.webFS != nil && a.cfg.Server.WebPort > 0 {
		apiTarget := fmt.Sprintf("http://127.0.0.1:%d", a.cfg.Server.Port)
		a.webServer = &http.Server{
			Addr:           fmt.Sprintf("%s:%d", a.cfg.Server.Host, a.cfg.Server.WebPort),
			Handler:        spaHandler(a.webFS, apiTarget),
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
		go func() {
			logx.Info("Web UI server starting", zap.String("address", a.webServer.Addr))
			if err := a.webServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logx.Fatal("Web UI server failed", zap.Error(err))
			}
		}()
	}

	logx.Info("Application started successfully")
	return nil
}

// migrateSourcesTable adds content-caching columns when upgrading from an older schema.
// Skips gracefully when the sources table doesn't exist yet (e.g. during -init-db).
func migrateSourcesTable(db *store.DB) error {
	// Check if the sources table exists at all; if not, skip — init-db will create it.
	var tableName string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='sources'").Scan(&tableName)
	if err != nil || tableName == "" {
		return nil // table doesn't exist yet; init-db will create it with all columns
	}

	migrations := []struct {
		col string
		ddl string
	}{
		{"content", "ALTER TABLE sources ADD COLUMN content TEXT"},
		{"content_size", "ALTER TABLE sources ADD COLUMN content_size INTEGER DEFAULT 0"},
		{"last_fetch", "ALTER TABLE sources ADD COLUMN last_fetch DATETIME"},
	}
	for _, m := range migrations {
		var dummy any
		checkErr := db.QueryRow("SELECT " + m.col + " FROM sources LIMIT 1").Scan(&dummy)
		if checkErr != nil && checkErr.Error() != "sql: no rows in result set" {
			if _, execErr := db.Exec(m.ddl); execErr != nil {
				return fmt.Errorf("add column %s: %w", m.col, execErr)
			}
			logx.Info("DB migration: added column", zap.String("column", "sources."+m.col))
		}
	}
	return nil
}

// autoStartMihomo restores mihomo if the last recorded runtime status was "running".
func (a *App) autoStartMihomo() {
	if a.runtimeStore == nil {
		return
	}
	runtime, err := a.runtimeStore.Get()
	if err != nil || runtime == nil {
		return
	}
	if runtime.Status != "running" || runtime.ConfigPath == "" {
		return
	}
	logx.Info("Auto-restoring mihomo from previous session", zap.String("config", runtime.ConfigPath))
	if err := a.mihomoManager.Start(runtime.ConfigPath); err != nil {
		logx.Error("Failed to auto-restore mihomo", zap.Error(err))
	} else {
		logx.Info("Mihomo auto-restored successfully")
	}
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

	// Shutdown HTTP servers with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	if a.webServer != nil {
		if err := a.webServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("web server shutdown failed: %w", err)
		}
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

// spaHandler returns a handler that:
//  1. Reverse-proxies /api/, /health and /ping to the API server at apiTarget,
//     so the browser can use relative URLs regardless of which port loads the UI.
//  2. Serves known static files directly from the embedded FS.
//  3. Falls back to index.html for all other paths (Vue Router history mode).
func spaHandler(fsys fs.FS, apiTarget string) http.Handler {
	// Build a reverse proxy that forwards API traffic to the API server.
	target, _ := url.Parse(apiTarget)
	proxy := httputil.NewSingleHostReverseProxy(target)
	// Preserve the original Host header so API CORS/logging works correctly.
	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origDirector(req)
		req.Host = target.Host
	}

	fileServer := http.FileServer(http.FS(fsys))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// Forward API and health-check paths to the API server.
		if strings.HasPrefix(path, "/api/") ||
			path == "/health" ||
			path == "/ping" {
			proxy.ServeHTTP(w, r)
			return
		}
		// Serve known static assets directly.
		if path != "/" {
			f, err := fsys.Open(strings.TrimPrefix(path, "/"))
			if err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		// Everything else → index.html (SPA client-side routing).
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
