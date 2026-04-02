package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/pandusatrianura/heimdall-travel-service/internal/handlers"
	"github.com/pandusatrianura/heimdall-travel-service/internal/pkg/logger"
	"github.com/pandusatrianura/heimdall-travel-service/internal/services"
)

func main() {
	// Initialize Structured Logging
	logger.InitLogger()

	// Dynamically load generic .env payload if it natively resolves in CWD
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found, using default values")
	}

	cwd, err := os.Getwd()
	if err != nil {
		slog.Error("Failed to get cwd", "error", err)
		os.Exit(1)
	}

	mockDataPath := os.Getenv("MOCK_DATA_PATH")
	if mockDataPath == "" {
		mockDataPath = "mock_provider" // sane default fallback
	}

	mockDataAbs := filepath.Join(cwd, mockDataPath)

	// Configuration parsing
	ttlMin, _ := strconv.Atoi(os.Getenv("CACHE_TTL_MINUTES"))
	if ttlMin == 0 {
		ttlMin = 5
	} // default 5m

	cleanupMin, _ := strconv.Atoi(os.Getenv("CACHE_CLEANUP_MINUTES"))
	if cleanupMin == 0 {
		cleanupMin = 10
	} // default 10m

	timeoutMs, _ := strconv.Atoi(os.Getenv("PROVIDER_TIMEOUT_MS"))
	if timeoutMs == 0 {
		timeoutMs = 1500
	} // default 1.5s

	pWeightStr := os.Getenv("BEST_VALUE_PRICE_WEIGHT")
	pWeight, err := strconv.ParseFloat(pWeightStr, 64)
	if err != nil || pWeight == 0 {
		pWeight = 0.6
	}

	dWeightStr := os.Getenv("BEST_VALUE_DURATION_WEIGHT")
	dWeight, err := strconv.ParseFloat(dWeightStr, 64)
	if err != nil || dWeight == 0 {
		dWeight = 0.4
	}

	// Initialize Services
	aggregatorSvc := services.NewAggregatorService(
		mockDataAbs,
		time.Duration(ttlMin)*time.Minute,
		time.Duration(cleanupMin)*time.Minute,
		time.Duration(timeoutMs)*time.Millisecond,
		pWeight,
		dWeight,
	)

	// Initialize Handlers
	searchHandler := handlers.NewSearchHandler(aggregatorSvc)

	// Setup Routes using Go 1.22+ ServeMux routing
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/search", searchHandler.HandleSearch)

	// Add simple healthcheck
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Server configuration
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Channel to listen for signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Run server in a separate goroutine
	go func() {
		slog.Info("Starting Heimdall Travel Service", "port", port)
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
