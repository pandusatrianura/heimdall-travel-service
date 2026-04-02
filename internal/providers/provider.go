package providers

import (
	"context"

	"github.com/pandusatrianura/heimdall-travel-service/internal/models"
)

// FlightProvider represents the standard interface that all mock airline providers implement.
type FlightProvider interface {
	SearchFlights(ctx context.Context, leg *models.SearchLeg) ([]models.Flight, error)
	Name() string
}
