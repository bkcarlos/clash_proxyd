package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// Process represents a mihomo process
type Process struct {
	binaryPath string
	configPath string
	logFile    string // path to write mihomo stdout+stderr; empty = os.Stdout/Stderr
	cmd        *exec.Cmd
	logFd      *os.File // kept open while process runs
}

// NewProcess creates a new mihomo process.
// logFile is the path where mihomo output is written; pass "" to inherit proxyd's stdout.
func NewProcess(binaryPath, configPath, logFile string) (*Process, error) {
	// Verify binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("mihomo binary not found: %s", binaryPath)
	}

	// Verify config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	return &Process{
		binaryPath: binaryPath,
		configPath: configPath,
		logFile:    logFile,
	}, nil
}

// Start starts the mihomo process
func (p *Process) Start() error {
	// -d sets mihomo's home directory; -f points to the specific config file.
	p.cmd = exec.Command(p.binaryPath, "-d", filepath.Dir(p.configPath), "-f", p.configPath)

	// Route mihomo output to a dedicated log file when configured.
	if p.logFile != "" {
		if err := os.MkdirAll(filepath.Dir(p.logFile), 0755); err != nil {
			return fmt.Errorf("failed to create mihomo log dir: %w", err)
		}
		f, err := os.OpenFile(p.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open mihomo log file: %w", err)
		}
		p.logFd = f
		p.cmd.Stdout = f
		p.cmd.Stderr = f
	} else {
		p.cmd.Stdout = os.Stdout
		p.cmd.Stderr = os.Stderr
	}

	// Ensure mihomo is killed when proxyd exits (even on SIGKILL / crash).
	p.cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}

	// Start process
	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start mihomo: %w", err)
	}

	// Give process time to start
	time.Sleep(500 * time.Millisecond)

	// Verify process is running
	if !p.isRunning() {
		return fmt.Errorf("mihomo process failed to start")
	}

	return nil
}

// Stop stops the mihomo process
func (p *Process) Stop() error {
	if p.cmd == nil || p.cmd.Process == nil {
		return fmt.Errorf("process not running")
	}

	pid := p.cmd.Process.Pid

	// Try graceful shutdown first
	if err := p.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// If SIGTERM fails, try SIGKILL
		if err := p.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		_, err := p.cmd.Process.Wait()
		done <- err
	}()

	select {
	case <-time.After(5 * time.Second):
		// Timeout, force kill
		_ = p.cmd.Process.Kill()
		return fmt.Errorf("timeout waiting for process to exit")
	case err := <-done:
		if err != nil {
			return fmt.Errorf("process exit error: %w", err)
		}
	}

	// Verify process is stopped
	for i := 0; i < 10; i++ {
		if !p.isRunning() {
			if p.logFd != nil {
				_ = p.logFd.Close()
				p.logFd = nil
			}
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("process may still be running (pid: %d)", pid)
}

// Restart restarts the mihomo process
func (p *Process) Restart() error {
	if err := p.Stop(); err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	return p.Start()
}

// PID returns the process ID
func (p *Process) PID() int {
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return 0
}

// IsRunning returns whether the process is running
func (p *Process) IsRunning() bool {
	return p.isRunning()
}

// isRunning checks if the process is running
func (p *Process) isRunning() bool {
	if p.cmd == nil || p.cmd.Process == nil {
		return false
	}

	// Check if process exists
	process, err := os.FindProcess(p.cmd.Process.Pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process is alive
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return false
	}

	return true
}
