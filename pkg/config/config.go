package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

const (
	defaultMihomoReleaseAPI = "https://api.github.com/repos/MetaCubeX/mihomo/releases/latest"
)

// Config represents the application configuration
type Config struct {
	Server       ServerConfig       `yaml:"server"`
	Database     DatabaseConfig     `yaml:"database"`
	Mihomo       MihomoConfig       `yaml:"mihomo"`
	Auth         AuthConfig         `yaml:"auth"`
	Logging      LoggingConfig      `yaml:"logging"`
	Subscription SubscriptionConfig `yaml:"subscription"`
	Policy       PolicyConfig       `yaml:"policy"`
	Scheduler    SchedulerConfig    `yaml:"scheduler"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host        string   `yaml:"host"`
	Port        int      `yaml:"port"`
	EnableCors  bool     `yaml:"enable_cors"`
	CorsOrigins []string `yaml:"cors_origins"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Path        string `yaml:"path"`
	ForeignKeys bool   `yaml:"foreign_keys"`
}

// MihomoConfig represents mihomo configuration
type MihomoConfig struct {
	BinaryPath              string `yaml:"binary_path"`
	ConfigDir               string `yaml:"config_dir"`
	APIPort                 int    `yaml:"api_port"`
	APISecret               string `yaml:"api_secret"`
	LogDir                  string `yaml:"log_dir"`
	AutoUpdateEnabled       bool   `yaml:"auto_update_enabled"`
	AutoUpdateCheckOnStart  bool   `yaml:"auto_update_check_on_start"`
	ReleaseAPI              string `yaml:"release_api"`
	DownloadDir             string `yaml:"download_dir"`
	TargetVersion           string `yaml:"target_version"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	JWTSecret        string `yaml:"jwt_secret"`
	SessionTimeout   int    `yaml:"session_timeout"`
	MaxLoginAttempts int    `yaml:"max_login_attempts"`
	LockoutDuration  int    `yaml:"lockout_duration"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level      string `yaml:"level"`
	Output     string `yaml:"output"`
	FilePath   string `yaml:"file_path"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
}

// SubscriptionConfig represents subscription configuration
type SubscriptionConfig struct {
	DefaultInterval int    `yaml:"default_interval"`
	Timeout         int    `yaml:"timeout"`
	UserAgent       string `yaml:"user_agent"`
	MaxRetries      int    `yaml:"max_retries"`
	RetryDelay      int    `yaml:"retry_delay"`
}

// PolicyConfig represents policy configuration
type PolicyConfig struct {
	MixedPort          int    `yaml:"mixed_port"`
	AllowLan           bool   `yaml:"allow_lan"`
	BindAddress        string `yaml:"bind_address"`
	LogLevel           string `yaml:"log_level"`
	Mode               string `yaml:"mode"`
	ExternalController string `yaml:"external_controller"`
	IPv6               bool   `yaml:"ipv6"`
}

// SchedulerConfig represents scheduler configuration
type SchedulerConfig struct {
	Enabled bool `yaml:"enabled"`
	Workers int  `yaml:"workers"`
}

// Load loads configuration from file and resolves all relative paths to be
// absolute, anchored at the directory that contains the proxyd executable.
func Load(path string) (*Config, error) {
	baseDir, err := execDir()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve executable directory: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	var raw struct {
		Mihomo struct {
			AutoUpdateCheckOnStart *bool `yaml:"auto_update_check_on_start"`
		} `yaml:"mihomo"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = "data/db/proxyd.db"
	}
	if raw.Mihomo.AutoUpdateCheckOnStart == nil {
		cfg.Mihomo.AutoUpdateCheckOnStart = true
	}
	if cfg.Mihomo.ReleaseAPI == "" {
		cfg.Mihomo.ReleaseAPI = defaultMihomoReleaseAPI
	}
	if cfg.Subscription.UserAgent == "" {
		cfg.Subscription.UserAgent = "clash.meta"
	}
	if cfg.Subscription.Timeout <= 0 {
		cfg.Subscription.Timeout = 30
	}
	if cfg.Subscription.MaxRetries <= 0 {
		cfg.Subscription.MaxRetries = 3
	}
	if cfg.Subscription.RetryDelay <= 0 {
		cfg.Subscription.RetryDelay = 5
	}
	if cfg.Mihomo.ConfigDir == "" {
		cfg.Mihomo.ConfigDir = "data/mihomo"
	}
	if cfg.Mihomo.LogDir == "" {
		cfg.Mihomo.LogDir = "logs"
	}
	if cfg.Logging.FilePath == "" {
		cfg.Logging.FilePath = "logs/proxyd.log"
	}

	// Default mihomo binary to the same directory as the proxyd executable.
	if cfg.Mihomo.BinaryPath == "" {
		cfg.Mihomo.BinaryPath = defaultMihomoBinaryPath()
	}

	// Resolve all relative paths relative to the config file's directory so
	// the program can be started from any working directory.
	cfg.Database.Path = resolveRelative(baseDir, cfg.Database.Path)
	cfg.Mihomo.ConfigDir = resolveRelative(baseDir, cfg.Mihomo.ConfigDir)
	cfg.Mihomo.LogDir = resolveRelative(baseDir, cfg.Mihomo.LogDir)
	cfg.Logging.FilePath = resolveRelative(baseDir, cfg.Logging.FilePath)

	// BinaryPath: keep the executable-relative default absolute, but resolve
	// any user-supplied relative path against baseDir.
	if !filepath.IsAbs(cfg.Mihomo.BinaryPath) {
		cfg.Mihomo.BinaryPath = filepath.Join(baseDir, cfg.Mihomo.BinaryPath)
	}

	if cfg.Mihomo.DownloadDir == "" {
		cfg.Mihomo.DownloadDir = filepath.Dir(cfg.Mihomo.BinaryPath)
	} else {
		cfg.Mihomo.DownloadDir = resolveRelative(baseDir, cfg.Mihomo.DownloadDir)
	}

	return &cfg, nil
}

// resolveRelative converts p to an absolute path anchored at baseDir when p
// is relative; absolute paths are returned unchanged.
func resolveRelative(baseDir, p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(baseDir, p)
}

// execDir returns the absolute directory that contains the running proxyd binary.
// During `go test` / `go run` the symlink is evaluated so we get a stable path.
func execDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	// Resolve symlinks (e.g. go run places binary in a temp dir under a symlink).
	real, err := filepath.EvalSymlinks(exe)
	if err != nil {
		real = exe
	}
	return filepath.Dir(real), nil
}

// defaultMihomoBinaryPath returns the path to the mihomo binary co-located
// with the running proxyd executable. Falls back to "mihomo" (PATH lookup)
// if the executable path cannot be determined.
func defaultMihomoBinaryPath() string {
	dir, err := execDir()
	if err != nil {
		return "mihomo"
	}
	name := "mihomo"
	if runtime.GOOS == "windows" {
		name = "mihomo.exe"
	}
	return filepath.Join(dir, name)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Auth.JWTSecret == "" || c.Auth.JWTSecret == "your-secret-key-change-me" {
		return fmt.Errorf("JWT secret must be set in production")
	}

	if c.Database.Path == "" {
		return fmt.Errorf("database path is required")
	}

	if c.Mihomo.AutoUpdateEnabled {
		if c.Mihomo.ReleaseAPI == "" {
			return fmt.Errorf("mihomo release_api is required when auto update is enabled")
		}
		if c.Mihomo.DownloadDir == "" {
			return fmt.Errorf("mihomo download_dir is required when auto update is enabled")
		}
	}

	return nil
}
