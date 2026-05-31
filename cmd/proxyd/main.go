// Command proxyd is the control-plane daemon for managing a mihomo (Clash Meta)
// instance: it loads configuration, initializes the database, serves the
// management API (and optionally the embedded web UI), and supervises the
// mihomo process lifecycle.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/clash-proxyd/proxyd/internal/app"
	"github.com/clash-proxyd/proxyd/internal/core"
	"github.com/clash-proxyd/proxyd/internal/store"
	"github.com/clash-proxyd/proxyd/internal/webui"
	"github.com/clash-proxyd/proxyd/pkg/config"
)

// version is the proxyd release version. Keep in sync with internal/app.
const version = "1.0.0"

func main() {
	var (
		configPath   = flag.String("c", "config.yaml", "path to the YAML configuration file")
		initDB       = flag.Bool("init-db", false, "initialize the database schema and exit")
		serveWeb     = flag.Bool("web", false, "serve the embedded web UI (requires a binary built with -tags webui, e.g. `make build-all`)")
		checkConfig  = flag.Bool("check", false, "validate the configuration file and exit")
		showVersion  = flag.Bool("version", false, "print the proxyd version and exit")
		mihomoAction = flag.String("mihomo", "", "control the mihomo process and exit: start|stop|restart|status")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("proxyd %s\n", version)
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fatalf("failed to load config %q: %v", *configPath, err)
	}

	// -check: validate the configuration and exit without side effects.
	if *checkConfig {
		if err := cfg.Validate(); err != nil {
			fatalf("config validation failed: %v", err)
		}
		fmt.Printf("config %q is valid\n", *configPath)
		return
	}

	// -mihomo <action>: one-shot mihomo control, then exit.
	if *mihomoAction != "" {
		if err := runMihomoControl(cfg, *mihomoAction); err != nil {
			fatalf("mihomo %s: %v", *mihomoAction, err)
		}
		return
	}

	// -init-db: (re)apply the database schema and exit.
	if *initDB {
		application, err := app.New(cfg)
		if err != nil {
			fatalf("failed to initialize application: %v", err)
		}
		if err := application.InitDB(); err != nil {
			fatalf("failed to initialize database: %v", err)
		}
		fmt.Println("database initialized")
		return
	}

	// Normal daemon mode: validate config (rejects empty/placeholder JWT secret,
	// bad port, etc.) before doing any startup work.
	if err := cfg.Validate(); err != nil {
		fatalf("config validation failed: %v", err)
	}

	application, err := app.New(cfg)
	if err != nil {
		fatalf("failed to initialize application: %v", err)
	}

	// Serve the embedded web UI only when explicitly requested. webui.FS is
	// non-nil only in binaries built with -tags webui; bail out clearly if the
	// flag is used against an API-only build.
	if *serveWeb {
		if webui.FS == nil {
			fatalf("-web requires an embedded web UI; rebuild with the web UI bundled (`make build-all`, i.e. go build -tags webui)")
		}
		application.EnableWebUI(webui.FS)
	}

	if err := application.Run(); err != nil {
		fatalf("application error: %v", err)
	}
}

// runMihomoControl performs a one-shot mihomo lifecycle action.
//
// status/stop operate across processes via the runtime table (the source of
// truth for the PID launched by the proxyd daemon). start/restart launch a
// fresh mihomo in the foreground (the process is tied to this command's
// lifetime via Pdeathsig), so they are intended for standalone use when the
// proxyd daemon is not already supervising mihomo.
func runMihomoControl(cfg *config.Config, action string) error {
	switch action {
	case "status":
		return mihomoStatus(cfg)
	case "stop":
		return mihomoStop(cfg)
	case "start":
		return mihomoStartForeground(cfg)
	case "restart":
		if err := mihomoStop(cfg); err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
		return mihomoStartForeground(cfg)
	default:
		return fmt.Errorf("unknown action %q (want start|stop|restart|status)", action)
	}
}

func mihomoStatus(cfg *config.Config) error {
	db, rs, err := openRuntimeStore(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	running, err := rs.GetAllRunning()
	if err != nil || len(running) == 0 {
		fmt.Println("mihomo: stopped")
		return nil
	}
	alive := false
	for _, rt := range running {
		if pidAlive(rt.PID) {
			alive = true
			fmt.Printf("mihomo: running (pid=%d, config=%s)\n", rt.PID, rt.ConfigPath)
		}
	}
	if !alive {
		fmt.Println("mihomo: stopped (stale runtime records)")
	}
	return nil
}

func mihomoStop(cfg *config.Config) error {
	db, rs, err := openRuntimeStore(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	running, err := rs.GetAllRunning()
	if err != nil {
		return fmt.Errorf("query running mihomo: %w", err)
	}
	stopped := 0
	for _, rt := range running {
		if !pidAlive(rt.PID) {
			continue
		}
		proc, err := os.FindProcess(rt.PID)
		if err != nil {
			continue
		}
		_ = proc.Signal(syscall.SIGTERM)
		time.Sleep(500 * time.Millisecond)
		if pidAlive(rt.PID) {
			_ = proc.Kill()
		}
		stopped++
		fmt.Printf("mihomo: stopped pid=%d\n", rt.PID)
	}
	if err := rs.PurgeStaleRows(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to purge runtime records: %v\n", err)
	}
	if stopped == 0 {
		fmt.Println("mihomo: not running")
	}
	return nil
}

func mihomoStartForeground(cfg *config.Config) error {
	configPath := filepath.Join(cfg.Mihomo.ConfigDir, "runtime.yaml")
	// Prefer the last config path recorded by the daemon, if available.
	if db, rs, err := openRuntimeStore(cfg); err == nil {
		if rt, gerr := rs.Get(); gerr == nil && rt != nil && rt.ConfigPath != "" {
			configPath = rt.ConfigPath
		}
		db.Close()
	}

	mgr := core.NewManager(
		cfg.Mihomo.BinaryPath,
		cfg.Mihomo.ConfigDir,
		cfg.Mihomo.LogDir,
		cfg.Mihomo.APIPort,
		cfg.Mihomo.APISecret,
	)
	if err := mgr.Start(configPath); err != nil {
		return err
	}
	fmt.Printf("mihomo: started (pid=%d, config=%s)\n", mgr.GetPID(), configPath)
	fmt.Println("mihomo: running in the foreground — press Ctrl-C to stop")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	if err := mgr.Stop(); err != nil {
		return err
	}
	fmt.Println("mihomo: stopped")
	return nil
}

// openRuntimeStore opens the SQLite database and returns a RuntimeStore.
// The caller is responsible for closing the returned *store.DB.
func openRuntimeStore(cfg *config.Config) (*store.DB, *store.RuntimeStore, error) {
	db, err := store.NewDB(cfg.Database.Path, cfg.Database.ForeignKeys)
	if err != nil {
		return nil, nil, fmt.Errorf("open database: %w", err)
	}
	return db, store.NewRuntimeStore(db), nil
}

// pidAlive reports whether a process with the given PID is currently alive.
func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "proxyd: "+format+"\n", args...)
	os.Exit(1)
}
