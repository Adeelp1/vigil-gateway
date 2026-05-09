package middleware

import (
	"net/http"
)

// Handler is the core unit that every middleware and the final upstream
// handler must satisfy.
type Handler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// HandlerFunc adapts a plain function to Handler.
type HandlerFunc func(http.ResponseWriter, *http.Request)

func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) { f(w, r) }

// Middleware wraps a Handler to produce a new Handler.
type Middleware func(Handler) Handler

// Chain composes a slice of middlewares into a single Handler.
// Middlewares execute left-to-right (outermost first).
//
// Example: Chain(Logger, RequestID)(upstream)
//
//	→ Logger( RequestID( upstream ) )
func Chain(mws ...Middleware) Middleware {
	return func(next Handler) Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			next = mws[i](next)
		}
		return next
	}
}
