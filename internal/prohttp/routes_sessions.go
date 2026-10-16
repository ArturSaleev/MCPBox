package prohttp

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/ArturSaleev/MCPBox/internal/proauth"
)

type createSessionRequest struct {
	UserID        uint   `json:"user_id"`
	Label         string `json:"label"`
	AuthMethod    string `json:"auth_method"`
	ExpiresInDays int    `json:"expires_in_days"`
}

func registerSessionRoutes(runtime *app.RuntimeContext, mux *http.ServeMux, auth *proauth.Service) {
	mux.HandleFunc("GET /api/pro/sessions", proauth.Middleware(auth, "pro:read", func(w http.ResponseWriter, r *http.Request) {
		records, err := auth.ListSessions(r.Context())
		if err != nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		proauth.WriteJSON(w, http.StatusOK, records)
	}))

	mux.HandleFunc("POST /api/pro/sessions", proauth.Middleware(auth, "pro:admin", func(w http.ResponseWriter, r *http.Request) {
		var req createSessionRequest
		if err := decodeJSONBody(r, &req); err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		session, rawToken, err := auth.CreateSession(r.Context(), proauth.CreateSessionInput{
			UserID:        req.UserID,
			Label:         req.Label,
			AuthMethod:    req.AuthMethod,
			ExpiresInDays: req.ExpiresInDays,
		})
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		var record *proauth.SessionRecord
		records, listErr := auth.ListSessions(r.Context())
		if listErr == nil {
			for _, item := range records {
				if item.ID == session.ID {
					copyItem := item
					record = &copyItem
					break
				}
			}
		}

		logProtectedAudit(runtime, r, app.AuditEntry{
			Action: "session_created",
			Detail: buildSessionAuditDetail(record, session),
		})

		proauth.WriteJSON(w, http.StatusCreated, map[string]any{
			"token":  rawToken,
			"record": record,
		})
	}))

	mux.HandleFunc("DELETE /api/pro/sessions/", proauth.Middleware(auth, "pro:admin", func(w http.ResponseWriter, r *http.Request) {
		id, err := sessionIDFromPath(r.URL.Path)
		if err != nil {
			proauth.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := auth.RevokeSession(r.Context(), id); err != nil {
			proauth.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		logProtectedAudit(runtime, r, app.AuditEntry{
			Action: "session_revoked",
			Detail: fmt.Sprintf(`{"session_id":%d}`, id),
		})
		proauth.WriteJSON(w, http.StatusOK, map[string]any{"id": id, "status": "revoked"})
	}))
}

func sessionIDFromPath(path string) (uint, error) {
	prefix := "/api/pro/sessions/"
	if !strings.HasPrefix(path, prefix) {
		return 0, fmt.Errorf("invalid session path")
	}

	rawID := strings.TrimSpace(strings.TrimPrefix(path, prefix))
	id, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid session id")
	}
	return uint(id), nil
}

func buildSessionAuditDetail(record *proauth.SessionRecord, session *proauth.ProSession) string {
	if record != nil {
		payload, err := marshalJSON(map[string]any{
			"session_id":  record.ID,
			"user_id":     record.UserID,
			"user_name":   record.UserName,
			"auth_method": record.AuthMethod,
			"roles":       record.Roles,
			"expires_at":  record.ExpiresAt,
		})
		if err == nil {
			return payload
		}
	}
	if session != nil {
		return fmt.Sprintf(`{"session_id":%d,"user_id":%d}`, session.ID, session.UserID)
	}
	return "{}"
}

func buildSessionRecordFromLogin(user *proauth.UserRecord, session *proauth.ProSession) *proauth.SessionRecord {
	if user == nil || session == nil {
		return nil
	}
	return &proauth.SessionRecord{
		ID:         session.ID,
		UserID:     user.ID,
		UserName:   user.DisplayName,
		Label:      session.Label,
		AuthMethod: session.AuthMethod,
		Roles:      append([]string(nil), user.Roles...),
		Scopes:     append([]string(nil), user.Scopes...),
		ExpiresAt:  session.ExpiresAt,
		LastUsedAt: session.LastUsedAt,
		RevokedAt:  session.RevokedAt,
		CreatedAt:  session.CreatedAt,
	}
}
