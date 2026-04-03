package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/pandusatrianura/heimdall-travel-service/internal/models"
	"github.com/pandusatrianura/heimdall-travel-service/internal/pkg/timeutil"
	"github.com/pandusatrianura/heimdall-travel-service/internal/pkg/utils"
)

type AirAsiaProvider struct {
	MockDataPath string
}

func NewAirAsiaProvider(mockDataPath string) *AirAsiaProvider {
	return &AirAsiaProvider{MockDataPath: mockDataPath}
}

func (p *AirAsiaProvider) Name() string {
	return "AirAsia"
}

type airAsiaResponse struct {
	Status  string          `json:"status"`
	Flights []airasiaFlight `json:"flights"`
}

type airasiaFlight struct {
	FlightCode    string  `json:"flight_code"`
	Airline       string  `json:"airline"`
	FromAirport   string  `json:"from_airport"`
	ToAirport     string  `json:"to_airport"`
	DepartTime    string  `json:"depart_time"`
	ArriveTime    string  `json:"arrive_time"`
	DurationHours float64 `json:"duration_hours"`
	DirectFlight  bool    `json:"direct_flight"`
	PriceIDR      int     `json:"price_idr"`
	Seats         int     `json:"seats"`
	CabinClass    string  `json:"cabin_class"`
	BaggageNote   string  `json:"baggage_note"`
}

func (p *AirAsiaProvider) SearchFlights(ctx context.Context, leg *models.SearchLeg) ([]models.Flight, error) {
	slog.InfoContext(ctx, "Beginning Provider search", "provider", p.Name(), "origin", leg.Origin, "destination", leg.Destination)

	failureRate := utils.ResolveFailureRate("airasia", 10)
	if rand.Intn(100) < failureRate {
		slog.WarnContext(ctx, "Simulated provider failure", "provider", p.Name())
		return nil, fmt.Errorf("airasia internal server error (simulated)")
	}

	// Simulate latency via dynamic config
	delayMs := utils.ResolveDelayMS("airasia", 100)
	delay := time.Duration(delayMs) * time.Millisecond
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	filenames := utils.ResolveMockFilenames("airasia")
	var results []models.Flight

	for _, filename := range filenames {
		path := filepath.Join(p.MockDataPath, filename)
		file, err := os.ReadFile(path)
		if err != nil {
			slog.WarnContext(ctx, "Failed to open mock JSON file, skipping", "provider", p.Name(), "filename", filename, "error", err)
			continue
		}

		var raw airAsiaResponse
		if err := json.Unmarshal(file, &raw); err != nil {
			slog.ErrorContext(ctx, "Deserialization Error", "provider", p.Name(), "filename", filename, "error", err)
			continue
		}

		slog.DebugContext(ctx, "Processing source JSON", "provider", p.Name(), "filename", filename)

		for _, f := range raw.Flights {
			if f.FromAirport != leg.Origin || f.ToAirport != leg.Destination {
				continue
			}
			// Date filter "2025-12-15"
			if len(f.DepartTime) >= 10 && f.DepartTime[:10] != leg.DepartureDate {
				continue
			}

			depTime, _ := timeutil.ParseTime(f.DepartTime, "")
			arrTime, _ := timeutil.ParseTime(f.ArriveTime, "")

			flight := models.Flight{
				ID:           fmt.Sprintf("%s_%s", f.FlightCode, p.Name()),
				Provider:     p.Name(),
				Airline:      models.Airline{Name: "AirAsia", Code: f.FlightCode[:2]},
				FlightNumber: f.FlightCode,
				Departure: models.FlightPoint{
					Airport:   f.FromAirport,
					City:      "",
					Datetime:  f.DepartTime,
					Timestamp: depTime.Unix(),
				},
				Arrival: models.FlightPoint{
					Airport:   f.ToAirport,
					City:      "",
					Datetime:  f.ArriveTime,
					Timestamp: arrTime.Unix(),
				},
				Duration: models.Duration{
					TotalMinutes: int(f.DurationHours * 60.0),
					Formatted:    timeutil.FormatDuration(int(f.DurationHours * 60.0)),
				},
				Stops: func() int {
					if !f.DirectFlight {
						return 1
					}
					return 0
				}(),
				Price: models.Price{
					Amount:   f.PriceIDR,
					Currency: "IDR",
				},
				AvailableSeats: f.Seats,
				CabinClass:     f.CabinClass,
				Aircraft:       nil,
				Amenities:      []string{},
				Baggage: models.Baggage{
					CarryOn: "Cabin baggage only, checked bags additional fee",
					Checked: "Cabin baggage only, checked bags additional fee",
				},
			}

			results = append(results, flight)
		}
	}

	slog.InfoContext(ctx, "Provider mapping success", "provider", p.Name(), "total_found", len(results))
	return results, nil
}
