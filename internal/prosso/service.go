package prosso

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/ArturSaleev/MCPBox/internal/proauth"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type Service struct {
	db       *gorm.DB
	logAudit func(ctx context.Context, entry app.AuditEntry) error
	auth     *proauth.Service
	http     *http.Client

	config    Config
	mu        sync.Mutex
	discovery *Discovery
}

type Config struct {
	ProviderName        string
	IssuerURL           string
	ClientID            string
	ClientSecret        string
	RedirectURL         string
	AllowedHostedDomain string
	DefaultRole         string
	SessionDays         int
	AutoCreateUsers     bool
	Scopes              []string
}

type Discovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
	Issuer                string `json:"issuer"`
}

type PublicConfig struct {
	Enabled             bool     `json:"enabled"`
	ProviderName        string   `json:"provider_name,omitempty"`
	IssuerURL           string   `json:"issuer_url,omitempty"`
	RedirectURL         string   `json:"redirect_url,omitempty"`
	StartURL            string   `json:"start_url,omitempty"`
	AllowedHostedDomain string   `json:"allowed_hosted_domain,omitempty"`
	DefaultRole         string   `json:"default_role,omitempty"`
	SessionDays         int      `json:"session_days,omitempty"`
	AutoCreateUsers     bool     `json:"auto_create_users"`
	Scopes              []string `json:"scopes,omitempty"`
}

type CallbackResult struct {
	SessionToken string
	ReturnTo     string
	User         *proauth.UserRecord
}

type userInfo struct {
	Sub               string `json:"sub"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	HostedDomain      string `json:"hd"`
}

func NewServiceFromRuntime(runtime *app.RuntimeContext, auth *proauth.Service) (*Service, error) {
	if runtime == nil || runtime.DB == nil {
		return nil, errors.New("runtime database is not available")
	}
	return &Service{
		db:       runtime.DB,
		logAudit: runtime.LogAudit,
		auth:     auth,
		http:     http.DefaultClient,
		config:   loadConfigFromEnv(),
	}, nil
}

func IsConfiguredFromEnv() bool {
	config := loadConfigFromEnv()
	return config.Enabled()
}

func (c Config) Enabled() bool {
	return strings.TrimSpace(c.IssuerURL) != "" &&
		strings.TrimSpace(c.ClientID) != "" &&
		strings.TrimSpace(c.ClientSecret) != "" &&
		strings.TrimSpace(c.RedirectURL) != ""
}

func (s *Service) Enabled() bool {
	return s != nil && s.config.Enabled()
}

func (s *Service) PublicConfig() PublicConfig {
	if s == nil || !s.Enabled() {
		return PublicConfig{Enabled: false}
	}
	return PublicConfig{
		Enabled:             true,
		ProviderName:        s.config.ProviderName,
		IssuerURL:           s.config.IssuerURL,
		RedirectURL:         s.config.RedirectURL,
		StartURL:            "/api/pro/sso/start",
		AllowedHostedDomain: s.config.AllowedHostedDomain,
		DefaultRole:         s.config.DefaultRole,
		SessionDays:         s.config.SessionDays,
		AutoCreateUsers:     s.config.AutoCreateUsers,
		Scopes:              append([]string(nil), s.config.Scopes...),
	}
}

func (s *Service) AutoMigrate(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("sso database is not initialized")
	}
	return s.db.WithContext(ctx).AutoMigrate(&OIDCState{})
}

func (s *Service) StartAuth(ctx context.Context, returnTo string) (string, error) {
	if !s.Enabled() {
		return "", errors.New("sso is not configured")
	}
	discovery, err := s.discover(ctx)
	if err != nil {
		return "", err
	}
	stateValue, err := randomURLToken(32)
	if err != nil {
		return "", err
	}
	state := &OIDCState{
		State:     stateValue,
		ReturnTo:  sanitizeReturnTo(returnTo),
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if err := s.db.WithContext(ctx).Create(state).Error; err != nil {
		return "", err
	}
	options := make([]oauth2.AuthCodeOption, 0, 2)
	if strings.TrimSpace(s.config.AllowedHostedDomain) != "" {
		options = append(options, oauth2.SetAuthURLParam("hd", strings.TrimSpace(s.config.AllowedHostedDomain)))
	}
	options = append(options, oauth2.SetAuthURLParam("prompt", "select_account"))
	return s.oauthConfig(discovery).AuthCodeURL(stateValue, options...), nil
}

func (s *Service) HandleCallback(ctx context.Context, stateValue, code string) (*CallbackResult, error) {
	if !s.Enabled() {
		return nil, errors.New("sso is not configured")
	}
	stateValue = strings.TrimSpace(stateValue)
	code = strings.TrimSpace(code)
	if stateValue == "" || code == "" {
		return nil, errors.New("missing state or code")
	}

	var state OIDCState
	if err := s.db.WithContext(ctx).Where("state = ?", stateValue).First(&state).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid sso state")
		}
		return nil, err
	}
	defer func() {
		_ = s.db.WithContext(ctx).Delete(&OIDCState{}, state.ID).Error
	}()

	if state.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("expired sso state")
	}

	discovery, err := s.discover(ctx)
	if err != nil {
		return nil, err
	}
	token, err := s.oauthConfig(discovery).Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	info, err := s.fetchUserInfo(ctx, discovery, token)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(info.Sub) == "" {
		return nil, errors.New("oidc provider did not return subject")
	}
	if s.config.AllowedHostedDomain != "" && !strings.EqualFold(strings.TrimSpace(info.HostedDomain), strings.TrimSpace(s.config.AllowedHostedDomain)) {
		return nil, fmt.Errorf("sso account must belong to hosted domain %q", s.config.AllowedHostedDomain)
	}

	displayName := strings.TrimSpace(info.Name)
	if displayName == "" {
		displayName = strings.TrimSpace(info.PreferredUsername)
	}
	if displayName == "" {
		displayName = strings.TrimSpace(info.Email)
	}
	if displayName == "" {
		displayName = "SSO User"
	}

	user, err := s.auth.FindOrCreateExternalUser(ctx, proauth.ExternalUserInput{
		AuthProvider: s.providerKey(discovery),
		ExternalID:   strings.TrimSpace(info.Sub),
		Email:        strings.TrimSpace(info.Email),
		DisplayName:  displayName,
		DefaultRole:  s.config.DefaultRole,
		AutoCreate:   s.config.AutoCreateUsers,
	})
	if err != nil {
		return nil, err
	}

	_, sessionToken, err := s.auth.CreateSession(ctx, proauth.CreateSessionInput{
		UserID:        user.ID,
		Label:         fmt.Sprintf("%s SSO", s.config.ProviderName),
		AuthMethod:    "oidc",
		ExpiresInDays: s.config.SessionDays,
	})
	if err != nil {
		return nil, err
	}

	if s.logAudit != nil {
		payload, _ := json.Marshal(map[string]any{
			"user_id":       user.ID,
			"display_name":  user.DisplayName,
			"email":         user.Email,
			"auth_provider": s.providerKey(discovery),
		})
		_ = s.logAudit(ctx, app.AuditEntry{
			Action: "sso_login",
			Actor:  proauth.FormatActorLabel(user.DisplayName, user.Email),
			Detail: string(payload),
		})
	}

	return &CallbackResult{
		SessionToken: sessionToken,
		ReturnTo:     state.ReturnTo,
		User:         user,
	}, nil
}

func (s *Service) fetchUserInfo(ctx context.Context, discovery *Discovery, token *oauth2.Token) (*userInfo, error) {
	if discovery == nil || strings.TrimSpace(discovery.UserInfoEndpoint) == "" {
		return nil, errors.New("oidc userinfo endpoint is missing")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discovery.UserInfoEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("userinfo request failed with status %d", resp.StatusCode)
	}

	var info userInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (s *Service) oauthConfig(discovery *Discovery) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.config.ClientID,
		ClientSecret: s.config.ClientSecret,
		RedirectURL:  s.config.RedirectURL,
		Scopes:       append([]string(nil), s.config.Scopes...),
		Endpoint: oauth2.Endpoint{
			AuthURL:  discovery.AuthorizationEndpoint,
			TokenURL: discovery.TokenEndpoint,
		},
	}
}

func (s *Service) discover(ctx context.Context) (*Discovery, error) {
	s.mu.Lock()
	cached := s.discovery
	s.mu.Unlock()
	if cached != nil {
		return cached, nil
	}

	discoveryURL := strings.TrimRight(s.config.IssuerURL, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("oidc discovery failed with status %d", resp.StatusCode)
	}

	var discovery Discovery
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return nil, err
	}
	if strings.TrimSpace(discovery.AuthorizationEndpoint) == "" || strings.TrimSpace(discovery.TokenEndpoint) == "" {
		return nil, errors.New("oidc discovery is incomplete")
	}

	s.mu.Lock()
	s.discovery = &discovery
	s.mu.Unlock()
	return &discovery, nil
}

func (s *Service) providerKey(discovery *Discovery) string {
	issuer := s.config.IssuerURL
	if discovery != nil && strings.TrimSpace(discovery.Issuer) != "" {
		issuer = discovery.Issuer
	}
	return "oidc:" + strings.TrimSpace(issuer)
}

func loadConfigFromEnv() Config {
	scopes := strings.Fields(strings.TrimSpace(os.Getenv("MCPBOX_PRO_OIDC_SCOPES")))
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}

	defaultRole := strings.TrimSpace(os.Getenv("MCPBOX_PRO_OIDC_DEFAULT_ROLE"))
	if defaultRole == "" {
		defaultRole = "reader"
	}

	sessionDays := 30
	if raw := strings.TrimSpace(os.Getenv("MCPBOX_PRO_OIDC_SESSION_DAYS")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			sessionDays = parsed
		}
	}

	autoCreate := true
	if raw := strings.TrimSpace(os.Getenv("MCPBOX_PRO_OIDC_AUTO_CREATE_USERS")); raw != "" {
		autoCreate = raw == "1" || strings.EqualFold(raw, "true") || strings.EqualFold(raw, "yes")
	}

	providerName := strings.TrimSpace(os.Getenv("MCPBOX_PRO_OIDC_PROVIDER_NAME"))
	if providerName == "" {
		providerName = "OIDC"
	}

	return Config{
		ProviderName:        providerName,
		IssuerURL:           strings.TrimSpace(os.Getenv("MCPBOX_PRO_OIDC_ISSUER_URL")),
		ClientID:            strings.TrimSpace(os.Getenv("MCPBOX_PRO_OIDC_CLIENT_ID")),
		ClientSecret:        strings.TrimSpace(os.Getenv("MCPBOX_PRO_OIDC_CLIENT_SECRET")),
		RedirectURL:         strings.TrimSpace(os.Getenv("MCPBOX_PRO_OIDC_REDIRECT_URL")),
		AllowedHostedDomain: strings.TrimSpace(os.Getenv("MCPBOX_PRO_OIDC_ALLOWED_HD")),
		DefaultRole:         defaultRole,
		SessionDays:         sessionDays,
		AutoCreateUsers:     autoCreate,
		Scopes:              scopes,
	}
}

func sanitizeReturnTo(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "/?pro_view=pro"
	}
	if !strings.HasPrefix(value, "/") {
		return "/?pro_view=pro"
	}
	return value
}

func randomURLToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func BuildCallbackRedirectURL(returnTo, sessionToken string) string {
	target := sanitizeReturnTo(returnTo)
	base, err := url.Parse(target)
	if err != nil {
		return "/?pro_view=pro"
	}
	query := base.Query()
	query.Set("pro_session_token", sessionToken)
	query.Set("pro_view", "pro")
	base.RawQuery = query.Encode()
	return base.String()
}

func BuildCallbackErrorRedirectURL(returnTo, message string) string {
	target := sanitizeReturnTo(returnTo)
	base, err := url.Parse(target)
	if err != nil {
		return "/?pro_view=pro"
	}
	query := base.Query()
	query.Set("pro_view", "pro")
	query.Set("pro_sso_error", message)
	base.RawQuery = query.Encode()
	return base.String()
}
