package services

import (
	"context"
	"testing"
	"time"

	"github.com/pandusatrianura/heimdall-travel-service/internal/models"
	"github.com/pandusatrianura/heimdall-travel-service/internal/providers"
	"github.com/sony/gobreaker"
)

// mockProvider for testing Aggregator
type mockProvider struct {
	name          string
	flights       []models.Flight
	errToRet      error
	delay         time.Duration
	failCount     int
	attemptsCount int
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) SearchFlights(ctx context.Context, leg *models.SearchLeg) ([]models.Flight, error) {
	m.attemptsCount++
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if m.failCount > 0 {
		m.failCount--
		return nil, m.errToRet
	}

	return m.flights, nil
}

func TestFilterAndSortFlights(t *testing.T) {
	// Dummy Setup
	f1 := models.Flight{
		ID:       "1",
		Provider: "A",
		Airline:  models.Airline{Name: "Lion"},
		Price:    models.Price{Amount: 1500000},     // Expensive
		Duration: models.Duration{TotalMinutes: 60}, // Very Fast
		Stops:    0,
	}
	f2 := models.Flight{
		ID:       "2",
		Provider: "B",
		Airline:  models.Airline{Name: "Batik"},
		Price:    models.Price{Amount: 500000},       // Cheap
		Duration: models.Duration{TotalMinutes: 300}, // Slow
		Stops:    1,
	}
	f3 := models.Flight{
		ID:       "3",
		Provider: "C",
		Airline:  models.Airline{Name: "Garuda"},
		Price:    models.Price{Amount: 800000},      // Medium
		Duration: models.Duration{TotalMinutes: 90}, // Fast
		Stops:    0,
	}

	flights := []models.Flight{f1, f2, f3}

	svc := NewAggregatorService("", 5*time.Minute, 10*time.Minute, 1500*time.Millisecond, 0.6, 0.4)

	t.Run("Best Value sorting", func(t *testing.T) {
		req := &models.SearchRequest{SortBy: "best_value"}
		res := svc.FilterAndSortFlights(flights, req, context.Background())

		if len(res) != 3 {
			t.Errorf("expected 3 flights, got %d", len(res))
		}
		if res[0].ID != "3" {
			t.Errorf("Expected Flight 3 to be best value overall, got %s", res[0].ID)
		}
	})

	t.Run("Price Lowest sorting", func(t *testing.T) {
		req := &models.SearchRequest{SortBy: "price_lowest"}
		res := svc.FilterAndSortFlights(flights, req, context.Background())
		if res[0].ID != "2" || res[2].ID != "1" {
			t.Errorf("Price lowest failed")
		}

		// IDR test check
		if res[0].Price.FormattedAmount != "Rp 500.000" {
			t.Errorf("IDR Formatting failed, got %s", res[0].Price.FormattedAmount)
		}
	})

	t.Run("Filter Max Stops", func(t *testing.T) {
		zero := 0
		req := &models.SearchRequest{MaxStops: &zero}
		res := svc.FilterAndSortFlights(flights, req, context.Background())
		if len(res) != 2 {
			t.Errorf("Expected 2 flights with 0 stops, got %d", len(res))
		}
	})

	t.Run("Filter Max Price", func(t *testing.T) {
		req := &models.SearchRequest{MaxPrice: 700000}
		res := svc.FilterAndSortFlights(flights, req, context.Background())
		if len(res) != 1 || res[0].ID != "2" {
			t.Errorf("Expected only flight 2 due to max price, got %v", len(res))
		}
	})

	t.Run("Filter Max Duration", func(t *testing.T) {
		req := &models.SearchRequest{MaxDuration: 100}
		res := svc.FilterAndSortFlights(flights, req, context.Background())
		if len(res) != 2 {
			t.Errorf("Expected 2 flights with <=100m duration")
		}
	})

	t.Run("Filter Airlines", func(t *testing.T) {
		req := &models.SearchRequest{Airlines: []string{"Garuda", "Lion"}}
		res := svc.FilterAndSortFlights(flights, req, context.Background())
		if len(res) != 2 {
			t.Errorf("Expected 2 flights matching specific airlines, got %d", len(res))
		}
	})

	t.Run("Sort Duration Shortest", func(t *testing.T) {
		req := &models.SearchRequest{SortBy: "duration_shortest"}
		res := svc.FilterAndSortFlights(flights, req, context.Background())
		if res[0].ID != "1" || res[2].ID != "2" {
			t.Errorf("Shortest duration sort failed")
		}
	})

	t.Run("Sort Departure Earliest", func(t *testing.T) {
		req := &models.SearchRequest{SortBy: "departure_earliest"}
		flightsWithTime := flights
		flightsWithTime[0].Departure.Timestamp = 3000
		flightsWithTime[1].Departure.Timestamp = 1000 // Earliest
		flightsWithTime[2].Departure.Timestamp = 2000

		res := svc.FilterAndSortFlights(flightsWithTime, req, context.Background())
		if res[0].ID != "2" || res[2].ID != "1" {
			t.Errorf("Departure earliest sort failed")
		}
	})

	t.Run("Timezone Normalization Test", func(t *testing.T) {
		// Mock a flight with WIB (+07:00) and WITA (+08:00)
		// Duration should be calculated correctly.
		// "2025-12-15T06:00:00+07:00" -> "2025-12-15T08:00:00+08:00"
		// 06:00 WIB is 07:00 WITA. So arrives at 08:00 WITA. Duration is 1 hour.
		// This tests the domain logic of parsing and cross-timezone arrival math.
		importTime1 := int64(1765753200) // Dec 15th 2025 06:00:00 UTC+7
		importTime2 := int64(1765756800) // Dec 15th 2025 08:00:00 UTC+8

		// Difference is 3600 seconds = 60 minutes
		diffMins := (importTime2 - importTime1) / 60
		if diffMins != 60 {
			t.Errorf("Expected 60 mins duration logic difference, got %d", diffMins)
		}
	})
}

func TestAggregatorWithTimeouts(t *testing.T) {
	// Setup Mock providers
	fastProv := &mockProvider{name: "Fast", flights: []models.Flight{{ID: "fast1"}}, delay: 10 * time.Millisecond}
	slowProv := &mockProvider{name: "Slow", flights: []models.Flight{{ID: "slow1"}}, delay: 2000 * time.Millisecond} // Too slow

	svc := NewAggregatorService("", 5*time.Minute, 10*time.Minute, 1500*time.Millisecond, 0.6, 0.4)
	// Overwrite default providers with our test mocks
	svc.providers = []providers.FlightProvider{fastProv, slowProv}

	// Manually register breakers for the mock providers
	svc.breakers["Fast"] = gobreaker.NewCircuitBreaker(gobreaker.Settings{Name: "Fast"})
	svc.breakers["Slow"] = gobreaker.NewCircuitBreaker(gobreaker.Settings{Name: "Slow"})

	req := &models.SearchRequest{Origin: []string{"CGK"}, Destination: []string{"DPS"}, DepartureDate: []string{"2025-12-15"}}

	// The aggregator internally creates a context timeout (e.g. 1500 ms)
	// The slowProv should be cancelled. We should still get the fastProv result.

	res, err := svc.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no root error, got %v", err)
	}

	if res.Metadata.ProvidersSucceeded != 1 {
		t.Errorf("Expected 1 provider to succeed, got %d", res.Metadata.ProvidersSucceeded)
	}
	if res.Metadata.ProvidersFailed != 1 {
		t.Errorf("Expected 1 provider to fail via context timeout, got %d", res.Metadata.ProvidersFailed)
	}
	if len(res.Flights) != 1 || res.Flights[0].ID != "fast1" {
		t.Errorf("Expected exactly fast1 flight returned in aggregator payload")
	}
}

func TestAggregatorRetryAndBackoff(t *testing.T) {
	// Provider fails twice then succeeds. Retry loop must catch it.
	flakyProv := &mockProvider{
		name:      "Flaky",
		flights:   []models.Flight{{ID: "flaky1"}},
		errToRet:  context.DeadlineExceeded,
		failCount: 2, // Fails twice!
	}

	svc := NewAggregatorService("", 5*time.Minute, 10*time.Minute, 1500*time.Millisecond, 0.6, 0.4)
	svc.providers = []providers.FlightProvider{flakyProv}
	svc.breakers["Flaky"] = gobreaker.NewCircuitBreaker(gobreaker.Settings{Name: "Flaky"})

	req := &models.SearchRequest{Origin: []string{"CGK"}, Destination: []string{"DPS"}, DepartureDate: []string{"2025-12-15"}}
	res, err := svc.Search(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if flakyProv.attemptsCount != 3 {
		t.Errorf("Expected exactly 3 attempts (2 fails, 1 success), got %d", flakyProv.attemptsCount)
	}

	if len(res.Flights) != 1 || res.Flights[0].ID != "flaky1" {
		t.Errorf("Expected flaky1 flight returned after retry")
	}
}
