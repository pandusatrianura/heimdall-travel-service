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

type LionAirProvider struct {
	MockDataPath string
	config       ProviderRuntimeConfig
	mockFiles    []string
}

func NewLionAirProvider(mockDataPath string) *LionAirProvider {
	return NewLionAirProviderWithConfig(mockDataPath, ProviderRuntimeConfig{DelayMS: 150}, nil)
}

func NewLionAirProviderWithConfig(mockDataPath string, config ProviderRuntimeConfig, mockFiles []string) *LionAirProvider {
	return &LionAirProvider{MockDataPath: mockDataPath, config: config, mockFiles: mockFiles}
}

func (p *LionAirProvider) Name() string {
	return "Lion Air"
}

// lionAirResponse matches lion_air_search_response.json
type lionAirResponse struct {
	Success bool `json:"success"`
	Data    struct {
		AvailableFlights []lionFlight `json:"available_flights"`
	} `json:"data"`
}

type lionFlight struct {
	ID      string `json:"id"`
	Carrier struct {
		Name string `json:"name"`
		Iata string `json:"iata"`
	} `json:"carrier"`
	Route struct {
		From struct {
			Code string `json:"code"`
			Name string `json:"name"`
			City string `json:"city"`
		} `json:"from"`
		To struct {
			Code string `json:"code"`
			Name string `json:"name"`
			City string `json:"city"`
		} `json:"to"`
	} `json:"route"`
	Schedule struct {
		Departure         string `json:"departure"`
		DepartureTimezone string `json:"departure_timezone"`
		Arrival           string `json:"arrival"`
		ArrivalTimezone   string `json:"arrival_timezone"`
	} `json:"schedule"`
	FlightTime int  `json:"flight_time"`
	IsDirect   bool `json:"is_direct"`
	StopCount  int  `json:"stop_count"` // Only present typically if is_direct is false
	Pricing    struct {
		Total    int    `json:"total"`
		Currency string `json:"currency"`
		FareType string `json:"fare_type"`
	} `json:"pricing"`
	SeatsLeft int    `json:"seats_left"`
	PlaneType string `json:"plane_type"`
	Services  struct {
		WifiAvailable    bool `json:"wifi_available"`
		MealsIncluded    bool `json:"meals_included"`
		BaggageAllowance struct {
			Cabin string `json:"cabin"`
			Hold  string `json:"hold"`
		} `json:"baggage_allowance"`
	} `json:"services"`
}

func (p *LionAirProvider) SearchFlights(ctx context.Context, leg *models.SearchLeg) ([]models.Flight, error) {
	slog.InfoContext(ctx, "Beginning Provider search", "provider", p.Name(), "origin", leg.Origin, "destination", leg.Destination)

	// Simulate latency (100-300ms)
	delayMs := p.config.DelayMS
	delay := time.Duration(delayMs) * time.Millisecond
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		slog.WarnContext(ctx, "Request abandoned due to context timeout", "provider", p.Name())
		return nil, ctx.Err()
	}

	filenames := utils.ResolveMockFilenames("lion_air", p.mockFiles)
	var results []models.Flight

	for _, filename := range filenames {
		path := filepath.Join(p.MockDataPath, filename)
		file, err := os.ReadFile(path)
		if err != nil {
			slog.WarnContext(ctx, "Failed to open mock JSON file, skipping", "provider", p.Name(), "filename", filename, "error", err)
			continue
		}

		var raw lionAirResponse
		if err := json.Unmarshal(file, &raw); err != nil {
			slog.ErrorContext(ctx, "Deserialization Error", "provider", p.Name(), "filename", filename, "error", err)
			continue
		}

		slog.DebugContext(ctx, "Processing source JSON", "provider", p.Name(), "filename", filename)

		for _, f := range raw.Data.AvailableFlights {
			if f.Route.From.Code != leg.Origin || f.Route.To.Code != leg.Destination {
				continue
			}
			if len(f.Schedule.Departure) >= 10 && f.Schedule.Departure[:10] != leg.DepartureDate {
				continue
			}

			depTime, _ := timeutil.ParseTime(f.Schedule.Departure, f.Schedule.DepartureTimezone)
			arrTime, _ := timeutil.ParseTime(f.Schedule.Arrival, f.Schedule.ArrivalTimezone)

			depFormat := depTime.Format(time.RFC3339)
			arrFormat := arrTime.Format(time.RFC3339)

			stops := f.StopCount
			if stops == 0 && !f.IsDirect {
				stops = 1
			}

			aircraft := f.PlaneType
			var amenities []string
			if f.Services.WifiAvailable {
				amenities = append(amenities, "wifi")
			}
			if f.Services.MealsIncluded {
				amenities = append(amenities, "meal")
			}

			flight := models.Flight{
				ID:           fmt.Sprintf("%s_%s", f.ID, p.Name()),
				Provider:     p.Name(),
				Airline:      models.Airline{Name: f.Carrier.Name, Code: f.Carrier.Iata},
				FlightNumber: f.ID,
				Departure: models.FlightPoint{
					Airport:   f.Route.From.Code,
					City:      f.Route.From.City,
					Datetime:  depFormat,
					Timestamp: depTime.Unix(),
				},
				Arrival: models.FlightPoint{
					Airport:   f.Route.To.Code,
					City:      f.Route.To.City,
					Datetime:  arrFormat,
					Timestamp: arrTime.Unix(),
				},
				Duration: models.Duration{
					TotalMinutes: f.FlightTime,
					Formatted:    timeutil.FormatDuration(f.FlightTime),
				},
				Stops: stops,
				Price: models.Price{
					Amount:   f.Pricing.Total,
					Currency: f.Pricing.Currency,
				},
				AvailableSeats: f.SeatsLeft,
				CabinClass:     f.Pricing.FareType,
				Aircraft:       &aircraft,
				Amenities:      amenities,
				Baggage: models.Baggage{
					CarryOn: f.Services.BaggageAllowance.Cabin,
					Checked: f.Services.BaggageAllowance.Hold,
				},
			}

			results = append(results, flight)
		}
	}

	slog.InfoContext(ctx, "Provider mapping success", "provider", p.Name(), "total_found", len(results))
	return results, nil
}
