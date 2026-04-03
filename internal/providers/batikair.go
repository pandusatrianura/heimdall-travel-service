package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/pandusatrianura/heimdall-travel-service/internal/models"
	"github.com/pandusatrianura/heimdall-travel-service/internal/pkg/timeutil"
	"github.com/pandusatrianura/heimdall-travel-service/internal/pkg/utils"
)

type BatikAirProvider struct {
	MockDataPath string
	config       ProviderRuntimeConfig
	mockFiles    []string
}

func NewBatikAirProvider(mockDataPath string) *BatikAirProvider {
	return NewBatikAirProviderWithConfig(mockDataPath, ProviderRuntimeConfig{DelayMS: 200}, nil)
}

func NewBatikAirProviderWithConfig(mockDataPath string, config ProviderRuntimeConfig, mockFiles []string) *BatikAirProvider {
	return &BatikAirProvider{MockDataPath: mockDataPath, config: config, mockFiles: mockFiles}
}

func (p *BatikAirProvider) Name() string {
	return "Batik Air"
}

type batikAirResponse struct {
	Code    int           `json:"code"`
	Results []batikFlight `json:"results"`
}

type batikFlight struct {
	FlightNumber      string `json:"flightNumber"`
	AirlineName       string `json:"airlineName"`
	AirlineIATA       string `json:"airlineIATA"`
	Origin            string `json:"origin"`
	Destination       string `json:"destination"`
	DepartureDateTime string `json:"departureDateTime"` // e.g. 2025-12-15T07:15:00+0700
	ArrivalDateTime   string `json:"arrivalDateTime"`
	TravelTime        string `json:"travelTime"` // e.g. 1h 45m
	NumberOfStops     int    `json:"numberOfStops"`
	Fare              struct {
		BasePrice    int    `json:"basePrice"`
		Taxes        int    `json:"taxes"`
		TotalPrice   int    `json:"totalPrice"`
		CurrencyCode string `json:"currencyCode"`
		Class        string `json:"class"`
	} `json:"fare"`
	SeatsAvailable  int      `json:"seatsAvailable"`
	AircraftModel   string   `json:"aircraftModel"`
	BaggageInfo     string   `json:"baggageInfo"`
	OnboardServices []string `json:"onboardServices"`
}

func (p *BatikAirProvider) SearchFlights(ctx context.Context, leg *models.SearchLeg) ([]models.Flight, error) {
	slog.InfoContext(ctx, "Beginning Provider search", "provider", p.Name(), "origin", leg.Origin, "destination", leg.Destination)

	// Simulate latency (200-400ms) delay per requirements (Slower response)
	delayMs := p.config.DelayMS
	delay := time.Duration(delayMs) * time.Millisecond
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	filenames := utils.ResolveMockFilenames("batik_air", p.mockFiles)
	var results []models.Flight

	for _, filename := range filenames {
		filePath := filepath.Join(p.MockDataPath, filename)
		file, err := os.Open(filePath)
		if err != nil {
			slog.WarnContext(ctx, "Failed to open mock JSON file, skipping", "provider", p.Name(), "filename", filename, "error", err)
			continue
		}

		var raw batikAirResponse
		if err := json.NewDecoder(file).Decode(&raw); err != nil {
			slog.ErrorContext(ctx, "Deserialization Error", "provider", p.Name(), "filename", filename, "error", err)
			file.Close()
			continue
		}
		file.Close()

		slog.DebugContext(ctx, "Processing source JSON", "provider", p.Name(), "filename", filename)

		for _, f := range raw.Results {
			// Very basic Origin/Dest filtering hook
			if f.Origin != leg.Origin || f.Destination != leg.Destination {
				continue
			}
			if len(f.DepartureDateTime) >= 10 && f.DepartureDateTime[:10] != leg.DepartureDate {
				continue
			}

			depTime, _ := timeutil.ParseTime(f.DepartureDateTime, "")
			arrTime, _ := timeutil.ParseTime(f.ArrivalDateTime, "")

			// Safely construct RFC3339 formatted output
			depFormat := depTime.Format(time.RFC3339)
			arrFormat := arrTime.Format(time.RFC3339)

			durationMins := int(arrTime.Sub(depTime).Minutes())
			if durationMins < 0 {
				durationMins = 0 // basic validation to avoid negative duration corrupted data
			}

			aircraft := f.AircraftModel

			flight := models.Flight{
				ID:           fmt.Sprintf("%s_%s", f.FlightNumber, p.Name()),
				Provider:     p.Name(),
				Airline:      models.Airline{Name: f.AirlineName, Code: f.AirlineIATA},
				FlightNumber: f.FlightNumber,
				Departure: models.FlightPoint{
					Airport:   f.Origin,
					City:      "",
					Datetime:  depFormat,
					Timestamp: depTime.Unix(),
				},
				Arrival: models.FlightPoint{
					Airport:   f.Destination,
					City:      "",
					Datetime:  arrFormat,
					Timestamp: arrTime.Unix(),
				},
				Duration: models.Duration{
					TotalMinutes: durationMins,
					Formatted:    timeutil.FormatDuration(durationMins),
				},
				Stops: f.NumberOfStops,
				Price: models.Price{
					Amount:   f.Fare.TotalPrice,
					Currency: f.Fare.CurrencyCode,
				},
				AvailableSeats: f.SeatsAvailable,
				CabinClass:     f.Fare.Class,
				Aircraft:       &aircraft,
				Amenities:      f.OnboardServices,
				Baggage: models.Baggage{
					CarryOn: f.BaggageInfo,
					Checked: f.BaggageInfo,
				},
			}

			results = append(results, flight)
		}
	}

	slog.InfoContext(ctx, "Provider mapping success", "provider", p.Name(), "total_found", len(results))
	return results, nil
}
