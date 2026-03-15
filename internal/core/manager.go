package core

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/clash-proxyd/proxyd/internal/logx"
	"github.com/clash-proxyd/proxyd/internal/types"

	"go.uber.org/zap"
)

// Manager manages mihomo process lifecycle
type Manager struct {
	mu                sync.RWMutex
	binaryPath        string
	configDir         string
	logDir            string
	apiPort           int
	apiSecret         string
	process           *Process
	client            *Client
	running           bool
	currentConfigPath string
}

// NewManager creates a new mihomo manager
func NewManager(binaryPath, configDir, logDir string, apiPort int, apiSecret string) *Manager {
	return &Manager{
		binaryPath: binaryPath,
		configDir:  configDir,
		logDir:     logDir,
		apiPort:    apiPort,
		apiSecret:  apiSecret,
	}
}

// mihomoLogFile returns the path for mihomo's log file.
// Returns empty string if logDir is not configured.
func (m *Manager) mihomoLogFile() string {
	if m.logDir == "" {
		return ""
	}
	return filepath.Join(m.logDir, "mihomo.log")
}

// Start starts the mihomo process
func (m *Manager) Start(configPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("mihomo is already running")
	}

	// Create process
	process, err := NewProcess(m.binaryPath, configPath, m.mihomoLogFile())
	if err != nil {
		return fmt.Errorf("failed to create process: %w", err)
	}

	// Start process
	if err := process.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	// Wait for mihomo to be ready
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := m.waitForReady(ctx); err != nil {
		process.Stop()
		return fmt.Errorf("mihomo did not become ready: %w", err)
	}

	// Create API client
	m.client = NewClient(fmt.Sprintf("http://127.0.0.1:%d", m.apiPort), m.apiSecret)

	m.process = process
	m.running = true
	m.currentConfigPath = configPath

	logx.Info("Mihomo started successfully",
		zap.Int("pid", process.PID()),
		zap.String("config", configPath),
		zap.String("binary", m.binaryPath))

	return nil
}

// Stop stops the mihomo process
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return fmt.Errorf("mihomo is not running")
	}

	if err := m.process.Stop(); err != nil {
		// If the process already exited on its own, that is still a
		// successful stop from our perspective — always clean up state.
		logx.Warn("error signalling mihomo during stop (process may have already exited)",
			zap.Error(err))
	}

	m.running = false
	m.process = nil
	m.client = nil

	logx.Info("Mihomo stopped successfully")

	return nil
}

// Restart restarts the mihomo process
func (m *Manager) Restart(configPath string) error {
	if configPath == "" {
		configPath = m.GetCurrentConfigPath()
	}
	if configPath == "" {
		return fmt.Errorf("config path is required")
	}

	if err := m.Stop(); err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	return m.Start(configPath)
}

// ApplyUpdatedBinary switches to updated binary and restarts mihomo if needed.
func (m *Manager) ApplyUpdatedBinary(newBinaryPath string) error {
	if newBinaryPath == "" {
		return fmt.Errorf("new binary path is required")
	}

	m.mu.RLock()
	wasRunning := m.running
	currentConfigPath := m.currentConfigPath
	m.mu.RUnlock()

	if !wasRunning {
		m.SetBinaryPath(newBinaryPath)
		logx.Info("Mihomo binary path updated (process not running)", zap.String("binary", newBinaryPath))
		return nil
	}

	if currentConfigPath == "" {
		return fmt.Errorf("cannot apply updated binary while running without current config path")
	}

	if err := m.Stop(); err != nil {
		return fmt.Errorf("failed to stop mihomo before applying updated binary: %w", err)
	}

	m.SetBinaryPath(newBinaryPath)
	if err := m.Start(currentConfigPath); err != nil {
		return fmt.Errorf("failed to start mihomo with updated binary: %w", err)
	}

	logx.Info("Applied updated mihomo binary", zap.String("binary", newBinaryPath))
	return nil
}

// SetBinaryPath updates the binary path.
func (m *Manager) SetBinaryPath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.binaryPath = path
}

// GetBinaryPath returns current binary path.
func (m *Manager) GetBinaryPath() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.binaryPath
}

// GetCurrentConfigPath returns last used config path.
func (m *Manager) GetCurrentConfigPath() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentConfigPath
}

// IsRunning returns whether mihomo is running.
// If the in-memory flag says running but the process has already exited,
// it cleans up the stale state and returns false.
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	running := m.running
	process := m.process
	m.mu.RUnlock()

	if !running {
		return false
	}

	// Verify the process is still actually alive.
	if process != nil && !process.IsRunning() {
		m.mu.Lock()
		m.running = false
		m.process = nil
		m.client = nil
		m.mu.Unlock()
		logx.Warn("mihomo process exited unexpectedly; cleared stale running state")
		return false
	}

	return true
}

// GetPID returns the mihomo process ID
func (m *Manager) GetPID() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.process != nil {
		return m.process.PID()
	}
	return 0
}

// GetClient returns the mihomo API client
func (m *Manager) GetClient() *Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.client
}

// GetVersion returns the mihomo version
func (m *Manager) GetVersion() (string, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return "", fmt.Errorf("mihomo is not running")
	}

	return client.GetVersion()
}

// GetTraffic returns traffic statistics
func (m *Manager) GetTraffic() (*types.MihomoTraffic, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("mihomo is not running")
	}

	return client.GetTraffic()
}

// GetMemory returns memory usage
func (m *Manager) GetMemory() (*types.MihomoMemory, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("mihomo is not running")
	}

	return client.GetMemory()
}

// GetProxies returns all proxies
func (m *Manager) GetProxies() (map[string]interface{}, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("mihomo is not running")
	}

	return client.GetProxies()
}

// GetProxy returns a specific proxy
func (m *Manager) GetProxy(name string) (map[string]interface{}, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("mihomo is not running")
	}

	return client.GetProxy(name)
}

// GetConnections returns active connections
func (m *Manager) GetConnections() (map[string]interface{}, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()
	if client == nil {
		return nil, fmt.Errorf("mihomo is not running")
	}
	return client.GetConnections()
}

// CloseConnection closes a specific connection
func (m *Manager) CloseConnection(id string) error {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()
	if client == nil {
		return fmt.Errorf("mihomo is not running")
	}
	return client.CloseConnection(id)
}

// CloseAllConnections closes all connections
func (m *Manager) CloseAllConnections() error {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()
	if client == nil {
		return fmt.Errorf("mihomo is not running")
	}
	return client.CloseAllConnections()
}

// GetRules returns active rules
func (m *Manager) GetRules() (map[string]interface{}, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("mihomo is not running")
	}

	return client.GetRules()
}

// GetProxyDelay returns proxy delay
func (m *Manager) GetProxyDelay(proxyName string, testURL string, timeout int) (int, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return 0, fmt.Errorf("mihomo is not running")
	}

	return client.GetProxyDelay(proxyName, testURL, timeout)
}

// SwitchProxy switches to a proxy in a group
func (m *Manager) SwitchProxy(group, proxy string) error {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("mihomo is not running")
	}

	return client.SwitchProxy(group, proxy)
}

// waitForReady waits for mihomo API to be ready
func (m *Manager) waitForReady(ctx context.Context) error {
	client := NewClient(fmt.Sprintf("http://127.0.0.1:%d", m.apiPort), m.apiSecret)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := client.GetVersion(); err == nil {
				return nil
			}
		}
	}
}
