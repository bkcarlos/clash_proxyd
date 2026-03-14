package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/clash-proxyd/proxyd/internal/types"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := NewDB(dbPath, true)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		t.Fatalf("failed to read schema: %v", err)
	}
	if err := db.InitSchema(string(schema)); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	return db
}

func TestSourceStoreCRUD(t *testing.T) {
	db := setupTestDB(t)
	store := NewSourceStore(db)

	src := &types.Source{
		Name:           "source-a",
		Type:           "file",
		Path:           "/tmp/a.yaml",
		UpdateInterval: 3600,
		Enabled:        true,
		Priority:       10,
	}

	if err := store.Create(src); err != nil {
		t.Fatalf("create source failed: %v", err)
	}
	if src.ID == 0 {
		t.Fatalf("expected created source id")
	}

	gotByID, err := store.GetByID(src.ID)
	if err != nil {
		t.Fatalf("get by id failed: %v", err)
	}
	if gotByID.Name != src.Name {
		t.Fatalf("expected name %q, got %q", src.Name, gotByID.Name)
	}

	gotByName, err := store.GetByName(src.Name)
	if err != nil {
		t.Fatalf("get by name failed: %v", err)
	}
	if gotByName.ID != src.ID {
		t.Fatalf("expected id %d, got %d", src.ID, gotByName.ID)
	}

	src.Priority = 99
	src.Enabled = false
	if err := store.Update(src); err != nil {
		t.Fatalf("update source failed: %v", err)
	}

	list, err := store.List()
	if err != nil {
		t.Fatalf("list sources failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 source, got %d", len(list))
	}
	if list[0].Priority != 99 {
		t.Fatalf("expected updated priority 99, got %d", list[0].Priority)
	}

	enabled, err := store.GetEnabled()
	if err != nil {
		t.Fatalf("get enabled failed: %v", err)
	}
	if len(enabled) != 0 {
		t.Fatalf("expected 0 enabled sources, got %d", len(enabled))
	}

	if err := store.UpdateLastFetch(src.ID); err != nil {
		t.Fatalf("update last fetch failed: %v", err)
	}

	if err := store.Delete(src.ID); err != nil {
		t.Fatalf("delete source failed: %v", err)
	}

	if _, err := store.GetByID(src.ID); err == nil {
		t.Fatalf("expected not found after delete")
	}
}
