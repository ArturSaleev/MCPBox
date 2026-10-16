package proauth

import (
	"context"
	"net"
	"net/http"
	"strings"
)

type requestMetadataContextKey struct{}

type RequestMetadata struct {
	ClientIP  string
	UserAgent string
}

func WithRequestMetadata(ctx context.Context, r *http.Request) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if r == nil {
		return ctx
	}
	return context.WithValue(ctx, requestMetadataContextKey{}, RequestMetadata{
		ClientIP:  ClientIPFromRequest(r),
		UserAgent: strings.TrimSpace(r.UserAgent()),
	})
}

func RequestMetadataFromContext(ctx context.Context) RequestMetadata {
	if ctx == nil {
		return RequestMetadata{}
	}
	metadata, _ := ctx.Value(requestMetadataContextKey{}).(RequestMetadata)
	return metadata
}

func ClientIPFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		for _, part := range parts {
			if candidate := strings.TrimSpace(part); candidate != "" {
				return candidate
			}
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}
