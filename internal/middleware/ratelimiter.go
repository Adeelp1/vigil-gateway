package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/Adeelp1/vigil-gateway/config"
)

type bucket struct {
	tokens     float64   // current token count
	lastRefill time.Time // when we last added tokens
}

var (
	buckets   = map[string]*bucket{}
	bucketsMu sync.Mutex
)

func RateLimiter(cfg config.Config) Middleware {
	return func(h Handler) Handler {
		return HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sub, _ := r.Context().Value(JWTSubjectKey).(string)
			if sub == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			bucketsMu.Lock()
			b, exists := buckets[sub]
			if !exists {
				b = &bucket{
					tokens:     float64(cfg.RateLimitBurst),
					lastRefill: time.Now(),
				}
				buckets[sub] = b
			}

			// Refill tokens based on elapsed time
			now := time.Now()
			elapsed := now.Sub(b.lastRefill).Seconds()
			b.tokens += elapsed * float64(cfg.RateLimitRPS)
			if b.tokens > float64(cfg.RateLimitBurst) {
				b.tokens = float64(cfg.RateLimitBurst)
			}

			if b.tokens < 1 {
				bucketsMu.Unlock()
				w.WriteHeader(http.StatusTooManyRequests)
				return // rate limit exceeded
			}

			b.tokens--
			b.lastRefill = now
			bucketsMu.Unlock()

			h.ServeHTTP(w, r)
		})
	}
}
