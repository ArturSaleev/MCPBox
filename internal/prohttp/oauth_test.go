package prohttp

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestChatGPTOAuthCodeFlowIssuesProjectBearerToken(t *testing.T) {
	server, db := newOAuthTestServer(t)
	defer server.Close()

	verifier := "test-code-verifier"
	challengeBytes := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(challengeBytes[:])
	callback := server.URL + "/callback"
	if err := db.Exec(`UPDATE projects SET oauth_redirect_uri = ? WHERE token = ?`, callback, "project-token").Error; err != nil {
		t.Fatalf("update project callback: %v", err)
	}

	code := authorizeOAuthTestCode(t, server, callback, challenge)

	tokenResponse := exchangeOAuthTestCode(t, server, code, callback, verifier, "project-bearer-secret")
	defer tokenResponse.Body.Close()
	if tokenResponse.StatusCode != http.StatusOK {
		t.Fatalf("token status = %d, want %d", tokenResponse.StatusCode, http.StatusOK)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}
	if err := json.NewDecoder(tokenResponse.Body).Decode(&payload); err != nil {
		t.Fatalf("decode token response: %v", err)
	}
	if payload.AccessToken != "project-bearer-secret" {
		t.Fatalf("access_token = %q, want project bearer token", payload.AccessToken)
	}
	if payload.TokenType != "Bearer" {
		t.Fatalf("token_type = %q, want Bearer", payload.TokenType)
	}
}

func TestChatGPTOAuthCodeFlowUsesNamedProjectOAuthClient(t *testing.T) {
	server, db := newOAuthTestServer(t)
	defer server.Close()

	callback := server.URL + "/callback-user-one"
	if err := db.Exec(
		`INSERT INTO project_oauth_clients (project_id, name, redirect_uri, token, is_enabled) VALUES (?, ?, ?, ?, ?)`,
		1,
		"chatgpt",
		callback,
		"client-specific-secret",
		true,
	).Error; err != nil {
		t.Fatalf("insert OAuth client: %v", err)
	}

	verifier := "named-client-code-verifier"
	challengeBytes := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(challengeBytes[:])
	code := authorizeOAuthTestCode(t, server, callback, challenge)

	tokenResponse := exchangeOAuthTestCode(t, server, code, callback, verifier, "client-specific-secret")
	defer tokenResponse.Body.Close()
	if tokenResponse.StatusCode != http.StatusOK {
		t.Fatalf("token status = %d, want %d", tokenResponse.StatusCode, http.StatusOK)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(tokenResponse.Body).Decode(&payload); err != nil {
		t.Fatalf("decode token response: %v", err)
	}
	if payload.AccessToken != "client-specific-secret" {
		t.Fatalf("access_token = %q, want named client token", payload.AccessToken)
	}
}

func TestChatGPTOAuthTokenInvalidClientDoesNotConsumeCode(t *testing.T) {
	server, db := newOAuthTestServer(t)
	defer server.Close()

	verifier := "test-code-verifier"
	challengeBytes := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(challengeBytes[:])
	callback := server.URL + "/callback"
	if err := db.Exec(`UPDATE projects SET oauth_redirect_uri = ? WHERE token = ?`, callback, "project-token").Error; err != nil {
		t.Fatalf("update project callback: %v", err)
	}

	code := authorizeOAuthTestCode(t, server, callback, challenge)
	badResponse := exchangeOAuthTestCode(t, server, code, callback, verifier, "wrong-secret")
	_ = badResponse.Body.Close()
	if badResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad token status = %d, want %d", badResponse.StatusCode, http.StatusUnauthorized)
	}

	goodResponse := exchangeOAuthTestCode(t, server, code, callback, verifier, "project-bearer-secret")
	defer goodResponse.Body.Close()
	if goodResponse.StatusCode != http.StatusOK {
		t.Fatalf("second token status = %d, want %d", goodResponse.StatusCode, http.StatusOK)
	}
}

func TestProtectedResourceMetadataOnlyAdvertisedForProtectedProject(t *testing.T) {
	server, db := newOAuthTestServer(t)
	defer server.Close()

	protectedResponse, err := server.Client().Get(server.URL + "/.well-known/oauth-protected-resource/mcp/project-token")
	if err != nil {
		t.Fatalf("get protected metadata: %v", err)
	}
	defer protectedResponse.Body.Close()
	if protectedResponse.StatusCode != http.StatusOK {
		t.Fatalf("protected metadata status = %d, want %d", protectedResponse.StatusCode, http.StatusOK)
	}

	if err := db.Exec(
		`INSERT INTO projects (name, token, bearer_auth_enabled, bearer_token) VALUES (?, ?, ?, ?)`,
		"Public Project",
		"public-project-token",
		false,
		"",
	).Error; err != nil {
		t.Fatalf("insert public project: %v", err)
	}

	publicResponse, err := server.Client().Get(server.URL + "/.well-known/oauth-protected-resource/mcp/public-project-token")
	if err != nil {
		t.Fatalf("get public metadata: %v", err)
	}
	defer publicResponse.Body.Close()
	if publicResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("public metadata status = %d, want %d", publicResponse.StatusCode, http.StatusNotFound)
	}

	rootResponse, err := server.Client().Get(server.URL + "/.well-known/oauth-protected-resource")
	if err != nil {
		t.Fatalf("get root metadata: %v", err)
	}
	defer rootResponse.Body.Close()
	if rootResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("root metadata status = %d, want %d", rootResponse.StatusCode, http.StatusNotFound)
	}
}

func newOAuthTestServer(t *testing.T) (*httptest.Server, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mcpboxpro.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Exec(`CREATE TABLE projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		token TEXT,
		bearer_auth_enabled NUMERIC,
		bearer_token TEXT,
		oauth_redirect_uri TEXT
	)`).Error; err != nil {
		t.Fatalf("create projects table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE project_oauth_clients (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		redirect_uri TEXT NOT NULL,
		token TEXT NOT NULL,
		is_enabled NUMERIC NOT NULL DEFAULT 1,
		created_at DATETIME,
		updated_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create project_oauth_clients table: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO projects (name, token, bearer_auth_enabled, bearer_token) VALUES (?, ?, ?, ?)`,
		"ChatGPT Project",
		"project-token",
		true,
		"project-bearer-secret",
	).Error; err != nil {
		t.Fatalf("insert project: %v", err)
	}

	mux := http.NewServeMux()
	registerChatGPTOAuthRoutes(&app.RuntimeContext{DB: db}, mux)
	server := httptest.NewServer(mux)
	return server, db
}

func authorizeOAuthTestCode(t *testing.T, server *httptest.Server, callback, challenge string) string {
	t.Helper()

	authURL, err := url.Parse(server.URL + "/oauth/authorize")
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}
	query := authURL.Query()
	query.Set("response_type", "code")
	query.Set("client_id", "chatgpt")
	query.Set("redirect_uri", callback)
	query.Set("resource", server.URL+"/mcp/project-token")
	query.Set("scope", "mcp:connect")
	query.Set("state", "state-1")
	query.Set("code_challenge", challenge)
	query.Set("code_challenge_method", "S256")
	authURL.RawQuery = query.Encode()

	client := server.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	response, err := client.Get(authURL.String())
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusFound {
		t.Fatalf("authorize status = %d, want %d", response.StatusCode, http.StatusFound)
	}
	location := response.Header.Get("Location")
	redirect, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	code := redirect.Query().Get("code")
	if code == "" {
		t.Fatalf("redirect did not include code: %s", location)
	}
	if got := redirect.Query().Get("state"); got != "state-1" {
		t.Fatalf("state = %q, want state-1", got)
	}
	return code
}

func exchangeOAuthTestCode(t *testing.T, server *httptest.Server, code, callback, verifier, secret string) *http.Response {
	t.Helper()

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", callback)
	form.Set("client_id", "chatgpt")
	form.Set("client_secret", secret)
	form.Set("code_verifier", verifier)
	client := server.Client()
	tokenResponse, err := client.Post(server.URL+"/oauth/token", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	return tokenResponse
}
