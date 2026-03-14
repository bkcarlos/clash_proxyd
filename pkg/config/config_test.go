package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMihomoAutoUpdateDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	content := `server:
  host: "127.0.0.1"
  port: 8080
database:
  path: "` + filepath.Join(tmpDir, "proxyd.db") + `"
mihomo:
  binary_path: "/usr/local/bin/mihomo"
  config_dir: "` + filepath.Join(tmpDir, "mihomo") + `"
  api_port: 9090
auth:
  jwt_secret: "test-secret"
logging:
  level: "info"
  output: "stdout"
subscription:
  default_interval: 3600
  timeout: 30
  user_agent: "test"
  max_retries: 3
  retry_delay: 5
policy:
  mixed_port: 7890
  allow_lan: true
  bind_address: "*"
  log_level: "info"
  mode: "rule"
  external_controller: "127.0.0.1:9090"
  ipv6: false
scheduler:
  enabled: true
  workers: 1
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	if !cfg.Mihomo.AutoUpdateCheckOnStart {
		t.Fatalf("expected auto_update_check_on_start default true")
	}
	if cfg.Mihomo.ReleaseAPI != defaultMihomoReleaseAPI {
		t.Fatalf("unexpected default release api: %s", cfg.Mihomo.ReleaseAPI)
	}
	// DownloadDir defaults to the directory of the mihomo binary.
	expectedDownloadDir := filepath.Dir(cfg.Mihomo.BinaryPath)
	if cfg.Mihomo.DownloadDir != expectedDownloadDir {
		t.Fatalf("expected download dir %s, got %s", expectedDownloadDir, cfg.Mihomo.DownloadDir)
	}
}

func TestLoadMihomoAutoUpdateCheckOnStartRespectsFalse(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	content := `server:
  host: "127.0.0.1"
  port: 8080
database:
  path: "` + filepath.Join(tmpDir, "proxyd.db") + `"
mihomo:
  binary_path: "/usr/local/bin/mihomo"
  config_dir: "` + filepath.Join(tmpDir, "mihomo") + `"
  api_port: 9090
  auto_update_check_on_start: false
auth:
  jwt_secret: "test-secret"
logging:
  level: "info"
  output: "stdout"
subscription:
  default_interval: 3600
  timeout: 30
  user_agent: "test"
  max_retries: 3
  retry_delay: 5
policy:
  mixed_port: 7890
  allow_lan: true
  bind_address: "*"
  log_level: "info"
  mode: "rule"
  external_controller: "127.0.0.1:9090"
  ipv6: false
scheduler:
  enabled: true
  workers: 1
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	if cfg.Mihomo.AutoUpdateCheckOnStart {
		t.Fatalf("expected auto_update_check_on_start to remain false")
	}
}

func TestValidateAutoUpdateFields(t *testing.T) {
	cfg := &Config{}
	cfg.Server.Port = 8080
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Database.Path = "data.db"
	cfg.Mihomo.BinaryPath = "/usr/local/bin/mihomo"
	cfg.Mihomo.AutoUpdateEnabled = true

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validate error when auto update fields are empty")
	}

	cfg.Mihomo.ReleaseAPI = "https://example.com/releases/latest"
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validate error when download_dir is empty")
	}

	cfg.Mihomo.DownloadDir = "/tmp/bin"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected validate success, got %v", err)
	}
}
