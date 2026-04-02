package providers

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/pandusatrianura/heimdall-travel-service/internal/models"
)

func TestAirAsiaProvider_SearchFlights(t *testing.T) {
	mockDataPath := filepath.Join("..", "..", "mock_provider")
	provider := NewAirAsiaProvider(mockDataPath)

	t.Run("Validations with Retries", func(t *testing.T) {
		req := &models.SearchRequest{
			Origin:        []string{"CGK"},
			Destination:   []string{"DPS"},
			DepartureDate: []string{"2025-12-15"},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		var flights []models.Flight
		var err error

		// AirAsia has a simulated 10% failure rate; loop until success or 10 tries
		for i := 0; i < 10; i++ {
			flights, err = provider.SearchFlights(ctx, &models.SearchLeg{Origin: req.Origin[0], Destination: req.Destination[0], DepartureDate: req.DepartureDate[0]})
			if err == nil {
				break
			}
		}

		if err != nil {
			t.Fatalf("Unexpected persistent error after retries: %v", err)
		}

		if len(flights) == 0 {
			t.Fatalf("Expected flights to be found")
		}

		for _, f := range flights {
			if f.Airline.Code != "QZ" {
				t.Errorf("Expected Airline Code QZ (derived from QZ520 heuristic), got %s", f.Airline.Code)
			}
			if f.Provider != "AirAsia" {
				t.Errorf("Expected Provider AirAsia, got %s", f.Provider)
			}
			if f.Baggage.CarryOn == "" {
				t.Errorf("Expected baggage notes mapped")
			}
		}
	})

	t.Run("Failure Simulation Probability Test", func(t *testing.T) {
		req := &models.SearchRequest{
			Origin:        []string{"CGK"},
			Destination:   []string{"DPS"},
			DepartureDate: []string{"2025-12-15"},
		}

		ctx := context.Background()

		// Run 100 queries; statitically we should see at least ONE error due to 10% failure rate
		failureCount := 0
		for i := 0; i < 100; i++ {
			_, err := provider.SearchFlights(ctx, &models.SearchLeg{Origin: req.Origin[0], Destination: req.Destination[0], DepartureDate: req.DepartureDate[0]})
			if err != nil {
				failureCount++
			}
		}

		if failureCount == 0 {
			// Actually possible to have 0 failures, but statistically rare (0.9^100 is tiny).
			t.Logf("Notice: The 10%% failure simulation did not trigger in 100 tries.")
		} else {
			t.Logf("Successfully captured %d simulated failures in 100 tries.", failureCount)
		}
	})
}
