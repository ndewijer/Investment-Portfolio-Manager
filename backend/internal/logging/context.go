package logging

import "context"

type ctxKey int

const (
	ctxKeyRequestID ctxKey = iota
	ctxKeyIP
	ctxKeyUserAgent
)

// WithRequestInfo returns a context enriched with HTTP request metadata.
func WithRequestInfo(ctx context.Context, requestID, ip, userAgent string) context.Context {
	ctx = context.WithValue(ctx, ctxKeyRequestID, requestID)
	ctx = context.WithValue(ctx, ctxKeyIP, ip)
	ctx = context.WithValue(ctx, ctxKeyUserAgent, userAgent)
	return ctx
}

// RequestIDFromContext extracts the request ID from context.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyRequestID).(string); ok {
		return v
	}
	return ""
}

// IPFromContext extracts the client IP from context.
func IPFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyIP).(string); ok {
		return v
	}
	return ""
}

// UserAgentFromContext extracts the user agent from context.
func UserAgentFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyUserAgent).(string); ok {
		return v
	}
	return ""
}
