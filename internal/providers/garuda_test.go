package providers

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/pandusatrianura/heimdall-travel-service/internal/models"
)

func TestGarudaProvider_SearchFlights(t *testing.T) {
	mockDataPath := filepath.Join("..", "..", "mock_provider")
	provider := NewGarudaProvider(mockDataPath)

	t.Run("Valid Search Matching Criteria", func(t *testing.T) {
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

		f := flights[0]
		if f.Provider != "Garuda Indonesia" {
			t.Errorf("Expected Provider Name 'Garuda Indonesia', got %s", f.Provider)
		}
		if f.Departure.Airport != "CGK" {
			t.Errorf("Expected Origin CGK, got %s", f.Departure.Airport)
		}
		if f.Arrival.Airport != "DPS" {
			t.Errorf("Expected Destination DPS, got %s", f.Arrival.Airport)
		}
		// Based on garuda mock: GA400
		if f.FlightNumber != "GA400" {
			t.Errorf("Expected GA400, got %s", f.FlightNumber)
		}
	})

	t.Run("Non-Matching Search Criteria", func(t *testing.T) {
		req := &models.SearchRequest{
			Origin:        []string{"LAX"},
			Destination:   []string{"JFK"},
			DepartureDate: []string{"2025-12-15"},
		}

		flights, err := provider.SearchFlights(context.Background(), &models.SearchLeg{Origin: req.Origin[0], Destination: req.Destination[0], DepartureDate: req.DepartureDate[0]})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(flights) != 0 {
			t.Errorf("Expected 0 flights for mismatched route, got %d", len(flights))
		}
	})
}
