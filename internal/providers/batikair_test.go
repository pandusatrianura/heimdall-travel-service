package providers

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/pandusatrianura/heimdall-travel-service/internal/models"
)

func TestBatikAirProvider_SearchFlights(t *testing.T) {
	mockDataPath := filepath.Join("..", "..", "mock_provider")
	provider := NewBatikAirProvider(mockDataPath)

	t.Run("Validations & Default Values", func(t *testing.T) {
		req := &models.SearchRequest{
			Origin:        []string{"CGK"},
			Destination:   []string{"DPS"},
			DepartureDate: []string{"2025-12-15"},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		flights, err := provider.SearchFlights(ctx, &models.SearchLeg{Origin: req.Origin[0], Destination: req.Destination[0], DepartureDate: req.DepartureDate[0]})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(flights) == 0 {
			t.Fatalf("Expected flights to be found")
		}

		// Ensure Batik parsing mapping succeeded
		for _, f := range flights {
			if f.Airline.Name != "Batik Air" {
				t.Errorf("Expected Airline Batik Air, got %s", f.Airline.Name)
			}
			if f.Duration.TotalMinutes <= 0 {
				t.Errorf("Calculated duration failed, got %d", f.Duration.TotalMinutes)
			}
			if f.Price.Amount <= 0 {
				t.Errorf("Price parsing failed, got %d", f.Price.Amount)
			}
		}
	})

	t.Run("Simulate Context Timeout", func(t *testing.T) {
		req := &models.SearchRequest{
			Origin:        []string{"CGK"},
			Destination:   []string{"DPS"},
			DepartureDate: []string{"2025-12-15"},
		}

		// BatikAir takes 200-400ms simulate delay. A 5ms Context Timeout should trigger circuit breaker.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		defer cancel()

		_, err := provider.SearchFlights(ctx, &models.SearchLeg{Origin: req.Origin[0], Destination: req.Destination[0], DepartureDate: req.DepartureDate[0]})
		if err == nil {
			t.Errorf("Expected context timeout error, got nil")
		}
	})
}
