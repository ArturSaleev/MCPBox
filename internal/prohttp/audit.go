package prohttp

import (
	"net/http"
	"strings"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/ArturSaleev/MCPBox/internal/proauth"
)

func logProtectedAudit(runtime *app.RuntimeContext, r *http.Request, entry app.AuditEntry) {
	if runtime == nil || runtime.LogAudit == nil || r == nil {
		return
	}

	principal := proauth.PrincipalFromContext(r.Context())
	if actor := strings.TrimSpace(proauth.FormatPrincipalActor(principal)); actor != "" {
		entry.Actor = actor
	} else if actor := strings.TrimSpace(entry.Actor); actor == "" {
		entry.Actor = proauth.ClientIPFromRequest(r)
	}

	extra := map[string]any{}
	if actorFields := proauth.PrincipalAuditFields(principal); len(actorFields) > 0 {
		extra["_actor"] = actorFields
	}
	if requestFields := proauth.RequestAuditFields(r.Context()); len(requestFields) > 0 {
		extra["_request"] = requestFields
	}
	entry.Detail = proauth.MergeAuditDetail(entry.Detail, extra)
	_ = runtime.LogAudit(r.Context(), entry)
}
