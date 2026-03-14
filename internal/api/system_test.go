package api

import (
	"testing"
	"time"

	"github.com/clash-proxyd/proxyd/internal/types"
)

type fakeAuditStore struct {
	logs []types.AuditLog
	err  error
}

func (f fakeAuditStore) List(limit, offset int) ([]types.AuditLog, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.logs, nil
}

func TestLastAutoUpdateSummary(t *testing.T) {
	ts := time.Now().UTC()
	logs := []types.AuditLog{
		{Action: "login", Resource: "auth", Details: "ok", CreatedAt: ts.Add(-2 * time.Minute)},
		{Action: "mihomo_update_applied", Resource: "mihomo", Details: "from=1.0.0 to=1.0.1", CreatedAt: ts},
	}

	action, details, at, ok := lastAutoUpdateSummary(fakeAuditStore{logs: logs})
	if !ok {
		t.Fatalf("expected summary to exist")
	}
	if action != "mihomo_update_applied" {
		t.Fatalf("unexpected action: %s", action)
	}
	if details == "" {
		t.Fatalf("expected details")
	}
	if !at.Equal(ts) {
		t.Fatalf("unexpected timestamp")
	}
}

func TestLastAutoUpdateSummaryNotFound(t *testing.T) {
	logs := []types.AuditLog{
		{Action: "login", Resource: "auth", Details: "ok"},
	}

	_, _, _, ok := lastAutoUpdateSummary(fakeAuditStore{logs: logs})
	if ok {
		t.Fatalf("expected no summary")
	}
}
