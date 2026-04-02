package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/pandusatrianura/heimdall-travel-service/internal/models"
	"github.com/pandusatrianura/heimdall-travel-service/internal/pkg/ctxutil"
	"github.com/pandusatrianura/heimdall-travel-service/internal/services"
)

type SearchHandler struct {
	aggregator *services.AggregatorService
}

func NewSearchHandler(aggregator *services.AggregatorService) *SearchHandler {
	return &SearchHandler{aggregator: aggregator}
}

func (h *SearchHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	reqID := uuid.New().String()
	ctx := ctxutil.ContextWithRequestID(r.Context(), reqID)

	// Parse the JSON body
	var req models.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.ErrorContext(ctx, "Failed to decode request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	slog.InfoContext(ctx, "Received Search Request",
		"origin", req.Origin,
		"destination", req.Destination,
		"departure_date", req.DepartureDate)

	// Basic Validation (Supporting aliases)
	origins := req.Origin
	if len(origins) == 0 {
		origins = req.Origins
	}
	destinations := req.Destination
	if len(destinations) == 0 {
		destinations = req.Destinations
	}

	if len(origins) == 0 || len(destinations) == 0 || len(req.DepartureDate) == 0 {
		slog.Warn("Validation failed: missing core fields")
		http.Error(w, `{"error": "origin, destination, and departureDate are required"}`, http.StatusBadRequest)
		return
	}

	// Parse query parameters for filtering if provided
	query := r.URL.Query()
	req.SortBy = query.Get("sort_by")
	if minPrice, err := strconv.Atoi(query.Get("min_price")); err == nil {
		req.MinPrice = minPrice
	}
	if maxPrice, err := strconv.Atoi(query.Get("max_price")); err == nil {
		req.MaxPrice = maxPrice
	}
	if maxStopsStr := query.Get("max_stops"); maxStopsStr != "" {
		if maxStops, err := strconv.Atoi(maxStopsStr); err == nil {
			req.MaxStops = &maxStops
		}
	}
	if maxDur, err := strconv.Atoi(query.Get("max_duration")); err == nil {
		req.MaxDuration = maxDur
	}
	if airlines := query.Get("airlines"); airlines != "" {
		req.Airlines = strings.Split(airlines, ",")
	}

	slog.InfoContext(ctx, "Dispatching to Aggregator Service")
	// Fetch from service
	resp, err := h.aggregator.Search(ctx, &req)
	if err != nil {
		slog.ErrorContext(ctx, "Internal Aggregator Error", "error", err)
		http.Error(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.InfoContext(ctx, "Successfully processed Search", "total_results", resp.Metadata.TotalResults)
	// Return JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Unable to encode response", http.StatusInternalServerError)
	}
}
