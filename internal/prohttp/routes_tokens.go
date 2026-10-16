package prohttp

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/ArturSaleev/MCPBox/internal/proauth"
)

type createTokenRequest struct {
	Name          string   `json:"name"`
	Scopes        []string `json:"scopes"`
	ExpiresInDays int      `json:"expires_in_days"`
}

func registerTokenRoutes(runtime *app.RuntimeContext, mux *http.ServeMux, auth *proauth.Service) {
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
		if err := decodeJSONBody(r, &req); err != nil {
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
		logProtectedAudit(runtime, r, app.AuditEntry{
			Action: "token_created",
			Detail: buildTokenAuditDetail(record),
		})

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
		logProtectedAudit(runtime, r, app.AuditEntry{
			Action: "token_revoked",
			Detail: fmt.Sprintf(`{"token_id":%d}`, id),
		})
		proauth.WriteJSON(w, http.StatusOK, map[string]any{"id": id, "status": "revoked"})
	}))
}

func tokenIDFromPath(path string) (uint, error) {
	prefix := "/api/pro/tokens/"
	if !strings.HasPrefix(path, prefix) {
		return 0, fmt.Errorf("invalid token path")
	}

	rawID := strings.TrimSpace(strings.TrimPrefix(path, prefix))
	id, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid token id")
	}
	return uint(id), nil
}

func buildTokenAuditDetail(record *proauth.TokenRecord) string {
	if record == nil {
		return "{}"
	}

	payload, err := marshalJSON(map[string]any{
		"token_id":   record.ID,
		"name":       record.Name,
		"scopes":     record.Scopes,
		"expires_at": formatTimePtr(record.ExpiresAt),
	})
	if err != nil {
		return fmt.Sprintf(`{"token_id":%d}`, record.ID)
	}
	return payload
}
