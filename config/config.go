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
}

func Load() Config {
	return Config{
		ListenAddr:      getEnv("LISTEN_ADDR", ":8080"),
		UpstreamAddr:    getEnv("UPSTREAM_ADDR", "localhost:9090"),
		WorkerPoolSize:  getEnvInt("WORKER_POOL_SIZE", 256),
		ReadTimeout:     getEnvDuration("READ_TIMEOUT", time.Duration(10*time.Second)),
		WriteTimeout:    getEnvDuration("WRITE_TIMEOUT", time.Duration(10*time.Second)),
		ShutdownTimeout: getEnvDuration("SHUTDOWNTIMEOUT", time.Duration(30*time.Second)),
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
