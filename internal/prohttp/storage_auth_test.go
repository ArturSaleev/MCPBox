package prohttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/ArturSaleev/MCPBox/internal/proauth"
	"github.com/ArturSaleev/MCPBox/internal/prostorage"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type proLoginResponse struct {
	Token   string                 `json:"token"`
	User    proauth.UserRecord     `json:"user"`
	Session *proauth.SessionRecord `json:"session"`
}

func TestStorageConfigUpdateDoesNotBreakBootstrapAuth(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mcpboxpro.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}

	runtime := &app.RuntimeContext{DB: db}

	authService, err := proauth.NewServiceFromRuntime(runtime)
	if err != nil {
		t.Fatalf("NewServiceFromRuntime(auth) error = %v", err)
	}
	if err := authService.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("auth AutoMigrate() error = %v", err)
	}
	if _, _, err := authService.CreateOrUpdateLocalPasswordUser(context.Background(), proauth.CreateLocalUserInput{
		Email:       "admin@example.com",
		DisplayName: "System Admin",
		Password:    "password123",
		RoleNames:   []string{"admin"},
		IsBootstrap: true,
	}); err != nil {
		t.Fatalf("CreateOrUpdateLocalPasswordUser() error = %v", err)
	}

	storageService, err := prostorage.NewServiceFromRuntime(runtime)
	if err != nil {
		t.Fatalf("NewServiceFromRuntime(storage) error = %v", err)
	}
	if err := storageService.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("storage AutoMigrate() error = %v", err)
	}

	storageManager, err := prostorage.NewManager(runtime, storageService)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	if _, err := storageManager.Reload(context.Background()); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	mux := http.NewServeMux()
	RegisterRoutes(runtime, mux, authService, nil, nil, nil, nil, storageService, storageManager, nil)

	loginBody := bytes.NewBufferString(`{"email":"admin@example.com","password":"password123","expires_in_days":30}`)
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/pro/login", loginBody)
	loginRequest.Header.Set("Content-Type", "application/json")
	loginResponse := httptest.NewRecorder()
	mux.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d, body=%s", loginResponse.Code, http.StatusOK, loginResponse.Body.String())
	}

	var loginPayload proLoginResponse
	if err := json.Unmarshal(loginResponse.Body.Bytes(), &loginPayload); err != nil {
		t.Fatalf("json.Unmarshal(login) error = %v", err)
	}
	if loginPayload.Token == "" {
		t.Fatal("expected login token")
	}

	updateBody := bytes.NewBufferString(`{
		"operational_backend":"postgres",
		"postgres_host":"127.0.0.1",
		"postgres_port":1,
		"postgres_database":"mcpboxpro",
		"postgres_schema":"public",
		"postgres_username":"user",
		"postgres_password":"secret",
		"logs_backend":"operational"
	}`)
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/pro/storage/config", updateBody)
	updateRequest.Header.Set("Content-Type", "application/json")
	updateRequest.Header.Set("Authorization", "Bearer "+loginPayload.Token)
	updateResponse := httptest.NewRecorder()
	mux.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("storage update status = %d, want %d, body=%s", updateResponse.Code, http.StatusOK, updateResponse.Body.String())
	}

	meRequest := httptest.NewRequest(http.MethodGet, "/api/pro/auth/me", nil)
	meRequest.Header.Set("Authorization", "Bearer "+loginPayload.Token)
	meResponse := httptest.NewRecorder()
	mux.ServeHTTP(meResponse, meRequest)
	if meResponse.Code != http.StatusOK {
		t.Fatalf("auth/me status after storage update = %d, want %d, body=%s", meResponse.Code, http.StatusOK, meResponse.Body.String())
	}

	reloginBody := bytes.NewBufferString(`{"email":"admin@example.com","password":"password123","expires_in_days":30}`)
	reloginRequest := httptest.NewRequest(http.MethodPost, "/api/pro/login", reloginBody)
	reloginRequest.Header.Set("Content-Type", "application/json")
	reloginResponse := httptest.NewRecorder()
	mux.ServeHTTP(reloginResponse, reloginRequest)
	if reloginResponse.Code != http.StatusOK {
		t.Fatalf("relogin status after storage update = %d, want %d, body=%s", reloginResponse.Code, http.StatusOK, reloginResponse.Body.String())
	}
}
