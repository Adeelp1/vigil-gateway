package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

// RequestID injects a unique X-Request-ID into every request. If the client
// already sent one we honour it (useful for distributed tracing). The ID is
// propagated both in context (for structured logging) and as a response header.
func RequestID(next Handler) Handler {
	return HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = generateID()
		}
		// Store in context so downstream handlers can grab it without
		// threading the http.Request header everywhere.
		ctx := context.WithValue(r.Context(), RequestIDKey, id)
		r = r.WithContext(ctx)

		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}

// generateID produces a 16-byte (32-char hex) random ID.
// crypto/rand is used over math/rand: we never want predictable IDs.
func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%v", time.Now())
	}
	return hex.EncodeToString(b)
}
