package middleware

import "context"

// GetRequestID retrieves the request ID from context, returning "" if absent.
func GetRequestID(ctx context.Context) string {
	id, ok := ctx.Value(RequestIDKey).(string)
	if !ok {
		return ""
	}
	return id
}

func GetJWTSubject(ctx context.Context) string {
	sub, ok := ctx.Value(JWTSubjectKey).(string)
	if !ok {
		return ""
	}
	return sub
}
