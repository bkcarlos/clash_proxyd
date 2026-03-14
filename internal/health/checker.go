package health

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/clash-proxyd/proxyd/internal/logx"
	"github.com/clash-proxyd/proxyd/internal/types"

	"go.uber.org/zap"
)

// Checker performs health checks
type Checker struct {
	mu         sync.RWMutex
	services   map[string]CheckFunc
	lastCheck  time.Time
	lastStatus *types.HealthStatus
	ticker     *time.Ticker
	stopCh     chan struct{}
}

// CheckFunc is a function that performs a health check
type CheckFunc func() error

// NewChecker creates a new health checker
func NewChecker() *Checker {
	return &Checker{
		services: make(map[string]CheckFunc),
	}
}

// Register registers a health check function
func (c *Checker) Register(name string, fn CheckFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services[name] = fn
}

// Unregister unregisters a health check function
func (c *Checker) Unregister(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.services, name)
}

// Check performs all registered health checks
func (c *Checker) Check() *types.HealthStatus {
	status := &types.HealthStatus{
		Timestamp: time.Now(),
		Services:  make(map[string]string),
	}

	c.mu.RLock()
	services := make(map[string]CheckFunc, len(c.services))
	for name, fn := range c.services {
		services[name] = fn
	}
	c.mu.RUnlock()

	allHealthy := true
	var wg sync.WaitGroup
	var resultMu sync.Mutex

	for name, fn := range services {
		wg.Add(1)
		go func(name string, fn CheckFunc) {
			defer wg.Done()
			if err := fn(); err != nil {
				resultMu.Lock()
				status.Services[name] = fmt.Sprintf("unhealthy: %v", err)
				allHealthy = false
				resultMu.Unlock()
				return
			}
			resultMu.Lock()
			status.Services[name] = "healthy"
			resultMu.Unlock()
		}(name, fn)
	}

	wg.Wait()

	if allHealthy {
		status.Status = "healthy"
	} else {
		status.Status = "unhealthy"
	}

	c.mu.Lock()
	c.lastCheck = time.Now()
	c.lastStatus = status
	c.mu.Unlock()

	return status
}

// GetStatus returns the last health check status
func (c *Checker) GetStatus() *types.HealthStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastStatus
}

// GetLastCheck returns the last check time
func (c *Checker) GetLastCheck() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastCheck
}

// CheckMihomo checks if mihomo binary is available
func CheckMihomo(mihomoPath string) CheckFunc {
	return func() error {
		_, err := exec.LookPath(mihomoPath)
		if err != nil {
			return fmt.Errorf("mihomo binary not found: %w", err)
		}
		return nil
	}
}

// CheckMihomoRunning checks if mihomo process is running
func CheckMihomoRunning(pid int) CheckFunc {
	return func() error {
		if pid <= 0 {
			return fmt.Errorf("mihomo not running")
		}

		process, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("failed to find process: %w", err)
		}

		// Check if process is running
		if runtime.GOOS != "windows" {
			err = process.Signal(syscall.Signal(0))
			if err != nil {
				return fmt.Errorf("process not running: %w", err)
			}
		}

		return nil
	}
}

// CheckDatabase checks database connectivity
func CheckDatabase(db DatabaseChecker) CheckFunc {
	return func() error {
		return db.Ping()
	}
}

// DatabaseChecker defines database ping interface
type DatabaseChecker interface {
	Ping() error
}

// CheckDiskSpace checks available disk space (Linux only)
func CheckDiskSpace(path string, minFreeMB int64) CheckFunc {
	return func() error {
		if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
			return nil // Skip on non-Unix systems
		}

		var stat syscall.Statfs_t
		if err := syscall.Statfs(path, &stat); err != nil {
			return fmt.Errorf("failed to stat fs: %w", err)
		}

		// Calculate available space in bytes
		available := stat.Bavail * uint64(stat.Bsize)
		availableMB := available / (1024 * 1024)

		if availableMB < uint64(minFreeMB) {
			return fmt.Errorf("low disk space: %d MB available, %d MB required", availableMB, minFreeMB)
		}

		return nil
	}
}

// CheckPathWritable checks whether a path (or its parent) is writable.
func CheckPathWritable(path string, pathIsFile bool) CheckFunc {
	return func() error {
		target := path
		if pathIsFile {
			target = filepath.Dir(path)
		}
		if err := os.MkdirAll(target, 0755); err != nil {
			return fmt.Errorf("failed to ensure path exists: %w", err)
		}

		probe, err := os.CreateTemp(target, ".proxyd-write-check-*")
		if err != nil {
			return fmt.Errorf("path not writable: %w", err)
		}
		name := probe.Name()
		_ = probe.Close()
		_ = os.Remove(name)
		return nil
	}
}

// CheckMemory checks if memory stats can be read.
func CheckMemory() CheckFunc {
	return func() error {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		if m.Sys == 0 {
			return fmt.Errorf("failed to read memory stats")
		}

		return nil
	}
}

// StartPeriodicCheck starts periodic health checks.
// Calling it again while already running is a no-op.
func (c *Checker) StartPeriodicCheck(interval time.Duration) {
	c.mu.Lock()
	if c.ticker != nil {
		c.mu.Unlock()
		return
	}
	c.ticker = time.NewTicker(interval)
	c.stopCh = make(chan struct{})
	ticker := c.ticker
	stopCh := c.stopCh
	c.mu.Unlock()

	go func() {
		for {
			select {
			case <-ticker.C:
				status := c.Check()
				if status.Status == "unhealthy" {
					logx.Warn("Health check failed", zap.Any("status", status))
				}
			case <-stopCh:
				return
			}
		}
	}()
}

// StopPeriodicCheck stops the periodic health check goroutine.
func (c *Checker) StopPeriodicCheck() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ticker == nil {
		return
	}
	c.ticker.Stop()
	close(c.stopCh)
	c.ticker = nil
	c.stopCh = nil
}
