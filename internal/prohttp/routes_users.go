package prohttp

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/ArturSaleev/MCPBox/internal/proauth"
)

type createUserRequest struct {
	Email        string   `json:"email"`
	DisplayName  string   `json:"display_name"`
	AuthProvider string   `json:"auth_provider"`
	ExternalID   string   `json:"external_id"`
	RoleNames    []string `json:"role_names"`
}

type updateUserRolesRequest struct {
	RoleNames []string `json:"role_names"`
}

type updateUserPasswordRequest struct {
	Password string `json:"password"`
}

func registerUserRoutes(runtime *app.RuntimeContext, mux *http.ServeMux, auth *proauth.Service) {
	mux.HandleFunc("GET /api/pro/users", proauth.Middleware(auth, "pro:read", func(w http.ResponseWriter, r *http.Request) {
		records, err := auth.ListUsers(r.Context())
		if err != nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		proauth.WriteJSON(w, http.StatusOK, records)
	}))

	mux.HandleFunc("POST /api/pro/users", proauth.Middleware(auth, "pro:admin", func(w http.ResponseWriter, r *http.Request) {
		var req createUserRequest
		if err := decodeJSONBody(r, &req); err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		record, err := auth.CreateUser(r.Context(), proauth.CreateUserInput{
			Email:        req.Email,
			DisplayName:  req.DisplayName,
			AuthProvider: req.AuthProvider,
			ExternalID:   req.ExternalID,
			RoleNames:    req.RoleNames,
		})
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		logProtectedAudit(runtime, r, app.AuditEntry{
			Action: "user_created",
			Detail: buildUserAuditDetail(record),
		})

		proauth.WriteJSON(w, http.StatusCreated, record)
	}))

	mux.HandleFunc("PUT /api/pro/users/", proauth.Middleware(auth, "pro:admin", func(w http.ResponseWriter, r *http.Request) {
		id, err := userIDFromPath(r.URL.Path)
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		var req updateUserRolesRequest
		if err := decodeJSONBody(r, &req); err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		record, err := auth.UpdateUserRoles(r.Context(), proauth.UpdateUserRolesInput{
			UserID:    id,
			RoleNames: req.RoleNames,
		})
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		logProtectedAudit(runtime, r, app.AuditEntry{
			Action: "user_roles_updated",
			Detail: buildUserAuditDetail(record),
		})

		proauth.WriteJSON(w, http.StatusOK, record)
	}))

	mux.HandleFunc("POST /api/pro/users/{id}/password", proauth.Middleware(auth, "pro:admin", func(w http.ResponseWriter, r *http.Request) {
		id, err := userIDFromPath(r.URL.Path)
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		var req updateUserPasswordRequest
		if err := decodeJSONBody(r, &req); err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		record, err := auth.UpdateLocalPassword(r.Context(), proauth.UpdateLocalPasswordInput{
			UserID:   id,
			Password: req.Password,
		})
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		logProtectedAudit(runtime, r, app.AuditEntry{
			Action: "user_password_updated",
			Detail: buildUserAuditDetail(record),
		})

		proauth.WriteJSON(w, http.StatusOK, record)
	}))

	mux.HandleFunc("POST /api/pro/users/{id}/disable", proauth.Middleware(auth, "pro:admin", func(w http.ResponseWriter, r *http.Request) {
		id, err := userIDFromPath(r.URL.Path)
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		record, err := auth.DisableUser(r.Context(), id)
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		logProtectedAudit(runtime, r, app.AuditEntry{
			Action: "user_disabled",
			Detail: buildUserAuditDetail(record),
		})

		proauth.WriteJSON(w, http.StatusOK, record)
	}))

	mux.HandleFunc("POST /api/pro/users/{id}/enable", proauth.Middleware(auth, "pro:admin", func(w http.ResponseWriter, r *http.Request) {
		id, err := userIDFromPath(r.URL.Path)
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		record, err := auth.EnableUser(r.Context(), id)
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		logProtectedAudit(runtime, r, app.AuditEntry{
			Action: "user_enabled",
			Detail: buildUserAuditDetail(record),
		})

		proauth.WriteJSON(w, http.StatusOK, record)
	}))

	mux.HandleFunc("DELETE /api/pro/users/", proauth.Middleware(auth, "pro:admin", func(w http.ResponseWriter, r *http.Request) {
		id, err := userIDFromPath(r.URL.Path)
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := auth.DeleteUser(r.Context(), id); err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		logProtectedAudit(runtime, r, app.AuditEntry{
			Action: "user_deleted",
			Detail: fmt.Sprintf(`{"user_id":%d}`, id),
		})

		proauth.WriteJSON(w, http.StatusOK, map[string]any{"id": id, "status": "deleted"})
	}))
}

func userIDFromPath(path string) (uint, error) {
	prefix := "/api/pro/users/"
	if !strings.HasPrefix(path, prefix) {
		return 0, fmt.Errorf("invalid user path")
	}

	rawID := strings.TrimSpace(strings.TrimPrefix(path, prefix))
	if slash := strings.Index(rawID, "/"); slash >= 0 {
		rawID = rawID[:slash]
	}
	id, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid user id")
	}
	return uint(id), nil
}

func buildUserAuditDetail(record *proauth.UserRecord) string {
	if record == nil {
		return "{}"
	}

	payload, err := marshalJSON(map[string]any{
		"user_id":      record.ID,
		"display_name": record.DisplayName,
		"email":        record.Email,
		"roles":        record.Roles,
	})
	if err != nil {
		return fmt.Sprintf(`{"user_id":%d}`, record.ID)
	}
	return payload
}
