package proauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/ArturSaleev/MCPBox/app"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	db             *gorm.DB
	bootstrapToken string
	logAudit       func(ctx context.Context, entry app.AuditEntry) error
}

type CreateTokenInput struct {
	Name          string
	Scopes        []string
	ExpiresInDays int
}

type CreateSessionInput struct {
	UserID        uint
	Label         string
	AuthMethod    string
	ExpiresInDays int
}

type CreateUserInput struct {
	Email        string
	DisplayName  string
	AuthProvider string
	ExternalID   string
	RoleNames    []string
}

type UpdateUserRolesInput struct {
	UserID    uint
	RoleNames []string
}

type UpdateLocalPasswordInput struct {
	UserID   uint
	Password string
}

type ExternalUserInput struct {
	AuthProvider string
	ExternalID   string
	Email        string
	DisplayName  string
	DefaultRole  string
	AutoCreate   bool
}

type CreateLocalUserInput struct {
	Email       string
	DisplayName string
	Password    string
	RoleNames   []string
	IsBootstrap bool
}

type LocalLoginInput struct {
	Email         string
	Password      string
	Label         string
	ExpiresInDays int
}

type systemRoleSeed struct {
	Name        string
	DisplayName string
	Description string
	Scopes      []string
}

var defaultSystemRoles = []systemRoleSeed{
	{
		Name:        "admin",
		DisplayName: "Admin",
		Description: "Full Pro administration and token lifecycle management.",
		Scopes:      []string{"pro:admin"},
	},
	{
		Name:        "operator",
		DisplayName: "Operator",
		Description: "Operational access for day-to-day protected Pro actions.",
		Scopes:      []string{"pro:read", "pro:write"},
	},
	{
		Name:        "reader",
		DisplayName: "Reader",
		Description: "Read-only access to protected Pro endpoints.",
		Scopes:      []string{"pro:read"},
	},
}

func NewServiceFromRuntime(runtime *app.RuntimeContext) (*Service, error) {
	if runtime == nil || runtime.DB == nil {
		return nil, errors.New("runtime database is not available")
	}

	return &Service{
		db:             runtime.DB,
		bootstrapToken: strings.TrimSpace(os.Getenv("MCPBOX_PRO_BOOTSTRAP_TOKEN")),
		logAudit:       runtime.LogAudit,
	}, nil
}

func (s *Service) AutoMigrate(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("service database is not initialized")
	}
	if err := s.db.WithContext(ctx).AutoMigrate(
		&ProUser{},
		&ProRole{},
		&ProUserRole{},
		&ProSession{},
		&AgentToken{},
	); err != nil {
		return err
	}
	if err := s.ensureSystemRoles(ctx); err != nil {
		return err
	}
	if s.bootstrapToken != "" {
		if _, err := s.ensureBootstrapUser(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) HasLocalPasswordUsers(ctx context.Context) (bool, error) {
	if s == nil || s.db == nil {
		return false, errors.New("service database is not initialized")
	}

	var count int64
	if err := s.db.WithContext(ctx).
		Model(&ProUser{}).
		Where("auth_provider = ? AND trim(coalesce(password_hash, '')) <> ''", "local").
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *Service) ListTokens(ctx context.Context) ([]TokenRecord, error) {
	var rows []AgentToken
	if err := s.db.WithContext(ctx).Order("id desc").Find(&rows).Error; err != nil {
		return nil, err
	}

	records := make([]TokenRecord, 0, len(rows))
	for _, row := range rows {
		scopes, err := decodeScopes(row.ScopesJSON)
		if err != nil {
			return nil, err
		}
		records = append(records, TokenRecord{
			ID:         row.ID,
			Name:       row.Name,
			Scopes:     scopes,
			ExpiresAt:  row.ExpiresAt,
			LastUsedAt: row.LastUsedAt,
			RevokedAt:  row.RevokedAt,
			CreatedAt:  row.CreatedAt,
		})
	}

	return records, nil
}

func (s *Service) CreateToken(ctx context.Context, input CreateTokenInput) (*TokenRecord, string, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, "", errors.New("name is required")
	}

	scopes := normalizeScopes(input.Scopes)
	if len(scopes) == 0 {
		return nil, "", errors.New("at least one scope is required")
	}

	rawToken, err := newRawToken()
	if err != nil {
		return nil, "", err
	}

	scopesJSON, err := json.Marshal(scopes)
	if err != nil {
		return nil, "", err
	}

	row := AgentToken{
		Name:       name,
		TokenHash:  hashToken(rawToken),
		ScopesJSON: string(scopesJSON),
	}
	if input.ExpiresInDays > 0 {
		expiresAt := time.Now().Add(time.Duration(input.ExpiresInDays) * 24 * time.Hour)
		row.ExpiresAt = &expiresAt
	}

	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, "", err
	}

	record := &TokenRecord{
		ID:        row.ID,
		Name:      row.Name,
		Scopes:    scopes,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}

	return record, rawToken, nil
}

func (s *Service) RevokeToken(ctx context.Context, id uint) error {
	now := time.Now()
	return s.db.WithContext(ctx).
		Model(&AgentToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", &now).Error
}

func (s *Service) RevokeSession(ctx context.Context, id uint) error {
	now := time.Now()
	return s.db.WithContext(ctx).
		Model(&ProSession{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", &now).Error
}

func (s *Service) AuthenticateBearer(ctx context.Context, bearer string) (*Principal, error) {
	token := strings.TrimSpace(bearer)
	if token == "" {
		return nil, errors.New("missing bearer token")
	}

	if s.bootstrapToken != "" && token == s.bootstrapToken {
		bootstrapUser, err := s.ensureBootstrapUser(ctx)
		if err != nil {
			return nil, err
		}
		bootstrapEmail := ""
		if bootstrapUser != nil {
			bootstrapEmail = bootstrapUser.Email
		}
		s.audit(ctx, app.AuditEntry{
			Action: "token_used",
			Actor:  FormatActorLabel("bootstrap", bootstrapEmail),
			Detail: `{"principal":"bootstrap","bootstrap":true}`,
		})
		now := time.Now()
		if bootstrapUser != nil {
			_ = s.db.WithContext(ctx).
				Model(&ProUser{}).
				Where("id = ?", bootstrapUser.ID).
				Update("last_login_at", &now).Error
		}
		var userID *uint
		if bootstrapUser != nil {
			userID = &bootstrapUser.ID
		}
		return &Principal{
			UserID:      userID,
			Name:        "bootstrap",
			Email:       strings.TrimSpace(bootstrapEmail),
			Scopes:      []string{"pro:admin"},
			Roles:       []string{"admin"},
			AuthMethod:  "bootstrap_token",
			IsBootstrap: true,
		}, nil
	}

	principal, err := s.authenticateSession(ctx, token)
	if err == nil {
		return principal, nil
	}
	if err != nil && err.Error() != "session not found" {
		return nil, err
	}

	var row AgentToken
	if err := s.db.WithContext(ctx).Where("token_hash = ?", hashToken(token)).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("token not found")
		}
		return nil, err
	}
	if row.RevokedAt != nil {
		return nil, errors.New("token revoked")
	}
	if row.ExpiresAt != nil && row.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("token expired")
	}

	scopes, err := decodeScopes(row.ScopesJSON)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	_ = s.db.WithContext(ctx).Model(&AgentToken{}).Where("id = ?", row.ID).Update("last_used_at", &now).Error
	s.audit(ctx, app.AuditEntry{
		Action: "token_used",
		Actor:  row.Name,
		Detail: fmt.Sprintf(`{"token_id":%d,"principal":"%s","scopes":%s}`, row.ID, row.Name, row.ScopesJSON),
	})

	return &Principal{
		Name:       row.Name,
		Scopes:     scopes,
		AuthMethod: "agent_token",
	}, nil
}

func (s *Service) CreateSession(ctx context.Context, input CreateSessionInput) (*ProSession, string, error) {
	if s == nil || s.db == nil {
		return nil, "", errors.New("service database is not initialized")
	}
	if input.UserID == 0 {
		return nil, "", errors.New("user id is required")
	}

	var user ProUser
	if err := s.db.WithContext(ctx).First(&user, input.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", errors.New("user not found")
		}
		return nil, "", err
	}

	rawToken, err := newSessionToken()
	if err != nil {
		return nil, "", err
	}

	authMethod := strings.TrimSpace(input.AuthMethod)
	if authMethod == "" {
		authMethod = "local"
	}

	session := &ProSession{
		UserID:           input.UserID,
		SessionTokenHash: hashToken(rawToken),
		Label:            strings.TrimSpace(input.Label),
		AuthMethod:       authMethod,
	}
	if input.ExpiresInDays > 0 {
		expiresAt := time.Now().Add(time.Duration(input.ExpiresInDays) * 24 * time.Hour)
		session.ExpiresAt = &expiresAt
	}

	if err := s.db.WithContext(ctx).Create(session).Error; err != nil {
		return nil, "", err
	}

	return session, rawToken, nil
}

func (s *Service) CreateUser(ctx context.Context, input CreateUserInput) (*UserRecord, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("service database is not initialized")
	}

	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		return nil, errors.New("display name is required")
	}

	authProvider := strings.TrimSpace(input.AuthProvider)
	if authProvider == "" {
		authProvider = "local"
	}

	roleNames := normalizeScopes(input.RoleNames)
	if len(roleNames) == 0 {
		roleNames = []string{"reader"}
	}

	user := &ProUser{
		Email:        strings.TrimSpace(input.Email),
		DisplayName:  displayName,
		AuthProvider: authProvider,
		ExternalID:   strings.TrimSpace(input.ExternalID),
	}

	if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
		return nil, err
	}

	for _, roleName := range roleNames {
		if err := s.assignRoleByName(ctx, user.ID, roleName); err != nil {
			return nil, err
		}
	}

	roles, scopes, err := s.loadUserRolesAndScopes(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &UserRecord{
		ID:           user.ID,
		Email:        user.Email,
		DisplayName:  user.DisplayName,
		AuthProvider: user.AuthProvider,
		ExternalID:   user.ExternalID,
		IsBootstrap:  user.IsBootstrap,
		Roles:        roles,
		Scopes:       scopes,
		LastLoginAt:  user.LastLoginAt,
		DisabledAt:   user.DisabledAt,
		CreatedAt:    user.CreatedAt,
	}, nil
}

func (s *Service) CreateLocalUser(ctx context.Context, input CreateLocalUserInput) (*UserRecord, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("service database is not initialized")
	}

	email := normalizeEmail(input.Email)
	if email == "" {
		return nil, errors.New("email is required")
	}
	password := strings.TrimSpace(input.Password)
	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = email
	}

	var existing ProUser
	if err := s.db.WithContext(ctx).
		Where("auth_provider = ? AND lower(email) = ?", "local", strings.ToLower(email)).
		First(&existing).Error; err == nil {
		return nil, errors.New("local user with this email already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	roleNames := normalizeScopes(input.RoleNames)
	if len(roleNames) == 0 {
		roleNames = []string{"reader"}
	}

	user := &ProUser{
		Email:        email,
		DisplayName:  displayName,
		AuthProvider: "local",
		PasswordHash: passwordHash,
		IsBootstrap:  input.IsBootstrap,
	}
	if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
		return nil, err
	}
	for _, roleName := range roleNames {
		if err := s.assignRoleByName(ctx, user.ID, roleName); err != nil {
			return nil, err
		}
	}

	roles, scopes, err := s.loadUserRolesAndScopes(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &UserRecord{
		ID:           user.ID,
		Email:        user.Email,
		DisplayName:  user.DisplayName,
		AuthProvider: user.AuthProvider,
		ExternalID:   user.ExternalID,
		IsBootstrap:  user.IsBootstrap,
		Roles:        roles,
		Scopes:       scopes,
		LastLoginAt:  user.LastLoginAt,
		DisabledAt:   user.DisabledAt,
		CreatedAt:    user.CreatedAt,
	}, nil
}

func (s *Service) CreateOrUpdateLocalPasswordUser(ctx context.Context, input CreateLocalUserInput) (*UserRecord, bool, error) {
	if s == nil || s.db == nil {
		return nil, false, errors.New("service database is not initialized")
	}

	email := normalizeEmail(input.Email)
	if email == "" {
		return nil, false, errors.New("email is required")
	}
	password := strings.TrimSpace(input.Password)
	if len(password) < 8 {
		return nil, false, errors.New("password must be at least 8 characters")
	}

	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = email
	}

	roleNames := normalizeScopes(input.RoleNames)
	if len(roleNames) == 0 {
		roleNames = []string{"reader"}
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return nil, false, err
	}

	var user ProUser
	err = s.db.WithContext(ctx).
		Where("auth_provider = ? AND lower(email) = ?", "local", strings.ToLower(email)).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		record, createErr := s.CreateLocalUser(ctx, input)
		return record, true, createErr
	}
	if err != nil {
		return nil, false, err
	}

	updates := map[string]any{
		"email":         email,
		"display_name":  displayName,
		"password_hash": passwordHash,
	}
	if input.IsBootstrap {
		updates["is_bootstrap"] = true
	}
	if err := s.db.WithContext(ctx).Model(&ProUser{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
		return nil, false, err
	}

	record, err := s.UpdateUserRoles(ctx, UpdateUserRolesInput{
		UserID:    user.ID,
		RoleNames: roleNames,
	})
	if err != nil {
		return nil, false, err
	}
	return record, false, nil
}

func (s *Service) LoginLocal(ctx context.Context, input LocalLoginInput) (*UserRecord, *ProSession, string, error) {
	if s == nil || s.db == nil {
		return nil, nil, "", errors.New("service database is not initialized")
	}

	email := normalizeEmail(input.Email)
	if email == "" {
		return nil, nil, "", errors.New("email is required")
	}
	password := strings.TrimSpace(input.Password)
	if password == "" {
		return nil, nil, "", errors.New("password is required")
	}

	var user ProUser
	if err := s.db.WithContext(ctx).
		Where("auth_provider = ? AND lower(email) = ?", "local", strings.ToLower(email)).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, "", errors.New("invalid email or password")
		}
		return nil, nil, "", err
	}
	if user.DisabledAt != nil {
		return nil, nil, "", errors.New("user is disabled")
	}
	if strings.TrimSpace(user.PasswordHash) == "" {
		return nil, nil, "", errors.New("local password login is not configured for this user")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, "", errors.New("invalid email or password")
	}

	session, rawToken, err := s.CreateSession(ctx, CreateSessionInput{
		UserID:        user.ID,
		Label:         strings.TrimSpace(input.Label),
		AuthMethod:    "local_password",
		ExpiresInDays: input.ExpiresInDays,
	})
	if err != nil {
		return nil, nil, "", err
	}

	now := time.Now()
	_ = s.db.WithContext(ctx).Model(&ProUser{}).Where("id = ?", user.ID).Update("last_login_at", &now).Error

	roles, scopes, err := s.loadUserRolesAndScopes(ctx, user.ID)
	if err != nil {
		return nil, nil, "", err
	}
	record := &UserRecord{
		ID:           user.ID,
		Email:        user.Email,
		DisplayName:  user.DisplayName,
		AuthProvider: user.AuthProvider,
		ExternalID:   user.ExternalID,
		IsBootstrap:  user.IsBootstrap,
		Roles:        roles,
		Scopes:       scopes,
		LastLoginAt:  &now,
		DisabledAt:   user.DisabledAt,
		CreatedAt:    user.CreatedAt,
	}

	s.audit(ctx, app.AuditEntry{
		Action: "local_login",
		Actor:  FormatActorLabel(user.DisplayName, user.Email),
		Detail: fmt.Sprintf(`{"user_id":%d,"email":%q}`, user.ID, user.Email),
	})

	return record, session, rawToken, nil
}

func (s *Service) ListUsers(ctx context.Context) ([]UserRecord, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("service database is not initialized")
	}

	var users []ProUser
	if err := s.db.WithContext(ctx).Order("id asc").Find(&users).Error; err != nil {
		return nil, err
	}

	records := make([]UserRecord, 0, len(users))
	for _, user := range users {
		roles, scopes, err := s.loadUserRolesAndScopes(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		records = append(records, UserRecord{
			ID:           user.ID,
			Email:        user.Email,
			DisplayName:  user.DisplayName,
			AuthProvider: user.AuthProvider,
			ExternalID:   user.ExternalID,
			IsBootstrap:  user.IsBootstrap,
			Roles:        roles,
			Scopes:       scopes,
			LastLoginAt:  user.LastLoginAt,
			DisabledAt:   user.DisabledAt,
			CreatedAt:    user.CreatedAt,
		})
	}

	return records, nil
}

func (s *Service) UpdateUserRoles(ctx context.Context, input UpdateUserRolesInput) (*UserRecord, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("service database is not initialized")
	}
	if input.UserID == 0 {
		return nil, errors.New("user id is required")
	}

	roleNames := normalizeScopes(input.RoleNames)
	if len(roleNames) == 0 {
		return nil, errors.New("at least one role is required")
	}

	var user ProUser
	if err := s.db.WithContext(ctx).First(&user, input.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	if user.IsBootstrap && !slices.Contains(roleNames, "admin") {
		return nil, errors.New("bootstrap user must keep admin role")
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", user.ID).Delete(&ProUserRole{}).Error; err != nil {
			return err
		}
		for _, roleName := range roleNames {
			var role ProRole
			if err := tx.Where("name = ?", roleName).First(&role).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("unknown role: %s", roleName)
				}
				return err
			}
			link := &ProUserRole{
				UserID: user.ID,
				RoleID: role.ID,
			}
			if err := tx.Create(link).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	roles, scopes, err := s.loadUserRolesAndScopes(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &UserRecord{
		ID:           user.ID,
		Email:        user.Email,
		DisplayName:  user.DisplayName,
		AuthProvider: user.AuthProvider,
		ExternalID:   user.ExternalID,
		IsBootstrap:  user.IsBootstrap,
		Roles:        roles,
		Scopes:       scopes,
		LastLoginAt:  user.LastLoginAt,
		DisabledAt:   user.DisabledAt,
		CreatedAt:    user.CreatedAt,
	}, nil
}

func (s *Service) UpdateLocalPassword(ctx context.Context, input UpdateLocalPasswordInput) (*UserRecord, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("service database is not initialized")
	}
	if input.UserID == 0 {
		return nil, errors.New("user id is required")
	}

	password := strings.TrimSpace(input.Password)
	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	var user ProUser
	if err := s.db.WithContext(ctx).First(&user, input.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	if strings.TrimSpace(user.AuthProvider) != "local" {
		return nil, errors.New("password can only be changed for local users")
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).
		Model(&ProUser{}).
		Where("id = ?", user.ID).
		Update("password_hash", passwordHash).Error; err != nil {
		return nil, err
	}
	user.PasswordHash = passwordHash

	roles, scopes, err := s.loadUserRolesAndScopes(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &UserRecord{
		ID:           user.ID,
		Email:        user.Email,
		DisplayName:  user.DisplayName,
		AuthProvider: user.AuthProvider,
		ExternalID:   user.ExternalID,
		IsBootstrap:  user.IsBootstrap,
		Roles:        roles,
		Scopes:       scopes,
		LastLoginAt:  user.LastLoginAt,
		DisabledAt:   user.DisabledAt,
		CreatedAt:    user.CreatedAt,
	}, nil
}

func (s *Service) FindOrCreateExternalUser(ctx context.Context, input ExternalUserInput) (*UserRecord, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("service database is not initialized")
	}

	authProvider := strings.TrimSpace(input.AuthProvider)
	externalID := strings.TrimSpace(input.ExternalID)
	if authProvider == "" || externalID == "" {
		return nil, errors.New("external auth provider and external id are required")
	}

	var user ProUser
	err := s.db.WithContext(ctx).
		Where("auth_provider = ? AND external_id = ?", authProvider, externalID).
		First(&user).Error
	switch {
	case err == nil:
		updates := map[string]any{
			"email":        strings.TrimSpace(input.Email),
			"display_name": strings.TrimSpace(input.DisplayName),
		}
		if err := s.db.WithContext(ctx).Model(&ProUser{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
			return nil, err
		}
		user.Email = strings.TrimSpace(input.Email)
		user.DisplayName = strings.TrimSpace(input.DisplayName)
	case errors.Is(err, gorm.ErrRecordNotFound):
		if !input.AutoCreate {
			return nil, errors.New("sso user is not provisioned")
		}
		defaultRole := strings.TrimSpace(input.DefaultRole)
		if defaultRole == "" {
			defaultRole = "reader"
		}
		record, createErr := s.CreateUser(ctx, CreateUserInput{
			Email:        strings.TrimSpace(input.Email),
			DisplayName:  strings.TrimSpace(input.DisplayName),
			AuthProvider: authProvider,
			ExternalID:   externalID,
			RoleNames:    []string{defaultRole},
		})
		if createErr != nil {
			return nil, createErr
		}
		return record, nil
	default:
		return nil, err
	}

	if user.DisabledAt != nil {
		return nil, errors.New("user is disabled")
	}

	roles, scopes, err := s.loadUserRolesAndScopes(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return &UserRecord{
		ID:           user.ID,
		Email:        user.Email,
		DisplayName:  user.DisplayName,
		AuthProvider: user.AuthProvider,
		ExternalID:   user.ExternalID,
		IsBootstrap:  user.IsBootstrap,
		Roles:        roles,
		Scopes:       scopes,
		LastLoginAt:  user.LastLoginAt,
		DisabledAt:   user.DisabledAt,
		CreatedAt:    user.CreatedAt,
	}, nil
}

func (s *Service) DisableUser(ctx context.Context, userID uint) (*UserRecord, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("service database is not initialized")
	}
	if userID == 0 {
		return nil, errors.New("user id is required")
	}

	var user ProUser
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	if user.IsBootstrap {
		return nil, errors.New("bootstrap user cannot be disabled")
	}

	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&ProUser{}).Where("id = ?", user.ID).Update("disabled_at", &now).Error; err != nil {
		return nil, err
	}
	_ = s.db.WithContext(ctx).Model(&ProSession{}).Where("user_id = ? AND revoked_at IS NULL", user.ID).Update("revoked_at", &now).Error
	user.DisabledAt = &now

	roles, scopes, err := s.loadUserRolesAndScopes(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return &UserRecord{
		ID:           user.ID,
		Email:        user.Email,
		DisplayName:  user.DisplayName,
		AuthProvider: user.AuthProvider,
		ExternalID:   user.ExternalID,
		IsBootstrap:  user.IsBootstrap,
		Roles:        roles,
		Scopes:       scopes,
		LastLoginAt:  user.LastLoginAt,
		DisabledAt:   user.DisabledAt,
		CreatedAt:    user.CreatedAt,
	}, nil
}

func (s *Service) EnableUser(ctx context.Context, userID uint) (*UserRecord, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("service database is not initialized")
	}
	if userID == 0 {
		return nil, errors.New("user id is required")
	}

	var user ProUser
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	if err := s.db.WithContext(ctx).Model(&ProUser{}).Where("id = ?", user.ID).Update("disabled_at", nil).Error; err != nil {
		return nil, err
	}
	user.DisabledAt = nil

	roles, scopes, err := s.loadUserRolesAndScopes(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return &UserRecord{
		ID:           user.ID,
		Email:        user.Email,
		DisplayName:  user.DisplayName,
		AuthProvider: user.AuthProvider,
		ExternalID:   user.ExternalID,
		IsBootstrap:  user.IsBootstrap,
		Roles:        roles,
		Scopes:       scopes,
		LastLoginAt:  user.LastLoginAt,
		DisabledAt:   user.DisabledAt,
		CreatedAt:    user.CreatedAt,
	}, nil
}

func (s *Service) DeleteUser(ctx context.Context, userID uint) error {
	if s == nil || s.db == nil {
		return errors.New("service database is not initialized")
	}
	if userID == 0 {
		return errors.New("user id is required")
	}

	var user ProUser
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}
	if user.IsBootstrap {
		return errors.New("bootstrap user cannot be deleted")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if err := tx.Model(&ProSession{}).Where("user_id = ? AND revoked_at IS NULL", user.ID).Update("revoked_at", &now).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", user.ID).Delete(&ProUserRole{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&ProUser{}, user.ID).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *Service) ListRoles(ctx context.Context) ([]RoleRecord, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("service database is not initialized")
	}

	var roles []ProRole
	if err := s.db.WithContext(ctx).Order("id asc").Find(&roles).Error; err != nil {
		return nil, err
	}

	records := make([]RoleRecord, 0, len(roles))
	for _, role := range roles {
		records = append(records, RoleRecord{
			ID:          role.ID,
			Name:        role.Name,
			DisplayName: role.DisplayName,
			Description: role.Description,
			Scopes:      roleScopes(role.Name),
			IsSystem:    role.IsSystem,
			CreatedAt:   role.CreatedAt,
		})
	}

	return records, nil
}

func (s *Service) ListSessions(ctx context.Context) ([]SessionRecord, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("service database is not initialized")
	}

	var sessions []ProSession
	if err := s.db.WithContext(ctx).Order("id desc").Find(&sessions).Error; err != nil {
		return nil, err
	}

	userIDs := make([]uint, 0, len(sessions))
	for _, session := range sessions {
		userIDs = append(userIDs, session.UserID)
	}

	var users []ProUser
	if len(userIDs) > 0 {
		if err := s.db.WithContext(ctx).Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return nil, err
		}
	}
	usersByID := make(map[uint]ProUser, len(users))
	for _, user := range users {
		usersByID[user.ID] = user
	}

	records := make([]SessionRecord, 0, len(sessions))
	for _, session := range sessions {
		user, ok := usersByID[session.UserID]
		if !ok {
			continue
		}
		roles, scopes, err := s.loadUserRolesAndScopes(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		records = append(records, SessionRecord{
			ID:         session.ID,
			UserID:     session.UserID,
			UserName:   user.DisplayName,
			Label:      session.Label,
			AuthMethod: session.AuthMethod,
			Roles:      roles,
			Scopes:     scopes,
			ExpiresAt:  session.ExpiresAt,
			LastUsedAt: session.LastUsedAt,
			RevokedAt:  session.RevokedAt,
			CreatedAt:  session.CreatedAt,
		})
	}

	return records, nil
}

func (s *Service) ensureSystemRoles(ctx context.Context) error {
	for _, role := range defaultSystemRoles {
		row := ProRole{
			Name:        role.Name,
			DisplayName: role.DisplayName,
			Description: role.Description,
			IsSystem:    true,
		}
		if err := s.db.WithContext(ctx).
			Where("name = ?", role.Name).
			Assign(map[string]any{
				"display_name": row.DisplayName,
				"description":  row.Description,
				"is_system":    true,
			}).
			FirstOrCreate(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ensureBootstrapUser(ctx context.Context) (*ProUser, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("service database is not initialized")
	}

	user := &ProUser{}
	if err := s.db.WithContext(ctx).
		Where("auth_provider = ? AND external_id = ?", "bootstrap", "local-bootstrap-admin").
		Assign(map[string]any{
			"auth_provider": "bootstrap",
			"external_id":   "local-bootstrap-admin",
			"display_name":  "Bootstrap Admin",
			"is_bootstrap":  true,
		}).
		FirstOrCreate(user).Error; err != nil {
		return nil, err
	}
	if err := s.assignRoleByName(ctx, user.ID, "admin"); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) assignRoleByName(ctx context.Context, userID uint, roleName string) error {
	var role ProRole
	if err := s.db.WithContext(ctx).Where("name = ?", roleName).First(&role).Error; err != nil {
		return err
	}

	link := &ProUserRole{
		UserID: userID,
		RoleID: role.ID,
	}
	return s.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ?", userID, role.ID).
		FirstOrCreate(link).Error
}

func (s *Service) authenticateSession(ctx context.Context, token string) (*Principal, error) {
	var session ProSession
	if err := s.db.WithContext(ctx).Where("session_token_hash = ?", hashToken(token)).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("session not found")
		}
		return nil, err
	}
	if session.RevokedAt != nil {
		return nil, errors.New("session revoked")
	}
	if session.ExpiresAt != nil && session.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("session expired")
	}

	var user ProUser
	if err := s.db.WithContext(ctx).First(&user, session.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("session user not found")
		}
		return nil, err
	}
	if user.DisabledAt != nil {
		return nil, errors.New("user disabled")
	}

	roles, scopes, err := s.loadUserRolesAndScopes(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	_ = s.db.WithContext(ctx).Model(&ProSession{}).Where("id = ?", session.ID).Update("last_used_at", &now).Error
	_ = s.db.WithContext(ctx).Model(&ProUser{}).Where("id = ?", user.ID).Update("last_login_at", &now).Error

	s.audit(ctx, app.AuditEntry{
		Action: "session_used",
		Actor:  FormatActorLabel(user.DisplayName, user.Email),
		Detail: fmt.Sprintf(`{"session_id":%d,"user_id":%d,"roles":%s}`, session.ID, user.ID, encodeJSONArray(roles)),
	})

	return &Principal{
		UserID:     &user.ID,
		SessionID:  &session.ID,
		Name:       user.DisplayName,
		Email:      user.Email,
		Scopes:     scopes,
		Roles:      roles,
		AuthMethod: session.AuthMethod,
	}, nil
}

func (s *Service) loadUserRolesAndScopes(ctx context.Context, userID uint) ([]string, []string, error) {
	type roleRow struct {
		Name string
	}

	var rows []roleRow
	if err := s.db.WithContext(ctx).
		Table("pro_roles").
		Select("pro_roles.name").
		Joins("join pro_user_roles on pro_user_roles.role_id = pro_roles.id").
		Where("pro_user_roles.user_id = ?", userID).
		Order("pro_roles.name asc").
		Scan(&rows).Error; err != nil {
		return nil, nil, err
	}

	roles := make([]string, 0, len(rows))
	scopes := make([]string, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, row.Name)
		scopes = append(scopes, roleScopes(row.Name)...)
	}

	return normalizeScopes(roles), normalizeScopes(scopes), nil
}

func HasScope(principal *Principal, required string) bool {
	if principal == nil {
		return false
	}
	if slices.Contains(principal.Scopes, "pro:admin") {
		return true
	}
	return slices.Contains(principal.Scopes, required)
}

func newRawToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return "mcpboxpro_" + base64.RawURLEncoding.EncodeToString(buf), nil
}

func newSessionToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return "mcpboxpro_session_" + base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashPassword(raw string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(bytes), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func decodeScopes(raw string) ([]string, error) {
	var scopes []string
	if err := json.Unmarshal([]byte(raw), &scopes); err != nil {
		return nil, err
	}
	return normalizeScopes(scopes), nil
}

func normalizeScopes(scopes []string) []string {
	normalized := make([]string, 0, len(scopes))
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		normalized = append(normalized, scope)
	}
	slices.Sort(normalized)
	return normalized
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (s *Service) audit(ctx context.Context, entry app.AuditEntry) {
	if s == nil || s.logAudit == nil {
		return
	}
	extra := map[string]any{}
	if request := RequestAuditFields(ctx); len(request) > 0 {
		extra["_request"] = request
	}
	entry.Detail = MergeAuditDetail(entry.Detail, extra)
	_ = s.logAudit(ctx, entry)
}

func roleScopes(roleName string) []string {
	switch strings.TrimSpace(roleName) {
	case "admin":
		return []string{"pro:admin"}
	case "operator":
		return []string{"pro:read", "pro:write"}
	case "reader":
		return []string{"pro:read"}
	default:
		return nil
	}
}

func encodeJSONArray(items []string) string {
	payload, err := json.Marshal(normalizeScopes(items))
	if err != nil {
		return "[]"
	}
	return string(payload)
}
