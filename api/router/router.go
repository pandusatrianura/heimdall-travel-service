package router

import (
	"net/http"

	"github.com/pandusatrianura/heimdall-travel-service/config"
	"github.com/pandusatrianura/heimdall-travel-service/internal/handlers"
	"github.com/pandusatrianura/heimdall-travel-service/internal/services"
)

func Build(cfg config.Config) *http.ServeMux {
	aggregatorSvc := services.NewAggregatorServiceWithProviderConfig(
		cfg.MockDataPath,
		cfg.CacheTTL,
		cfg.CacheCleanup,
		cfg.ProviderTimeout,
		cfg.BestValuePriceWeight,
		cfg.BestValueDurationWeight,
		cfg.ProviderRuntime,
	)

	searchHandler := handlers.NewSearchHandler(aggregatorSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/search", searchHandler.HandleSearch)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	return mux
}
