package core

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("write executable failed: %v", err)
	}
}

func TestUpdaterNeedsUpdate(t *testing.T) {
	u := NewUpdater(UpdaterConfig{})

	if !u.NeedsUpdate("1.0.0", "1.0.1") {
		t.Fatalf("expected update needed")
	}
	if u.NeedsUpdate("1.2.0", "1.1.9") {
		t.Fatalf("expected no update for older latest")
	}
	if u.NeedsUpdate("1.2.3", "1.2.3") {
		t.Fatalf("expected no update for same version")
	}
}

func TestUpdaterCheckAndUpdateNoUpdate(t *testing.T) {
	tmp := t.TempDir()
	binaryPath := filepath.Join(tmp, "mihomo")
	writeExecutable(t, binaryPath, "#!/bin/sh\necho 'Mihomo Meta v1.2.3'\n")

	assetName := fmt.Sprintf("mihomo-%s-%s", runtime.GOOS, runtime.GOARCH)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		release := map[string]any{
			"tag_name": "v1.2.3",
			"assets": []map[string]string{{
				"name":                 assetName,
				"browser_download_url": "http://example.invalid/asset",
			}},
		}
		_ = json.NewEncoder(w).Encode(release)
	}))
	defer ts.Close()

	u := NewUpdater(UpdaterConfig{ReleaseAPI: ts.URL, DownloadDir: filepath.Join(tmp, "bin")})
	result, err := u.CheckAndUpdate(context.Background(), binaryPath)
	if err != nil {
		t.Fatalf("check and update failed: %v", err)
	}
	if result.Updated {
		t.Fatalf("expected no update")
	}
}

func TestUpdaterCheckAndUpdateSuccessAndRollback(t *testing.T) {
	tmp := t.TempDir()
	binaryPath := filepath.Join(tmp, "mihomo")
	writeExecutable(t, binaryPath, "#!/bin/sh\nif [ \"$1\" = \"-v\" ]; then echo 'Mihomo Meta v1.0.0'; else echo old; fi\n")

	assetName := fmt.Sprintf("mihomo-%s-%s", runtime.GOOS, runtime.GOARCH)
	newBinary := "#!/bin/sh\nif [ \"$1\" = \"-v\" ]; then echo 'Mihomo Meta v1.0.1'; else echo new; fi\n"

	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/release":
			release := map[string]any{
				"tag_name": "v1.0.1",
				"assets": []map[string]string{{
					"name":                 assetName,
					"browser_download_url": ts.URL + "/asset",
				}},
			}
			_ = json.NewEncoder(w).Encode(release)
		case "/asset":
			_, _ = w.Write([]byte(newBinary))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	u := NewUpdater(UpdaterConfig{ReleaseAPI: ts.URL + "/release", DownloadDir: filepath.Join(tmp, "downloads")})
	result, err := u.CheckAndUpdate(context.Background(), binaryPath)
	if err != nil {
		t.Fatalf("check and update failed: %v", err)
	}
	if !result.Updated {
		t.Fatalf("expected updated=true")
	}
	if _, err := os.Stat(result.BackupPath); err != nil {
		t.Fatalf("expected backup file to exist: %v", err)
	}
	ver, err := detectBinaryVersion(binaryPath)
	if err != nil {
		t.Fatalf("detect new version failed: %v", err)
	}
	if ver != "1.0.1" {
		t.Fatalf("expected new version 1.0.1, got %s", ver)
	}

	if err := u.Rollback(result); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}
	ver, err = detectBinaryVersion(binaryPath)
	if err != nil {
		t.Fatalf("detect rolled back version failed: %v", err)
	}
	if ver != "1.0.0" {
		t.Fatalf("expected rolled back version 1.0.0, got %s", ver)
	}
}

func TestUpdaterCheckAndUpdateDownloadFailure(t *testing.T) {
	tmp := t.TempDir()
	binaryPath := filepath.Join(tmp, "mihomo")
	writeExecutable(t, binaryPath, "#!/bin/sh\necho 'Mihomo Meta v1.0.0'\n")

	assetName := fmt.Sprintf("mihomo-%s-%s", runtime.GOOS, runtime.GOARCH)
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/release":
			release := map[string]any{
				"tag_name": "v1.0.1",
				"assets": []map[string]string{{
					"name":                 assetName,
					"browser_download_url": ts.URL + "/asset",
				}},
			}
			_ = json.NewEncoder(w).Encode(release)
		case "/asset":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	u := NewUpdater(UpdaterConfig{ReleaseAPI: ts.URL + "/release", DownloadDir: filepath.Join(tmp, "downloads")})
	_, err := u.CheckAndUpdate(context.Background(), binaryPath)
	if err == nil {
		t.Fatalf("expected download failure")
	}
}
