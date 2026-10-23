package prohttp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/ArturSaleev/MCPBox/internal/proauth"
	"gorm.io/gorm"
)

const chatGPTOAuthCodeTTL = 10 * time.Minute

type projectOAuthRecord struct {
	ID                uint
	Name              string
	Token             string
	BearerAuthEnabled bool
	BearerToken       string
	OAuthRedirectURI  string
}

type projectOAuthClientRecord struct {
	ID          uint
	ProjectID   uint
	Name        string
	RedirectURI string
	Token       string
	IsEnabled   bool
}

type chatGPTOAuthCode struct {
	Code                 string
	ProjectToken         string
	ProjectOAuthClientID uint
	ClientID             string
	RedirectURI          string
	Resource             string
	Scope                string
	CodeChallenge        string
	CodeChallengeMethod  string
	ExpiresAt            time.Time
}

var chatGPTOAuthCodes = struct {
	sync.Mutex
	items map[string]chatGPTOAuthCode
}{
	items: make(map[string]chatGPTOAuthCode),
}

func registerChatGPTOAuthRoutes(runtime *app.RuntimeContext, mux *http.ServeMux) {
	protectedResourceHandler := func(w http.ResponseWriter, r *http.Request) {
		resource := strings.TrimSpace(r.URL.Query().Get("resource"))
		if resource == "" {
			wellKnownSuffix := strings.TrimPrefix(r.URL.EscapedPath(), "/.well-known/oauth-protected-resource")
			if strings.Trim(wellKnownSuffix, "/") != "" {
				resource = requestOrigin(r) + wellKnownSuffix
			}
		}

		projectToken := projectTokenFromOAuthResource(resource)
		project, err := loadProjectOAuthRecord(r.Context(), runtime, projectToken)
		if err != nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "project lookup failed"})
			return
		}
		// Protected-resource metadata must only be advertised for a project that
		// actually requires a bearer token. Returning generic metadata here makes
		// clients such as LibreChat incorrectly force OAuth for public MCP projects.
		if project == nil || !project.BearerAuthEnabled || strings.TrimSpace(project.BearerToken) == "" {
			http.NotFound(w, r)
			return
		}

		proauth.WriteJSON(w, http.StatusOK, map[string]any{
			"resource":                              resource,
			"authorization_servers":                 []string{requestOrigin(r)},
			"bearer_methods_supported":              []string{"header"},
			"resource_documentation":                requestOrigin(r) + "/",
			"scopes_supported":                      []string{"mcp:connect"},
			"mcpboxpro_project_oauth_token_hint":    "Create a named OAuth connection in MCPBox and use its generated token as the OAuth client secret in ChatGPT.",
			"mcpboxpro_project_oauth_resource_hint": "Use the public MCP URL, for example https://example.ngrok.app/mcp/<project-token>.",
		})
	}
	mux.HandleFunc("GET /.well-known/oauth-protected-resource", protectedResourceHandler)
	mux.HandleFunc("GET /.well-known/oauth-protected-resource/", protectedResourceHandler)

	mux.HandleFunc("GET /.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		origin := requestOrigin(r)
		proauth.WriteJSON(w, http.StatusOK, map[string]any{
			"issuer":                                             origin,
			"authorization_endpoint":                             origin + "/oauth/authorize",
			"token_endpoint":                                     origin + "/oauth/token",
			"response_types_supported":                           []string{"code"},
			"grant_types_supported":                              []string{"authorization_code"},
			"code_challenge_methods_supported":                   []string{"S256"},
			"token_endpoint_auth_methods_supported":              []string{"client_secret_basic", "client_secret_post"},
			"scopes_supported":                                   []string{"mcp:connect"},
			"client_id_metadata_document_supported":              false,
			"authorization_response_iss_parameter_supported":     true,
			"mcpboxpro_predefined_oauth_client_secret_supported": true,
		})
	})

	mux.HandleFunc("GET /oauth/authorize", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		authorizeDetail := map[string]any{
			"stage":                 "authorize",
			"response_type":         strings.TrimSpace(query.Get("response_type")),
			"client_id":             strings.TrimSpace(query.Get("client_id")),
			"has_redirect_uri":      strings.TrimSpace(query.Get("redirect_uri")) != "",
			"resource":              strings.TrimSpace(query.Get("resource")),
			"scope":                 normalizeOAuthScope(query.Get("scope")),
			"code_challenge_method": strings.TrimSpace(query.Get("code_challenge_method")),
			"has_code_challenge":    strings.TrimSpace(query.Get("code_challenge")) != "",
			"user_agent":            strings.TrimSpace(r.Header.Get("User-Agent")),
		}
		if query.Get("response_type") != "code" {
			logOAuthAudit(runtime, r.Context(), nil, "project_oauth_authorize_failed", authorizeDetail, "unsupported_response_type")
			writeOAuthRedirectOrError(w, r, query.Get("redirect_uri"), query.Get("state"), "unsupported_response_type", "response_type must be code")
			return
		}

		redirectURI := strings.TrimSpace(query.Get("redirect_uri"))
		clientID := strings.TrimSpace(query.Get("client_id"))
		codeChallenge := strings.TrimSpace(query.Get("code_challenge"))
		codeChallengeMethod := strings.TrimSpace(query.Get("code_challenge_method"))
		resource := strings.TrimSpace(query.Get("resource"))
		scope := normalizeOAuthScope(query.Get("scope"))
		if scope == "" {
			scope = "mcp:connect"
		}
		if redirectURI == "" || clientID == "" || codeChallenge == "" {
			logOAuthAudit(runtime, r.Context(), nil, "project_oauth_authorize_failed", authorizeDetail, "missing_redirect_client_or_challenge")
			writeOAuthRedirectOrError(w, r, redirectURI, query.Get("state"), "invalid_request", "redirect_uri, client_id and code_challenge are required")
			return
		}
		if codeChallengeMethod != "S256" {
			logOAuthAudit(runtime, r.Context(), nil, "project_oauth_authorize_failed", authorizeDetail, "invalid_code_challenge_method")
			writeOAuthRedirectOrError(w, r, redirectURI, query.Get("state"), "invalid_request", "code_challenge_method must be S256")
			return
		}

		projectToken := projectTokenFromOAuthResource(resource)
		if projectToken == "" {
			projectToken = strings.TrimSpace(query.Get("project_token"))
		}
		project, err := loadProjectOAuthRecord(r.Context(), runtime, projectToken)
		if err != nil {
			logOAuthAudit(runtime, r.Context(), nil, "project_oauth_authorize_failed", authorizeDetail, "project_lookup_failed:"+err.Error())
			writeOAuthRedirectOrError(w, r, redirectURI, query.Get("state"), "server_error", err.Error())
			return
		}
		if project == nil {
			logOAuthAudit(runtime, r.Context(), nil, "project_oauth_authorize_failed", authorizeDetail, "project_not_found")
			writeOAuthRedirectOrError(w, r, redirectURI, query.Get("state"), "invalid_request", "project token was not found in resource")
			return
		}
		authorizeDetail["project_id"] = project.ID
		authorizeDetail["project_name"] = project.Name
		authorizeDetail["project_token_preview"] = previewProjectToken(project.Token)
		if !project.BearerAuthEnabled || strings.TrimSpace(project.BearerToken) == "" {
			logOAuthAudit(runtime, r.Context(), &project.ID, "project_oauth_authorize_failed", authorizeDetail, "bearer_protection_disabled")
			writeOAuthRedirectOrError(w, r, redirectURI, query.Get("state"), "access_denied", "project bearer protection must be enabled before OAuth can issue a token")
			return
		}
		oauthClient, oauthClientCount, err := loadProjectOAuthClientByName(r.Context(), runtime, project.ID, clientID)
		if err != nil {
			logOAuthAudit(runtime, r.Context(), &project.ID, "project_oauth_authorize_failed", authorizeDetail, "oauth_client_lookup_failed:"+err.Error())
			writeOAuthRedirectOrError(w, r, redirectURI, query.Get("state"), "server_error", "OAuth client lookup failed")
			return
		}
		projectOAuthClientID := uint(0)
		configuredRedirectURI := project.OAuthRedirectURI
		if oauthClient != nil {
			if !oauthClient.IsEnabled {
				logOAuthAudit(runtime, r.Context(), &project.ID, "project_oauth_authorize_failed", authorizeDetail, "oauth_client_disabled")
				writeOAuthRedirectOrError(w, r, redirectURI, query.Get("state"), "access_denied", "OAuth client is disabled")
				return
			}
			projectOAuthClientID = oauthClient.ID
			configuredRedirectURI = oauthClient.RedirectURI
		} else if oauthClientCount > 0 {
			legacyRedirectURI := strings.TrimSpace(project.OAuthRedirectURI)
			if legacyRedirectURI == "" || redirectURI != legacyRedirectURI {
				logOAuthAudit(runtime, r.Context(), &project.ID, "project_oauth_authorize_failed", authorizeDetail, "oauth_client_not_found")
				writeOAuthRedirectOrError(w, r, redirectURI, query.Get("state"), "invalid_request", "OAuth client was not found")
				return
			}
		}
		if !oauthRedirectURIAllowed(redirectURI, configuredRedirectURI) {
			logOAuthAudit(runtime, r.Context(), &project.ID, "project_oauth_authorize_failed", authorizeDetail, "redirect_uri_mismatch")
			writeOAuthRedirectOrError(w, r, redirectURI, query.Get("state"), "invalid_request", "redirect_uri does not match the callback URL configured in the project")
			return
		}

		code, err := randomURLToken(32)
		if err != nil {
			logOAuthAudit(runtime, r.Context(), &project.ID, "project_oauth_authorize_failed", authorizeDetail, "code_generation_failed:"+err.Error())
			writeOAuthRedirectOrError(w, r, redirectURI, query.Get("state"), "server_error", err.Error())
			return
		}
		chatGPTOAuthCodes.Lock()
		chatGPTOAuthCodes.items[code] = chatGPTOAuthCode{
			Code:                 code,
			ProjectToken:         project.Token,
			ProjectOAuthClientID: projectOAuthClientID,
			ClientID:             clientID,
			RedirectURI:          redirectURI,
			Resource:             resource,
			Scope:                scope,
			CodeChallenge:        codeChallenge,
			CodeChallengeMethod:  codeChallengeMethod,
			ExpiresAt:            time.Now().Add(chatGPTOAuthCodeTTL),
		}
		chatGPTOAuthCodes.Unlock()
		authorizeDetail["issued_code"] = true
		authorizeDetail["expires_at"] = time.Now().Add(chatGPTOAuthCodeTTL).UTC().Format(time.RFC3339)
		logOAuthAudit(runtime, r.Context(), &project.ID, "project_oauth_authorize_succeeded", authorizeDetail, "")

		target, err := url.Parse(redirectURI)
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid redirect_uri"})
			return
		}
		values := target.Query()
		values.Set("code", code)
		if state := strings.TrimSpace(query.Get("state")); state != "" {
			values.Set("state", state)
		}
		values.Set("iss", requestOrigin(r))
		target.RawQuery = values.Encode()
		http.Redirect(w, r, target.String(), http.StatusFound)
	})

	mux.HandleFunc("POST /oauth/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			logOAuthAudit(runtime, r.Context(), nil, "project_oauth_token_failed", map[string]any{
				"stage":      "token",
				"user_agent": strings.TrimSpace(r.Header.Get("User-Agent")),
			}, "parse_form_failed:"+err.Error())
			writeOAuthTokenError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		tokenDetail := map[string]any{
			"stage":             "token",
			"grant_type":        strings.TrimSpace(r.PostForm.Get("grant_type")),
			"client_id":         strings.TrimSpace(r.PostForm.Get("client_id")),
			"has_redirect_uri":  strings.TrimSpace(r.PostForm.Get("redirect_uri")) != "",
			"has_code":          strings.TrimSpace(r.PostForm.Get("code")) != "",
			"has_code_verifier": strings.TrimSpace(r.PostForm.Get("code_verifier")) != "",
			"auth_method":       oauthTokenAuthMethod(r),
			"has_client_secret": clientSecretFromOAuthTokenRequest(r) != "",
			"user_agent":        strings.TrimSpace(r.Header.Get("User-Agent")),
			"basic_auth_client": clientIDFromBasicAuth(r),
			"content_type":      strings.TrimSpace(r.Header.Get("Content-Type")),
		}
		if r.PostForm.Get("grant_type") != "authorization_code" {
			logOAuthAudit(runtime, r.Context(), nil, "project_oauth_token_failed", tokenDetail, "unsupported_grant_type")
			writeOAuthTokenError(w, http.StatusBadRequest, "unsupported_grant_type", "grant_type must be authorization_code")
			return
		}

		code := strings.TrimSpace(r.PostForm.Get("code"))
		codeVerifier := strings.TrimSpace(r.PostForm.Get("code_verifier"))
		redirectURI := strings.TrimSpace(r.PostForm.Get("redirect_uri"))
		clientID := strings.TrimSpace(r.PostForm.Get("client_id"))
		clientSecret := clientSecretFromOAuthTokenRequest(r)
		if code == "" || codeVerifier == "" || redirectURI == "" {
			logOAuthAudit(runtime, r.Context(), nil, "project_oauth_token_failed", tokenDetail, "missing_code_verifier_or_redirect")
			writeOAuthTokenError(w, http.StatusBadRequest, "invalid_request", "code, redirect_uri and code_verifier are required")
			return
		}

		chatGPTOAuthCodes.Lock()
		record, ok := chatGPTOAuthCodes.items[code]
		chatGPTOAuthCodes.Unlock()
		if !ok || time.Now().After(record.ExpiresAt) {
			logOAuthAudit(runtime, r.Context(), nil, "project_oauth_token_failed", tokenDetail, "authorization_code_invalid_or_expired")
			writeOAuthTokenError(w, http.StatusBadRequest, "invalid_grant", "authorization code is invalid or expired")
			return
		}
		tokenDetail["stored_client_id"] = record.ClientID
		tokenDetail["resource"] = record.Resource
		tokenDetail["scope"] = record.Scope
		if clientID == "" {
			clientID = clientIDFromBasicAuth(r)
			tokenDetail["client_id"] = clientID
		}
		if clientID != record.ClientID || redirectURI != record.RedirectURI {
			logOAuthAudit(runtime, r.Context(), nil, "project_oauth_token_failed", tokenDetail, "client_id_or_redirect_mismatch")
			writeOAuthTokenError(w, http.StatusBadRequest, "invalid_grant", "client_id or redirect_uri does not match the authorization request")
			return
		}
		if !verifyPKCES256(record.CodeChallenge, codeVerifier) {
			logOAuthAudit(runtime, r.Context(), nil, "project_oauth_token_failed", tokenDetail, "pkce_verification_failed")
			writeOAuthTokenError(w, http.StatusBadRequest, "invalid_grant", "code_verifier does not match code_challenge")
			return
		}

		project, err := loadProjectOAuthRecord(r.Context(), runtime, record.ProjectToken)
		if err != nil {
			logOAuthAudit(runtime, r.Context(), nil, "project_oauth_token_failed", tokenDetail, "project_lookup_failed:"+err.Error())
			writeOAuthTokenError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		if project == nil || !project.BearerAuthEnabled || strings.TrimSpace(project.BearerToken) == "" {
			logOAuthAudit(runtime, r.Context(), nil, "project_oauth_token_failed", tokenDetail, "project_bearer_protection_unavailable")
			writeOAuthTokenError(w, http.StatusBadRequest, "invalid_grant", "project bearer protection is not available")
			return
		}
		tokenDetail["project_id"] = project.ID
		tokenDetail["project_name"] = project.Name
		tokenDetail["project_token_preview"] = previewProjectToken(project.Token)
		expectedSecret := project.BearerToken
		accessToken := project.BearerToken
		if record.ProjectOAuthClientID != 0 {
			oauthClient, clientErr := loadProjectOAuthClientByID(r.Context(), runtime, record.ProjectOAuthClientID)
			if clientErr != nil {
				logOAuthAudit(runtime, r.Context(), &project.ID, "project_oauth_token_failed", tokenDetail, "oauth_client_lookup_failed:"+clientErr.Error())
				writeOAuthTokenError(w, http.StatusInternalServerError, "server_error", "OAuth client lookup failed")
				return
			}
			if oauthClient == nil || !oauthClient.IsEnabled || oauthClient.ProjectID != project.ID || oauthClient.Name != clientID || oauthClient.RedirectURI != redirectURI {
				logOAuthAudit(runtime, r.Context(), &project.ID, "project_oauth_token_failed", tokenDetail, "oauth_client_invalid_or_disabled")
				writeOAuthTokenError(w, http.StatusUnauthorized, "invalid_client", "OAuth client is invalid or disabled")
				return
			}
			expectedSecret = oauthClient.Token
			accessToken = oauthClient.Token
			tokenDetail["oauth_client_name"] = oauthClient.Name
		} else {
		}
		if !constantTimeEqual(clientSecret, expectedSecret) {
			logOAuthAudit(runtime, r.Context(), &project.ID, "project_oauth_token_failed", tokenDetail, "client_secret_mismatch")
			writeOAuthTokenError(w, http.StatusUnauthorized, "invalid_client", "client secret must match the project bearer token")
			return
		}

		chatGPTOAuthCodes.Lock()
		delete(chatGPTOAuthCodes.items, code)
		chatGPTOAuthCodes.Unlock()
		logOAuthAudit(runtime, r.Context(), &project.ID, "project_oauth_token_succeeded", tokenDetail, "")

		proauth.WriteJSON(w, http.StatusOK, map[string]any{
			"access_token": accessToken,
			"token_type":   "Bearer",
			"scope":        record.Scope,
		})
	})
}

func loadProjectOAuthRecord(ctx context.Context, runtime *app.RuntimeContext, token string) (*projectOAuthRecord, error) {
	token = strings.TrimSpace(token)
	if runtime == nil || runtime.DB == nil {
		return nil, errors.New("runtime database is not configured")
	}
	if token == "" {
		return nil, nil
	}

	var row struct {
		ID                uint
		Name              string
		Token             string
		BearerAuthEnabled bool
		BearerToken       string
		OAuthRedirectURI  string
	}
	err := runtime.DB.WithContext(ctx).
		Table("projects").
		Select("id, name, token, bearer_auth_enabled, bearer_token, oauth_redirect_uri").
		Where("token = ?", token).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &projectOAuthRecord{
		ID:                row.ID,
		Name:              row.Name,
		Token:             row.Token,
		BearerAuthEnabled: row.BearerAuthEnabled,
		BearerToken:       row.BearerToken,
		OAuthRedirectURI:  row.OAuthRedirectURI,
	}, nil
}

func loadProjectOAuthClientByName(ctx context.Context, runtime *app.RuntimeContext, projectID uint, name string) (*projectOAuthClientRecord, int64, error) {
	if runtime == nil || runtime.DB == nil {
		return nil, 0, errors.New("runtime database is not configured")
	}
	var count int64
	if err := runtime.DB.WithContext(ctx).Table("project_oauth_clients").Where("project_id = ?", projectID).Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if count == 0 {
		return nil, 0, nil
	}
	var row projectOAuthClientRecord
	err := runtime.DB.WithContext(ctx).
		Table("project_oauth_clients").
		Select("id, project_id, name, redirect_uri, token, is_enabled").
		Where("project_id = ? AND name = ?", projectID, strings.TrimSpace(name)).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, count, nil
	}
	if err != nil {
		return nil, count, err
	}
	return &row, count, nil
}

func loadProjectOAuthClientByID(ctx context.Context, runtime *app.RuntimeContext, clientID uint) (*projectOAuthClientRecord, error) {
	if runtime == nil || runtime.DB == nil {
		return nil, errors.New("runtime database is not configured")
	}
	var row projectOAuthClientRecord
	err := runtime.DB.WithContext(ctx).
		Table("project_oauth_clients").
		Select("id, project_id, name, redirect_uri, token, is_enabled").
		Where("id = ?", clientID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func oauthRedirectURIAllowed(redirectURI, configured string) bool {
	redirectURI = strings.TrimSpace(redirectURI)
	configured = strings.TrimSpace(configured)
	if redirectURI == "" {
		return false
	}
	if configured == "" {
		return true
	}
	return redirectURI == configured
}

func logOAuthAudit(runtime *app.RuntimeContext, ctx context.Context, projectID *uint, action string, detail map[string]any, outcome string) {
	if runtime == nil || runtime.LogAudit == nil {
		return
	}
	payload := make(map[string]any, len(detail)+1)
	for key, value := range detail {
		payload[key] = value
	}
	if strings.TrimSpace(outcome) != "" {
		payload["outcome"] = strings.TrimSpace(outcome)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		raw = []byte(fmt.Sprintf(`{"outcome":%q}`, strings.TrimSpace(outcome)))
	}
	_ = runtime.LogAudit(ctx, app.AuditEntry{
		ProjectID: projectID,
		Action:    action,
		Actor:     "chatgpt_oauth",
		Detail:    string(raw),
	})
}

func previewProjectToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if len(token) <= 8 {
		return token
	}
	return token[:4] + "..." + token[len(token)-4:]
}

func projectTokenFromOAuthResource(resource string) string {
	resource = strings.TrimSpace(resource)
	if resource == "" {
		return ""
	}
	parsed, err := url.Parse(resource)
	if err != nil {
		return ""
	}
	cleanedPath := strings.Trim(parsed.EscapedPath(), "/")
	parts := strings.Split(cleanedPath, "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "mcp" || parts[i] == "connect" {
			if token, err := url.PathUnescape(parts[i+1]); err == nil {
				return strings.TrimSpace(token)
			}
			return strings.TrimSpace(parts[i+1])
		}
	}
	return ""
}

func requestOrigin(r *http.Request) string {
	if configured := strings.TrimRight(strings.TrimSpace(os.Getenv("MCPBOX_PUBLIC_MCP_URL")), "/"); configured != "" {
		if parsed, err := url.Parse(configured); err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == "" && (parsed.Path == "" || parsed.Path == "/") {
			return parsed.Scheme + "://" + parsed.Host
		}
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwardedProto != "" {
		scheme = forwardedProto
	}
	host := strings.TrimSpace(r.Host)
	if forwardedHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
		host = forwardedHost
	}
	if host == "" {
		return ""
	}
	return scheme + "://" + host
}

func writeOAuthRedirectOrError(w http.ResponseWriter, r *http.Request, redirectURI, state, code, description string) {
	redirectURI = strings.TrimSpace(redirectURI)
	if redirectURI == "" {
		proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": code, "error_description": description})
		return
	}
	target, err := url.Parse(redirectURI)
	if err != nil {
		proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": code, "error_description": description})
		return
	}
	values := target.Query()
	values.Set("error", code)
	if description != "" {
		values.Set("error_description", description)
	}
	if strings.TrimSpace(state) != "" {
		values.Set("state", state)
	}
	target.RawQuery = values.Encode()
	http.Redirect(w, r, target.String(), http.StatusFound)
}

func writeOAuthTokenError(w http.ResponseWriter, status int, code, description string) {
	proauth.WriteJSON(w, status, map[string]string{"error": code, "error_description": description})
}

func clientSecretFromOAuthTokenRequest(r *http.Request) string {
	if _, password, ok := r.BasicAuth(); ok {
		return strings.TrimSpace(password)
	}
	return strings.TrimSpace(r.PostForm.Get("client_secret"))
}

func clientIDFromBasicAuth(r *http.Request) string {
	username, _, ok := r.BasicAuth()
	if !ok {
		return ""
	}
	return strings.TrimSpace(username)
}

func oauthTokenAuthMethod(r *http.Request) string {
	if _, _, ok := r.BasicAuth(); ok {
		return "client_secret_basic"
	}
	if strings.TrimSpace(r.PostForm.Get("client_secret")) != "" {
		return "client_secret_post"
	}
	return "none"
}

func verifyPKCES256(expectedChallenge, verifier string) bool {
	sum := sha256.Sum256([]byte(verifier))
	actual := base64.RawURLEncoding.EncodeToString(sum[:])
	return constantTimeEqual(actual, expectedChallenge)
}

func constantTimeEqual(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" || len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func randomURLToken(size int) (string, error) {
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func normalizeOAuthScope(scope string) string {
	fields := strings.Fields(scope)
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
}
