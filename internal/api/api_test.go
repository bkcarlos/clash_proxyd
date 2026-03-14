package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/clash-proxyd/proxyd/internal/auth"
	"github.com/clash-proxyd/proxyd/internal/renderer"
	"github.com/clash-proxyd/proxyd/internal/store"
	"github.com/clash-proxyd/proxyd/pkg/config"
	"github.com/gin-gonic/gin"
)

func setupRouter(t *testing.T) (*gin.Engine, *store.SettingStore) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "api-test.db")
	db, err := store.NewDB(dbPath, true)
	if err != nil {
		t.Fatalf("new db failed: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schema, err := os.ReadFile(filepath.Join("..", "store", "schema.sql"))
	if err != nil {
		t.Fatalf("read schema failed: %v", err)
	}
	if err := db.InitSchema(string(schema)); err != nil {
		t.Fatalf("init schema failed: %v", err)
	}

	sourceStore := store.NewSourceStore(db)
	settingStore := store.NewSettingStore(db)
	revisionStore := store.NewRevisionStore(db)
	runtimeStore := store.NewRuntimeStore(db)
	auditStore := store.NewAuditStore(db)

	authManager := auth.NewManager("test-secret", 3600, settingStore)
	if err := authManager.SetCredentials("admin", "admin"); err != nil {
		t.Fatalf("set credentials failed: %v", err)
	}

	h := NewHandler(
		authManager,
		sourceStore,
		settingStore,
		revisionStore,
		runtimeStore,
		auditStore,
		nil,
		nil,
		renderer.NewRenderer(&config.PolicyConfig{}),
		nil,
		tmpDir,
		7890,
	)

	return h.SetupRouter(authManager, []string{"*"}), settingStore
}

func doJSONRequest(t *testing.T, router http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body failed: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func loginAndGetToken(t *testing.T, router http.Handler) string {
	t.Helper()
	resp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": "admin",
	}, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("login failed: status=%d body=%s", resp.Code, resp.Body.String())
	}

	var data struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &data); err != nil {
		t.Fatalf("unmarshal login response failed: %v", err)
	}
	if data.Token == "" {
		t.Fatalf("expected token in login response")
	}
	return data.Token
}

func TestLogin(t *testing.T) {
	router, _ := setupRouter(t)
	_ = loginAndGetToken(t, router)
}

func TestUpdateSettingsBatch(t *testing.T) {
	router, settingStore := setupRouter(t)
	token := loginAndGetToken(t, router)

	resp := doJSONRequest(t, router, http.MethodPut, "/api/v1/system/settings/batch", map[string]any{
		"settings": []map[string]string{
			{"key": "log_level", "value": "debug", "description": "Log level"},
			{"key": "enable_cors", "value": "false", "description": "Enable CORS"},
		},
	}, token)

	if resp.Code != http.StatusOK {
		t.Fatalf("update settings batch failed: status=%d body=%s", resp.Code, resp.Body.String())
	}

	logLevel, err := settingStore.Get("log_level")
	if err != nil {
		t.Fatalf("get log_level failed: %v", err)
	}
	if logLevel != "debug" {
		t.Fatalf("expected log_level debug, got %q", logLevel)
	}

	enableCors, err := settingStore.Get("enable_cors")
	if err != nil {
		t.Fatalf("get enable_cors failed: %v", err)
	}
	if enableCors != "false" {
		t.Fatalf("expected enable_cors false, got %q", enableCors)
	}
}

func TestMihomoControlInvalidAction(t *testing.T) {
	router, _ := setupRouter(t)
	token := loginAndGetToken(t, router)

	resp := doJSONRequest(t, router, http.MethodPost, "/api/v1/proxy/mihomo/not-supported", nil, token)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid action, got %d body=%s", resp.Code, resp.Body.String())
	}
}
