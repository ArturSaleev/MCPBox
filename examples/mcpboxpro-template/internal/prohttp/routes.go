package prohttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/ArturSaleev/mcpboxpro/internal/proauth"
)

type createTokenRequest struct {
	Name          string   `json:"name"`
	Scopes        []string `json:"scopes"`
	ExpiresInDays int      `json:"expires_in_days"`
}

func RegisterRoutes(runtime *app.RuntimeContext, mux *http.ServeMux, auth *proauth.Service) {
	_ = runtime

	mux.HandleFunc("GET /api/pro/meta", proauth.Middleware(auth, "pro:read", func(w http.ResponseWriter, _ *http.Request) {
		proauth.WriteJSON(w, http.StatusOK, map[string]any{
			"edition_id":   "pro",
			"edition_name": "MCPBox Pro",
			"features": []string{
				"auth",
				"agent_tokens",
			},
		})
	}))

	mux.HandleFunc("GET /api/pro/auth/me", proauth.Middleware(auth, "pro:read", func(w http.ResponseWriter, r *http.Request) {
		principal := proauth.PrincipalFromContext(r.Context())
		proauth.WriteJSON(w, http.StatusOK, map[string]any{
			"name":         principal.Name,
			"scopes":       principal.Scopes,
			"is_bootstrap": principal.IsBootstrap,
		})
	}))

	mux.HandleFunc("GET /api/pro/tokens", proauth.Middleware(auth, "pro:read", func(w http.ResponseWriter, r *http.Request) {
		records, err := auth.ListTokens(r.Context())
		if err != nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		proauth.WriteJSON(w, http.StatusOK, records)
	}))

	mux.HandleFunc("POST /api/pro/tokens", proauth.Middleware(auth, "pro:write", func(w http.ResponseWriter, r *http.Request) {
		var req createTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		principal := proauth.PrincipalFromContext(r.Context())
		if principal == nil {
			proauth.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing principal"})
			return
		}
		if requestsAdminScope(req.Scopes) && !proauth.HasScope(principal, "pro:admin") {
			proauth.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "admin scope can only be issued by pro:admin"})
			return
		}

		record, rawToken, err := auth.CreateToken(r.Context(), proauth.CreateTokenInput{
			Name:          req.Name,
			Scopes:        req.Scopes,
			ExpiresInDays: req.ExpiresInDays,
		})
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if runtime != nil && runtime.LogAudit != nil {
			_ = runtime.LogAudit(r.Context(), app.AuditEntry{
				Action: "token_created",
				Actor:  principal.Name,
				Detail: buildTokenAuditDetail(record),
			})
		}

		proauth.WriteJSON(w, http.StatusCreated, map[string]any{
			"token":  rawToken,
			"record": record,
		})
	}))

	mux.HandleFunc("DELETE /api/pro/tokens/", proauth.Middleware(auth, "pro:admin", func(w http.ResponseWriter, r *http.Request) {
		id, err := tokenIDFromPath(r.URL.Path)
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := auth.RevokeToken(r.Context(), id); err != nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		principal := proauth.PrincipalFromContext(r.Context())
		if runtime != nil && runtime.LogAudit != nil {
			_ = runtime.LogAudit(r.Context(), app.AuditEntry{
				Action: "token_revoked",
				Actor:  principal.Name,
				Detail: fmt.Sprintf(`{"token_id":%d}`, id),
			})
		}
		proauth.WriteJSON(w, http.StatusOK, map[string]any{"id": id, "status": "revoked"})
	}))
}

func tokenIDFromPath(path string) (uint, error) {
	prefix := "/api/pro/tokens/"
	if !strings.HasPrefix(path, prefix) {
		return 0, errors.New("invalid token path")
	}

	rawID := strings.TrimSpace(strings.TrimPrefix(path, prefix))
	id, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil {
		return 0, errors.New("invalid token id")
	}
	return uint(id), nil
}

func buildTokenAuditDetail(record *proauth.TokenRecord) string {
	if record == nil {
		return "{}"
	}

	expiresAt := ""
	if record.ExpiresAt != nil {
		expiresAt = record.ExpiresAt.UTC().Format(time.RFC3339)
	}

	payload, err := json.Marshal(map[string]any{
		"token_id":   record.ID,
		"name":       record.Name,
		"scopes":     record.Scopes,
		"expires_at": expiresAt,
	})
	if err != nil {
		return fmt.Sprintf(`{"token_id":%d}`, record.ID)
	}
	return string(payload)
}

func requestsAdminScope(scopes []string) bool {
	for _, scope := range scopes {
		if strings.TrimSpace(scope) == "pro:admin" {
			return true
		}
	}
	return false
}
