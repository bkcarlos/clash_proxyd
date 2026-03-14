package core

import "testing"

func TestManagerApplyUpdatedBinaryWhenStopped(t *testing.T) {
	m := NewManager("/old/mihomo", "/etc/mihomo", 9090, "")

	if err := m.ApplyUpdatedBinary("/new/mihomo"); err != nil {
		t.Fatalf("apply updated binary failed: %v", err)
	}
	if got := m.GetBinaryPath(); got != "/new/mihomo" {
		t.Fatalf("expected binary path updated, got %s", got)
	}
}

func TestManagerApplyUpdatedBinaryRunningWithoutConfigPath(t *testing.T) {
	m := &Manager{running: true, binaryPath: "/old/mihomo"}

	err := m.ApplyUpdatedBinary("/new/mihomo")
	if err == nil {
		t.Fatalf("expected error when running without config path")
	}
}

func TestManagerRestartWithoutConfigPath(t *testing.T) {
	m := NewManager("/old/mihomo", "/etc/mihomo", 9090, "")

	if err := m.Restart(""); err == nil {
		t.Fatalf("expected restart to fail without config path")
	}
}
