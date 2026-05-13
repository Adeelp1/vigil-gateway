package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Adeelp1/vigil-gateway/config"
	"github.com/Adeelp1/vigil-gateway/internal/proxy"
	"github.com/Adeelp1/vigil-gateway/internal/store"
)

func main() {
	// Structured JSON logging in production; pretty text in development.
	// Switch the handler based on an env var.
	if os.Getenv("LOG_FORMAT") == "json" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	}

	cfg := config.Load()

	rdb := store.NewRedisClient(cfg)

	srv, err := proxy.New(cfg, rdb)
	if err != nil {
		slog.Error("failed to start server", "err", err)
		os.Exit(1)
	}

	// Root context cancelled on SIGINT/SIGTERM - drives graceful shutdown.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Serve blocka until ctx is cancelled.
	if err := srv.Serve(ctx); err != nil {
		slog.Error("server error", "err", err)
	}

	// Drain with aseparated timout context - we don't want to inherit a
	// cancelled context for the shutdown itself.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "err", err)
		os.Exit(1)
	}
}
