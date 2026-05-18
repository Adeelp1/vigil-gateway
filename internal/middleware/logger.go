package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/Adeelp1/vigil-gateway/config"
	"github.com/Adeelp1/vigil-gateway/internal/store"
	"github.com/Adeelp1/vigil-gateway/internal/stream"
)

// responseRecorder wraps http.ResponseWritter to capture the status and
// bytes written so we can log them after the handler returns
type responseRecorder struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.bytesWritten += n
	return n, err
}

// Logger emits a structured log line for every request using the stdlib
func Logger(cfg config.Config, rdb store.Redis) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := newResponseRecorder(w)

			next.ServeHTTP(rec, r)

			ip, _, _ := net.SplitHostPort(r.RemoteAddr)

			go func() {
				stream.Publish(context.Background(), rdb, cfg, stream.Event{
					IP:         ip,
					Sub:        GetJWTSubject(r.Context()),
					Method:     r.Method,
					Path:       r.URL.Path,
					Status:     rec.statusCode,
					DurationMS: time.Since(start).Milliseconds(),
					UserAgent:  r.Header.Get("User-Agent"),
					Timestamp:  time.Now().Unix(),
				})
			}()

			slog.Info("request",
				"id", GetRequestID(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.statusCode,
				"bytes", rec.bytesWritten,
				"remote", r.RemoteAddr,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}
