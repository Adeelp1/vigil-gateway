package store

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/Adeelp1/vigil-gateway/config"
	"github.com/redis/go-redis/v9"
)

type Redis struct {
	client *redis.Client
}

type ScoreRecord struct {
	Score     float64
	WrittenAt int64
}

var scoreKey string = "vigil:score:"

func NewRedisClient(cfg config.Config) Redis {
	return Redis{
		client: redis.NewClient(
			&redis.Options{
				Addr:     cfg.RedisAddr,
				Password: cfg.RedisPassword,
				DB:       cfg.RedisDB,
				Protocol: cfg.RedisProtocol,
			},
		),
	}
}

// Store saves a threat score for an IP with a timestamp.
// Value formate: "score|unix_timestamp"
func (rdb *Redis) Store(ctx context.Context, ip string, score float64) error {
	value := fmt.Sprintf("%f|%d", score, time.Now().Unix())

	err := rdb.client.Set(ctx, scoreKey+ip, value, 30*time.Minute).Err()
	if err != nil {
		slog.Error("redis store failed", "ip", ip, "err", err)
		return err
	}
	return nil
}

// Get retrieves and parses the score record for an IP.
// Returns redis.Nil error if the key doesn't exist (client is new/clean).
func (rdb *Redis) Get(ctx context.Context, ip string) (ScoreRecord, error) {
	val, err := rdb.client.Get(ctx, scoreKey+ip).Result()
	if err != nil {
		return ScoreRecord{}, err
	}

	parts := strings.SplitN(val, "|", 2)
	if len(parts) != 2 {
		return ScoreRecord{}, fmt.Errorf("malformed score record: %q", val)
	}

	score, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return ScoreRecord{}, fmt.Errorf("invalid score value: %q", parts[0])
	}

	writtenAt, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return ScoreRecord{}, fmt.Errorf("invalid timestamp: %q", parts[1])
	}

	return ScoreRecord{
		Score:     score,
		WrittenAt: writtenAt,
	}, nil
}
