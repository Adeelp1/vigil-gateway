package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Adeelp1/vigil-gateway/config"
	"github.com/golang-jwt/jwt/v5"
)

const JWTSubjectKey contextKey = "jwt_subject"

func JWTAuth(cfg config.Config) Middleware {
	return func(h Handler) Handler {
		return HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
				// Ensure the signing method is HMAC
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method")
				}

				return []byte(cfg.JWTSecret), nil
			})
			if err != nil {
				slog.Error("jwt parse error", "err", err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if !token.Valid {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			sub, err := token.Claims.GetSubject()
			if err != nil || sub == "" {
				slog.Error("token claims Subject error", "err", err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), JWTSubjectKey, sub)
			r = r.WithContext(ctx)

			h.ServeHTTP(w, r)

		})
	}
}
