package prohttp

import (
	"net/http"

	"github.com/ArturSaleev/MCPBox/internal/proauth"
	"github.com/ArturSaleev/MCPBox/internal/prosso"
)

type localLoginRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	Label         string `json:"label"`
	ExpiresInDays int    `json:"expires_in_days"`
}

func registerAuthRoutes(mux *http.ServeMux, auth *proauth.Service, sso *prosso.Service) {
	mux.HandleFunc("GET /api/pro/sso/config", func(w http.ResponseWriter, _ *http.Request) {
		if sso == nil {
			proauth.WriteJSON(w, http.StatusOK, prosso.PublicConfig{Enabled: false})
			return
		}
		proauth.WriteJSON(w, http.StatusOK, sso.PublicConfig())
	})

	mux.HandleFunc("POST /api/pro/login", func(w http.ResponseWriter, r *http.Request) {
		if auth == nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "auth service is not configured"})
			return
		}

		var req localLoginRequest
		if err := decodeJSONBody(r, &req); err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		ctx := proauth.WithRequestMetadata(r.Context(), r)
		user, session, token, err := auth.LoginLocal(ctx, proauth.LocalLoginInput{
			Email:         req.Email,
			Password:      req.Password,
			Label:         req.Label,
			ExpiresInDays: req.ExpiresInDays,
		})
		if err != nil {
			proauth.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}

		proauth.WriteJSON(w, http.StatusOK, map[string]any{
			"token":   token,
			"user":    user,
			"session": buildSessionRecordFromLogin(user, session),
		})
	})

	mux.HandleFunc("GET /api/pro/sso/start", func(w http.ResponseWriter, r *http.Request) {
		if sso == nil || !sso.Enabled() {
			proauth.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "sso is not configured"})
			return
		}
		authURL, err := sso.StartAuth(proauth.WithRequestMetadata(r.Context(), r), r.URL.Query().Get("return_to"))
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		http.Redirect(w, r, authURL, http.StatusFound)
	})

	mux.HandleFunc("GET /api/pro/sso/callback", func(w http.ResponseWriter, r *http.Request) {
		if sso == nil || !sso.Enabled() {
			proauth.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "sso is not configured"})
			return
		}
		if providerError := r.URL.Query().Get("error"); providerError != "" {
			description := r.URL.Query().Get("error_description")
			message := providerError
			if description != "" {
				message = providerError + ": " + description
			}
			http.Redirect(w, r, prosso.BuildCallbackErrorRedirectURL("/?pro_view=settings", message), http.StatusFound)
			return
		}
		result, err := sso.HandleCallback(proauth.WithRequestMetadata(r.Context(), r), r.URL.Query().Get("state"), r.URL.Query().Get("code"))
		if err != nil {
			http.Redirect(w, r, prosso.BuildCallbackErrorRedirectURL("/?pro_view=settings", err.Error()), http.StatusFound)
			return
		}
		http.Redirect(w, r, prosso.BuildCallbackRedirectURL(result.ReturnTo, result.SessionToken), http.StatusFound)
	})

	mux.HandleFunc("GET /api/pro/auth/me", proauth.Middleware(auth, "pro:read", func(w http.ResponseWriter, r *http.Request) {
		principal := proauth.PrincipalFromContext(r.Context())
		var userID any
		if principal != nil && principal.UserID != nil {
			userID = *principal.UserID
		}
		var sessionID any
		if principal != nil && principal.SessionID != nil {
			sessionID = *principal.SessionID
		}
		proauth.WriteJSON(w, http.StatusOK, map[string]any{
			"name":         principal.Name,
			"scopes":       principal.Scopes,
			"roles":        principal.Roles,
			"user_id":      userID,
			"session_id":   sessionID,
			"auth_method":  principal.AuthMethod,
			"is_bootstrap": principal.IsBootstrap,
		})
	}))
}
