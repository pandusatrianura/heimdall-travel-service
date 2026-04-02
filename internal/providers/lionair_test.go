package providers

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/pandusatrianura/heimdall-travel-service/internal/models"
)

func TestLionAirProvider_SearchFlights(t *testing.T) {
	mockDataPath := filepath.Join("..", "..", "mock_provider")
	provider := NewLionAirProvider(mockDataPath)

	t.Run("Timezone Parsing & Result Validations", func(t *testing.T) {
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
			t.Fatalf("Expected mock flights to be returned")
		}

		// Spot check specifically LionAir's complex tz format handling
		for _, f := range flights {
			if f.Provider != "Lion Air" {
				t.Errorf("Expected Lion Air, got %s", f.Provider)
			}
			if f.Departure.Timestamp <= 0 {
				t.Errorf("Unix timestamp not translated correctly from timezone strings")
			}
			if len(f.Departure.Datetime) == 0 {
				t.Errorf("Datetime string missing")
			}
			if f.Price.Currency != "IDR" {
				t.Errorf("Expected currency IDR mapped, got %s", f.Price.Currency)
			}
		}
	})
}
