package providers

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/pandusatrianura/heimdall-travel-service/internal/models"
)

func TestAllProviders(t *testing.T) {
	// Pointing to the mock_provider directory relative to this test file's location
	mockDataPath := filepath.Join("..", "..", "mock_provider")

	providers := []FlightProvider{
		NewGarudaProvider(mockDataPath),
		NewLionAirProvider(mockDataPath),
		NewBatikAirProvider(mockDataPath),
		NewAirAsiaProvider(mockDataPath),
	}

	for _, p := range providers {
		t.Run(p.Name(), func(t *testing.T) {

			// If it's AirAsia, we retry until it succeeds because it has a 10% failure rate
			var flights []models.Flight
			var err error
			req := &models.SearchRequest{
				Origin:        []string{"CGK"},
				Destination:   []string{"DPS"},
				DepartureDate: []string{"2025-12-15"},
			}

			ctx := context.Background()
			for i := 0; i < 10; i++ {
				flights, err = p.SearchFlights(ctx, &models.SearchLeg{Origin: req.Origin[0], Destination: req.Destination[0], DepartureDate: req.DepartureDate[0]})
				if err == nil {
					break // Succeded
				}
			}

			if err != nil {
				t.Fatalf("Provider %s failed consistently: %v", p.Name(), err)
			}

			if len(flights) == 0 {
				t.Errorf("Provider %s returned 0 flights for exact match search", p.Name())
			}

			// Spot check basic mappings
			for _, f := range flights {
				if f.Provider != p.Name() {
					t.Errorf("Expected Flight Provider %s, got %s", p.Name(), f.Provider)
				}
				if f.FlightNumber == "" {
					t.Errorf("Expected FlightNumber, got empty string")
				}
				if f.Duration.TotalMinutes <= 0 {
					t.Errorf("Expected Duration > 0, got %v", f.Duration.TotalMinutes)
				}
				if f.Departure.Airport != "CGK" || f.Arrival.Airport != "DPS" {
					t.Errorf("Expected CGK to DPS, got %s to %s", f.Departure.Airport, f.Arrival.Airport)
				}
			}
		})
	}
}

func TestProviderTimeout(t *testing.T) {
	// Test context timeout explicitly
	mockDataPath := filepath.Join("..", "..", "mock_provider")

	// Batik Air simulates 200ms delay, giving 5ms timeout should fail
	batik := NewBatikAirProvider(mockDataPath)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	req := &models.SearchRequest{
		Origin:        []string{"CGK"},
		Destination:   []string{"DPS"},
		DepartureDate: []string{"2025-12-15"},
	}

	_, err := batik.SearchFlights(ctx, &models.SearchLeg{Origin: req.Origin[0], Destination: req.Destination[0], DepartureDate: req.DepartureDate[0]})
	if err == nil {
		t.Errorf("Expected timeout error, got nil")
	}
}
