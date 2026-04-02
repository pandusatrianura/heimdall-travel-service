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
)

type GarudaProvider struct {
	MockDataPath string
}

func NewGarudaProvider(mockDataPath string) *GarudaProvider {
	return &GarudaProvider{MockDataPath: mockDataPath}
}

func (p *GarudaProvider) Name() string {
	return "Garuda Indonesia"
}

// garudaResponse matches the JSON structure of garuda mock provider
type garudaResponse struct {
	Status  string         `json:"status"`
	Flights []garudaFlight `json:"flights"`
}

type garudaFlight struct {
	FlightID    string `json:"flight_id"`
	Airline     string `json:"airline"`
	AirlineCode string `json:"airline_code"`
	Departure   struct {
		Airport  string `json:"airport"`
		City     string `json:"city"`
		Time     string `json:"time"`
		Terminal string `json:"terminal"`
	} `json:"departure"`
	Arrival struct {
		Airport  string `json:"airport"`
		City     string `json:"city"`
		Time     string `json:"time"`
		Terminal string `json:"terminal"`
	} `json:"arrival"`
	DurationMinutes int    `json:"duration_minutes"`
	Stops           int    `json:"stops"`
	Aircraft        string `json:"aircraft"`
	Price           struct {
		Amount   int    `json:"amount"`
		Currency string `json:"currency"`
	} `json:"price"`
	AvailableSeats int    `json:"available_seats"`
	FareClass      string `json:"fare_class"`
	Baggage        struct {
		CarryOn int `json:"carry_on"`
		Checked int `json:"checked"`
	} `json:"baggage"`
	Amenities []string `json:"amenities"`
}

func (p *GarudaProvider) SearchFlights(ctx context.Context, leg *models.SearchLeg) ([]models.Flight, error) {
	slog.InfoContext(ctx, "Beginning Provider search", "provider", p.Name(), "origin", leg.Origin, "destination", leg.Destination)

	// Simulate 50-100ms delay per requirements
	delayMs := ResolveDelayMS("garuda_indonesia", 50)
	delay := time.Duration(delayMs) * time.Millisecond

	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	filenames := ResolveMockFilenames("garuda_indonesia")
	var results []models.Flight

	for _, filename := range filenames {
		path := filepath.Join(p.MockDataPath, filename)
		file, err := os.ReadFile(path)
		if err != nil {
			slog.WarnContext(ctx, "Failed to open mock JSON file, skipping", "provider", p.Name(), "filename", filename, "error", err)
			continue
		}

		var raw garudaResponse
		if err := json.Unmarshal(file, &raw); err != nil {
			slog.ErrorContext(ctx, "Deserialization Error", "provider", p.Name(), "filename", filename, "error", err)
			continue
		}

		slog.DebugContext(ctx, "Processing source JSON", "provider", p.Name(), "filename", filename)

		for _, f := range raw.Flights {
			// Basic origin/dest filter here
			if f.Departure.Airport != leg.Origin || f.Arrival.Airport != leg.Destination {
				continue
			}
			// Ensure date match (basic timestamp sub-string match)
			if len(f.Departure.Time) >= 10 && f.Departure.Time[:10] != leg.DepartureDate {
				continue
			}

			depTime, _ := timeutil.ParseTime(f.Departure.Time, "")
			arrTime, _ := timeutil.ParseTime(f.Arrival.Time, "")

			// Duration calculation
			durationMins := f.DurationMinutes
			if durationMins == 0 {
				durationMins = int(arrTime.Sub(depTime).Minutes())
			}

			carryOnStr := "Included"
			if f.Baggage.CarryOn == 0 {
				carryOnStr = "Not Included"
			}
			checkedStr := "Included"
			if f.Baggage.Checked == 0 {
				checkedStr = "Not Included"
			}

			aircraft := f.Aircraft // copy to pointer

			flight := models.Flight{
				ID:           fmt.Sprintf("%s_%s", f.FlightID, p.Name()),
				Provider:     p.Name(),
				Airline:      models.Airline{Name: f.Airline, Code: f.AirlineCode},
				FlightNumber: f.FlightID,
				Departure: models.FlightPoint{
					Airport:   f.Departure.Airport,
					City:      f.Departure.City,
					Datetime:  f.Departure.Time,
					Timestamp: depTime.Unix(),
				},
				Arrival: models.FlightPoint{
					Airport:   f.Arrival.Airport,
					City:      f.Arrival.City,
					Datetime:  f.Arrival.Time,
					Timestamp: arrTime.Unix(),
				},
				Duration: models.Duration{
					TotalMinutes: durationMins,
					Formatted:    timeutil.FormatDuration(durationMins),
				},
				Stops: f.Stops,
				Price: models.Price{
					Amount:   f.Price.Amount,
					Currency: f.Price.Currency,
				},
				AvailableSeats: f.AvailableSeats,
				CabinClass:     f.FareClass,
				Aircraft:       &aircraft,
				Amenities:      f.Amenities,
				Baggage: models.Baggage{
					CarryOn: carryOnStr,
					Checked: checkedStr,
				},
			}

			results = append(results, flight)
		}
	}

	slog.InfoContext(ctx, "Provider mapping success", "provider", p.Name(), "total_found", len(results))
	return results, nil
}
