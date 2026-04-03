package providers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pandusatrianura/heimdall-travel-service/internal/models"
)

func testLeg() *models.SearchLeg {
	return &models.SearchLeg{Origin: "CGK", Destination: "DPS", DepartureDate: "2025-12-15"}
}

func providerFactory(name string) func(string) FlightProvider {
	switch name {
	case "Garuda Indonesia":
		return func(mockDataPath string) FlightProvider { return NewGarudaProvider(mockDataPath) }
	case "Lion Air":
		return func(mockDataPath string) FlightProvider { return NewLionAirProvider(mockDataPath) }
	case "Batik Air":
		return func(mockDataPath string) FlightProvider { return NewBatikAirProvider(mockDataPath) }
	case "AirAsia":
		return func(mockDataPath string) FlightProvider { return NewAirAsiaProvider(mockDataPath) }
	default:
		return nil
	}
}

func envPrefixForProvider(name string) string {
	switch name {
	case "Garuda Indonesia":
		return "GARUDA_INDONESIA"
	case "Lion Air":
		return "LION_AIR"
	case "Batik Air":
		return "BATIK_AIR"
	case "AirAsia":
		return "AIRASIA"
	default:
		return ""
	}
}

func defaultFilenameForProvider(name string) string {
	switch name {
	case "Garuda Indonesia":
		return "garuda_indonesia_search_response.json"
	case "Lion Air":
		return "lion_air_search_response.json"
	case "Batik Air":
		return "batik_air_search_response.json"
	case "AirAsia":
		return "airasia_search_response.json"
	default:
		return "unknown.json"
	}
}

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
			if p.Name() == "AirAsia" {
				t.Setenv("AIRASIA_FAILURE_RATE", "0")
			}

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

func TestProviders_HandleMissingFilesGracefully(t *testing.T) {
	providerNames := []string{"Garuda Indonesia", "Lion Air", "Batik Air", "AirAsia"}

	for _, name := range providerNames {
		t.Run(name, func(t *testing.T) {
			factory := providerFactory(name)
			provider := factory(t.TempDir())

			prefix := envPrefixForProvider(name)
			t.Setenv("MOCK_DATA_PROVIDER", `["missing.json"]`)
			t.Setenv(prefix+"_DELAY_MS", "0")
			if name == "AirAsia" {
				t.Setenv(prefix+"_FAILURE_RATE", "0")
			}

			flights, err := provider.SearchFlights(context.Background(), testLeg())
			if err != nil {
				t.Fatalf("expected no error for missing file skip, got %v", err)
			}
			if len(flights) != 0 {
				t.Fatalf("expected 0 flights when all files are missing, got %d", len(flights))
			}
		})
	}
}

func TestProviders_HandleMalformedJSONGracefully(t *testing.T) {
	providerNames := []string{"Garuda Indonesia", "Lion Air", "Batik Air", "AirAsia"}

	for _, name := range providerNames {
		t.Run(name, func(t *testing.T) {
			tempDir := t.TempDir()
			filename := defaultFilenameForProvider(name)
			if err := os.WriteFile(filepath.Join(tempDir, filename), []byte(`{invalid json`), 0o600); err != nil {
				t.Fatalf("os.WriteFile() error = %v", err)
			}

			factory := providerFactory(name)
			provider := factory(tempDir)

			prefix := envPrefixForProvider(name)
			t.Setenv("MOCK_DATA_PROVIDER", `["`+filename+`"]`)
			t.Setenv(prefix+"_DELAY_MS", "0")
			if name == "AirAsia" {
				t.Setenv(prefix+"_FAILURE_RATE", "0")
			}

			flights, err := provider.SearchFlights(context.Background(), testLeg())
			if err != nil {
				t.Fatalf("expected no error for malformed file skip, got %v", err)
			}
			if len(flights) != 0 {
				t.Fatalf("expected 0 flights when all files are malformed, got %d", len(flights))
			}
		})
	}
}

func TestProviders_RespectPreCancelledContext(t *testing.T) {
	providerNames := []string{"Garuda Indonesia", "Lion Air", "Batik Air", "AirAsia"}

	for _, name := range providerNames {
		t.Run(name, func(t *testing.T) {
			factory := providerFactory(name)
			provider := factory(filepath.Join("..", "..", "mock_provider"))

			prefix := envPrefixForProvider(name)
			t.Setenv(prefix+"_DELAY_MS", "50")
			if name == "AirAsia" {
				t.Setenv(prefix+"_FAILURE_RATE", "0")
			}

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			_, err := provider.SearchFlights(ctx, testLeg())
			if err == nil {
				t.Fatal("expected context cancellation error, got nil")
			}
			if !strings.Contains(err.Error(), "context canceled") {
				t.Fatalf("expected context canceled error, got %v", err)
			}
		})
	}
}

func TestAirAsiaProvider_SimulatedFailureReturnsError(t *testing.T) {
	provider := NewAirAsiaProvider(filepath.Join("..", "..", "mock_provider"))
	t.Setenv("AIRASIA_FAILURE_RATE", "100")
	t.Setenv("AIRASIA_DELAY_MS", "0")

	_, err := provider.SearchFlights(context.Background(), testLeg())
	if err == nil {
		t.Fatal("expected simulated failure error, got nil")
	}
	if !strings.Contains(err.Error(), "airasia internal server error (simulated)") {
		t.Fatalf("unexpected AirAsia error: %v", err)
	}
}
