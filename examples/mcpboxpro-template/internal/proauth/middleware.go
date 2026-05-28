package proauth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type principalContextKey struct{}

func Middleware(service *Service, requiredScope string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "auth service is not configured"})
			return
		}

		principal, err := service.AuthenticateBearer(r.Context(), bearerTokenFromRequest(r))
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}
		if requiredScope != "" && !HasScope(principal, requiredScope) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "missing required scope"})
			return
		}

		next(w, r.WithContext(context.WithValue(r.Context(), principalContextKey{}, principal)))
	}
}

func PrincipalFromContext(ctx context.Context) *Principal {
	principal, _ := ctx.Value(principalContextKey{}).(*Principal)
	return principal
}

func bearerTokenFromRequest(r *http.Request) string {
	raw := strings.TrimSpace(r.Header.Get("Authorization"))
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(raw), "bearer ") {
		return strings.TrimSpace(raw[7:])
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	writeJSON(w, status, payload)
}
