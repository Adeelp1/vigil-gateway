package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration. All fields have sane defaults;
// override via environment variables for twelve-factor compatibility
type Config struct {
	ListenAddr      string        // e.g. ":8080"
	UpstreamAddr    string        // e.g. "localhost:9090"
	WorkerPoolSize  int           // goroutines ready to accept connections
	ReadTimeout     time.Duration // max time to read a full request
	WriteTimeout    time.Duration // max time to write a full response
	ShutdownTimeout time.Duration // graceful drain window
	RedisAddr       string
	RedisPassword   string
	RedisDB         int
	RedisProtocol   int
	JWTSecret       string
	RateLimitRPS    int
	RateLimitBurst  int
	StreamKey       string
	StreamGroup     string
	StreamMaxLen    int
}

func Load() Config {
	return Config{
		ListenAddr:      getEnv("LISTEN_ADDR", ":8080"),
		UpstreamAddr:    getEnv("UPSTREAM_ADDR", "localhost:9090"),
		WorkerPoolSize:  getEnvInt("WORKER_POOL_SIZE", 256),
		ReadTimeout:     getEnvDuration("READ_TIMEOUT", time.Duration(10*time.Second)),
		WriteTimeout:    getEnvDuration("WRITE_TIMEOUT", time.Duration(10*time.Second)),
		ShutdownTimeout: getEnvDuration("SHUTDOWNTIMEOUT", time.Duration(30*time.Second)),
		RedisAddr:       getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:   getEnv("REDIS_PASSWORD", ""),
		RedisDB:         getEnvInt("REDIS_DB", 0),
		RedisProtocol:   getEnvInt("REDIS_PROTOCOL", 3),
		JWTSecret:       getEnv("JWT_SECRET", ""),
		RateLimitRPS:    getEnvInt("RATE_LIMIT_RPS", 10),
		RateLimitBurst:  getEnvInt("RATE_LIMIT_BURST", 100),
		StreamKey:       getEnv("STREAM_KEY", "vigil:events"),
		StreamGroup:     getEnv("STREAM_GROUP", "ml-workers"),
		StreamMaxLen:    getEnvInt("STREAM_MAX_LEN", 10000),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}

	return fallback
}
