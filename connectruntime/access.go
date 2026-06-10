package connectruntime

import (
	"context"
	"net/http"
	"strings"
)

type accessContextKey struct{}

// Access carries validated runtime access data for a single connect request.
type Access struct {
	Actor             string
	PublicConnectPath string
	ProjectID         string
	ProjectSlug       string
	SessionID         string
	SubjectID         string
	Role              string
	Scopes            []string
	Context           map[string]string
	Metadata          map[string]string
	UpstreamHeaders   map[string]string
}

type Project struct {
	ID                   uint
	Token                string
	IdentityVerification bool
}

type AuthorizationError struct {
	StatusCode int
	Message    string
}

func (e *AuthorizationError) Error() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(e.Message)
}

type ProjectAuthorizer func(r *http.Request, project Project) (*Access, error)

func WithAccess(ctx context.Context, access *Access) context.Context {
	if access == nil {
		return ctx
	}
	return context.WithValue(ctx, accessContextKey{}, access)
}

func FromContext(ctx context.Context) *Access {
	access, _ := ctx.Value(accessContextKey{}).(*Access)
	return access
}

func (a *Access) HeaderValue(key string) string {
	if a == nil {
		return ""
	}
	return strings.TrimSpace(a.UpstreamHeaders[key])
}
