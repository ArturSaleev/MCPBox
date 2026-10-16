package proauth

import (
	"context"
	"encoding/json"
	"strings"
)

func FormatActorLabel(name, email string) string {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	switch {
	case name != "" && email != "":
		if strings.EqualFold(name, email) {
			return email
		}
		return name + " <" + email + ">"
	case name != "":
		return name
	default:
		return email
	}
}

func FormatPrincipalActor(principal *Principal) string {
	if principal == nil {
		return ""
	}
	return FormatActorLabel(principal.Name, principal.Email)
}

func PrincipalAuditFields(principal *Principal) map[string]any {
	if principal == nil {
		return nil
	}
	fields := map[string]any{}
	if principal.UserID != nil {
		fields["user_id"] = *principal.UserID
	}
	if principal.SessionID != nil {
		fields["session_id"] = *principal.SessionID
	}
	if value := strings.TrimSpace(principal.Name); value != "" {
		fields["name"] = value
	}
	if value := strings.TrimSpace(principal.Email); value != "" {
		fields["email"] = value
	}
	if value := strings.TrimSpace(principal.AuthMethod); value != "" {
		fields["auth_method"] = value
	}
	if len(principal.Roles) > 0 {
		fields["roles"] = append([]string(nil), principal.Roles...)
	}
	if principal.IsBootstrap {
		fields["bootstrap"] = true
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func RequestAuditFields(ctx context.Context) map[string]any {
	metadata := RequestMetadataFromContext(ctx)
	fields := map[string]any{}
	if value := strings.TrimSpace(metadata.ClientIP); value != "" {
		fields["ip"] = value
	}
	if value := strings.TrimSpace(metadata.UserAgent); value != "" {
		fields["user_agent"] = value
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func MergeAuditDetail(detail string, extra map[string]any) string {
	detail = strings.TrimSpace(detail)
	if len(extra) == 0 {
		return detail
	}

	payload := map[string]any{}
	if detail != "" {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(detail), &decoded); err == nil && decoded != nil {
			for key, value := range decoded {
				payload[key] = value
			}
		} else {
			payload["message"] = detail
		}
	}
	for key, value := range extra {
		if value == nil {
			continue
		}
		payload[key] = value
	}
	if len(payload) == 0 {
		return detail
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return detail
	}
	return string(raw)
}
