package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/benzjeremy/benzcloud-server/internal/core/auth"
	"github.com/benzjeremy/benzcloud-server/internal/core/config"
	"github.com/benzjeremy/benzcloud-server/internal/core/crypto"
	"github.com/benzjeremy/benzcloud-server/internal/core/dns"
	"github.com/benzjeremy/benzcloud-server/internal/core/drive"
	"github.com/benzjeremy/benzcloud-server/internal/core/nebula"
	"github.com/benzjeremy/benzcloud-server/internal/core/plugins"
)

func TestAPISetupAndPairing(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "benzcloud_api_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	cfg := config.DefaultConfig(tempDir)
	salt, _ := crypto.GenerateSalt(16)
	masterKey := crypto.DeriveKey("MasterTestKey123", salt)

	authMgr, err := auth.NewManager(tempDir)
	if err != nil {
		t.Fatalf("auth.NewManager: %v", err)
	}

	dnsSrv := dns.NewServer("intern", "10.42.0.1", 15354)
	nebulaMgr := nebula.NewManager(tempDir)
	driveMgr, err := drive.NewDriveManager(tempDir, masterKey)
	if err != nil {
		t.Fatalf("drive.NewDriveManager: %v", err)
	}
	pluginMgr := plugins.NewManager(tempDir, "intern", "token123")

	apiServer := NewServer(cfg, authMgr, dnsSrv, nebulaMgr, driveMgr, pluginMgr)
	handler := apiServer.Handler()

	// 1. Check /api/status before setup
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 on /api/status, got %d", rec.Code)
	}
	var statusResp map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&statusResp)
	if statusResp["setup_completed"] != false {
		t.Fatal("Expected setup_completed to be false")
	}

	// 2. Perform /api/setup
	setupPayload := map[string]string{
		"base_domain":    "benzjeremy.de",
		"admin_username": "admin",
		"admin_password": "SuperSecretAdminPassword2026!",
	}
	body, _ := json.Marshal(setupPayload)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewReader(body))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 on /api/setup, got %d: %s", rec.Code, rec.Body.String())
	}
	var setupResp map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&setupResp)
	adminToken := setupResp["session_token"].(string)

	if cfg.BaseDomain != "benzjeremy.de" {
		t.Fatalf("Expected base_domain benzjeremy.de, got %s", cfg.BaseDomain)
	}
	if !cfg.SetupCompleted {
		t.Fatal("Expected SetupCompleted to be true")
	}

	// 3. Client-Kopplung via /api/pair
	pairPayload := map[string]string{
		"username": "admin",
		"password": "SuperSecretAdminPassword2026!",
	}
	pairBody, _ := json.Marshal(pairPayload)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/pair", bytes.NewReader(pairBody))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 on /api/pair, got %d: %s", rec.Code, rec.Body.String())
	}
	var pairResp map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&pairResp)
	if pairResp["status"] != "paired" {
		t.Fatalf("Expected status paired, got %v", pairResp["status"])
	}
	if pairResp["client_config_yaml"] == "" {
		t.Fatal("Expected non-empty client_config_yaml in pairing response")
	}

	// 4. Test Drive Upload & Download via API
	fileContent := []byte("BenzCloud Enterprise Shared File via API")
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/drive/upload?path=report.txt", bytes.NewReader(fileContent))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Drive upload failed with %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/drive/download?path=report.txt", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Drive download failed with %d: %s", rec.Code, rec.Body.String())
	}
	if !bytes.Equal(rec.Body.Bytes(), fileContent) {
		t.Fatalf("Downloaded file mismatch: %s", rec.Body.String())
	}

	// 5. Test User Creation and Invariant Enforcement
	userPayload := map[string]interface{}{
		"username": "charlie",
		"password": "CharliePassword2026!",
		"permissions": map[string]bool{
			"admin": false,
			"drive": true,
			"vpn":   false, // Trying to revoke
			"dns":   false, // Trying to revoke
		},
	}
	uBody, _ := json.Marshal(userPayload)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader(uBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Create user failed: %d: %s", rec.Code, rec.Body.String())
	}
	var createdUser auth.User
	_ = json.NewDecoder(rec.Body).Decode(&createdUser)
	if !createdUser.Permissions.VPN || !createdUser.Permissions.DNS {
		t.Fatal("CRITICAL INVARIANT VIOLATION: Created user must have VPN and DNS set to true!")
	}
}
