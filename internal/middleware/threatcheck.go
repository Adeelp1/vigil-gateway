package middleware

import (
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/Adeelp1/vigil-gateway/internal/scoring"
	"github.com/Adeelp1/vigil-gateway/internal/store"
	"github.com/redis/go-redis/v9"
)

func ThreatCheck(rbd store.Redis) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			remoteAdd := r.RemoteAddr

			ip, _, err := net.SplitHostPort(remoteAdd)
			if err != nil {
				slog.Info("ThreatCheck", "cannot extract IP", err)
				return
			}

			record, err := rbd.Get(r.Context(), ip)
			if err != nil {
				if errors.Is(err, redis.Nil) {
					next.ServeHTTP(w, r)
					return
				}

				slog.Error("redis get failed", "ip", ip, "err", err)
				next.ServeHTTP(w, r)
				return
			}

			secondsElapsed := float64(time.Now().Unix() - record.WrittenAt)
			decayed := scoring.Decayed(record.Score, secondsElapsed)
			tier := scoring.Evaluate(decayed).String()

			switch tier {
			case scoring.TierPass.String():
				next.ServeHTTP(w, r)
			case scoring.TierRateLimit.String():
				w.WriteHeader(429)
				return
			case scoring.TierBlock.String():
				w.WriteHeader(403)
				return
			default:
				slog.Error("tier", "unknown tier", tier)
				return
			}

			newScore := scoring.RunningAverage(record.Score, decayed)
			rbd.Store(r.Context(), ip, newScore)
		})

	}
}
