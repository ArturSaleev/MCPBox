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
	return s.db.WithContext(ctx).AutoMigrate(&AgentToken{})
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

func (s *Service) AuthenticateBearer(ctx context.Context, bearer string) (*Principal, error) {
	token := strings.TrimSpace(bearer)
	if token == "" {
		return nil, errors.New("missing bearer token")
	}

	if s.bootstrapToken != "" && token == s.bootstrapToken {
		s.audit(ctx, app.AuditEntry{
			Action: "token_used",
			Actor:  "bootstrap",
			Detail: `{"principal":"bootstrap","bootstrap":true}`,
		})
		return &Principal{
			Name:        "bootstrap",
			Scopes:      []string{"pro:admin"},
			IsBootstrap: true,
		}, nil
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
		Name:   row.Name,
		Scopes: scopes,
	}, nil
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

func (s *Service) audit(ctx context.Context, entry app.AuditEntry) {
	if s == nil || s.logAudit == nil {
		return
	}
	_ = s.logAudit(ctx, entry)
}
