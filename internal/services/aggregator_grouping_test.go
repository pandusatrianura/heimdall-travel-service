package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/pandusatrianura/heimdall-travel-service/internal/models"
	"github.com/pandusatrianura/heimdall-travel-service/internal/providers"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
)

type trackingProvider struct {
	name string
	mu   sync.Mutex
	seen []models.SearchLeg
}

func (p *trackingProvider) Name() string { return p.name }

func (p *trackingProvider) SearchFlights(_ context.Context, leg *models.SearchLeg) ([]models.Flight, error) {
	p.mu.Lock()
	p.seen = append(p.seen, *leg)
	p.mu.Unlock()
	return []models.Flight{{
		ID:       p.name + "-" + leg.Direction,
		Provider: p.name,
		Airline:  models.Airline{Name: p.name},
		Price:    models.Price{Amount: 1000000, Currency: "IDR"},
		Duration: models.Duration{TotalMinutes: 90},
	}}, nil
}

func TestAggregatorService_Search_ReturnsFlatResultsForPositionalTrips(t *testing.T) {
	provider := &trackingProvider{name: "tracking"}
	svc := NewAggregatorService("", 5*time.Minute, 10*time.Minute, 1500*time.Millisecond, 0.6, 0.4)
	svc.providers = []providers.FlightProvider{provider}
	svc.breakers = map[string]*gobreaker.CircuitBreaker{
		"tracking": gobreaker.NewCircuitBreaker(gobreaker.Settings{Name: "tracking"}),
	}
	svc.limiters = map[string]*rate.Limiter{
		"tracking": rate.NewLimiter(rate.Limit(100), 100),
	}

	req := &models.SearchRequest{
		Origins:       []string{"CGK", "SUB"},
		Destinations:  []string{"DPS", "SIN"},
		DepartureDate: []string{"2025-12-15", "2025-12-20"},
		ReturnDate:    []string{"", "2025-12-26"},
	}

	resp, err := svc.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.Metadata.TotalResults != 3 {
		t.Fatalf("expected 3 flat results, got %d", resp.Metadata.TotalResults)
	}

	if len(resp.Flights) != 3 {
		t.Fatalf("expected 3 flights in flat response, got %d", len(resp.Flights))
	}

	if len(provider.seen) != 3 {
		t.Fatalf("expected provider to receive 3 positional legs, got %d", len(provider.seen))
	}

	seen := make(map[models.SearchLeg]int, len(provider.seen))
	for _, leg := range provider.seen {
		seen[leg]++
	}

	expectedLegs := []models.SearchLeg{
		{TripIndex: 0, Direction: models.DirectionOutbound, Origin: "CGK", Destination: "DPS", DepartureDate: "2025-12-15"},
		{TripIndex: 1, Direction: models.DirectionOutbound, Origin: "SUB", Destination: "SIN", DepartureDate: "2025-12-20"},
		{TripIndex: 1, Direction: models.DirectionInbound, Origin: "SIN", Destination: "SUB", DepartureDate: "2025-12-26"},
	}

	for _, expected := range expectedLegs {
		if seen[expected] != 1 {
			t.Fatalf("expected dispatched leg %#v exactly once, got count %d", expected, seen[expected])
		}
	}

	if len(resp.SearchCriteria.ReturnDate) != 2 {
		t.Fatalf("expected search criteria to preserve positional return dates, got %#v", resp.SearchCriteria.ReturnDate)
	}

	if resp.SearchCriteria.ReturnDate[0] != "" || resp.SearchCriteria.ReturnDate[1] != "2025-12-26" {
		t.Fatalf("unexpected positional return dates: %#v", resp.SearchCriteria.ReturnDate)
	}
}
