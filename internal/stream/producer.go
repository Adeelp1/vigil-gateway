package stream

import (
	"context"
	"log/slog"

	"github.com/Adeelp1/vigil-gateway/config"
	"github.com/Adeelp1/vigil-gateway/internal/store"
	"github.com/redis/go-redis/v9"
)

type Event struct {
	IP         string
	Sub        string
	Method     string
	Path       string
	Status     int
	DurationMS int64
	UserAgent  string
	Timestamp  int64
}

func Publish(ctx context.Context, rdb store.Redis, cfg config.Config, e Event) error {
	slog.Info("publishing event", "stream", cfg.StreamKey, "ip", e.IP)
	redis_arg := redis.XAddArgs{
		Stream: cfg.StreamKey,
		MaxLen: int64(cfg.StreamMaxLen),
		Approx: true,
		Values: map[string]any{
			"ip":          e.IP,
			"sub":         e.Sub,
			"method":      e.Method,
			"path":        e.Path,
			"status":      e.Status,
			"duration_ms": e.DurationMS,
			"user_agent":  e.UserAgent,
			"timestamp":   e.Timestamp,
		},
	}
	result := rdb.Client.XAdd(ctx, &redis_arg)
	slog.Info("xadd result", "id", result.Val(), "err", result.Err())
	err := result.Err()
	if err != nil {
		slog.Error("stream publish failed", "err", err)
	} else {
		slog.Info("event published successfully", "stream", cfg.StreamKey, "ip", e.IP)
	}
	return err
}
