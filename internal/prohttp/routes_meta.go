package prohttp

import (
	"net/http"

	"github.com/ArturSaleev/MCPBox/internal/proauth"
	"github.com/ArturSaleev/MCPBox/internal/prosso"
)

func registerMetaRoutes(mux *http.ServeMux, auth *proauth.Service, sso *prosso.Service) {
	mux.HandleFunc("GET /api/pro/meta", proauth.Middleware(auth, "pro:read", func(w http.ResponseWriter, _ *http.Request) {
		features := []string{
			"auth",
			"local_auth",
			"agent_tokens",
			"policybox_config",
			"ragbox_config",
			"storage_config",
		}
		if sso != nil && sso.Enabled() {
			features = append(features, "sso")
		}
		proauth.WriteJSON(w, http.StatusOK, map[string]any{
			"edition_id":   "mcpbox",
			"edition_name": "MCPBox",
			"features":     features,
		})
	}))

	mux.HandleFunc("GET /api/pro/roles", proauth.Middleware(auth, "pro:read", func(w http.ResponseWriter, r *http.Request) {
		records, err := auth.ListRoles(r.Context())
		if err != nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		proauth.WriteJSON(w, http.StatusOK, records)
	}))
}

func requestsAdminScope(scopes []string) bool {
	for _, scope := range scopes {
		if scope == "pro:admin" {
			return true
		}
	}
	return false
}
