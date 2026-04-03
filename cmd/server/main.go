package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pandusatrianura/heimdall-travel-service/api/router"
	"github.com/pandusatrianura/heimdall-travel-service/config"
	"github.com/pandusatrianura/heimdall-travel-service/internal/pkg/logger"
)

func main() {
	// Initialize Structured Logging
	logger.InitLogger()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	mux := router.Build(cfg)

	// Server configuration
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	// Channel to listen for signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Run server in a separate goroutine
	go func() {
		slog.Info("Starting Heimdall Travel Service", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for terminate signal
	<-stop
	slog.Info("Shutting down server gracefully...")

	// Create a context with timeout for shutdown period
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server exited properly")
}
