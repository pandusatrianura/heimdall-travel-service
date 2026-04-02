package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pandusatrianura/heimdall-travel-service/internal/models"
	"github.com/pandusatrianura/heimdall-travel-service/internal/providers"
	"github.com/patrickmn/go-cache"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
)

type AggregatorService struct {
	providers       []providers.FlightProvider
	cache           *cache.Cache
	providerTimeout time.Duration
	priceWeight     float64
	durationWeight  float64
	breakers        map[string]*gobreaker.CircuitBreaker
	limiters        map[string]*rate.Limiter
}

func NewAggregatorService(mockDataPath string, ttl, cleanup time.Duration, providerTimeout time.Duration, pWeight, dWeight float64) *AggregatorService {
	// Initialize cache with configured expiration and cleanup
	c := cache.New(ttl, cleanup)

	provs := []providers.FlightProvider{
		providers.NewGarudaProvider(mockDataPath),
		providers.NewLionAirProvider(mockDataPath),
		providers.NewBatikAirProvider(mockDataPath),
		providers.NewAirAsiaProvider(mockDataPath),
	}

	// Initialize Circuit Breakers and Rate Limiters for each provider
	breakers := make(map[string]*gobreaker.CircuitBreaker)
	limiters := make(map[string]*rate.Limiter)
	for _, p := range provs {
		name := p.Name()
		// Circuit Breaker
		breakers[name] = gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:        name,
			MaxRequests: 1,
			Timeout:     30 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= 5
			},
			OnStateChange: func(name string, from, to gobreaker.State) {
				slog.Warn("Circuit Breaker State Change",
					"provider", name,
					"from", from.String(),
					"to", to.String())
			},
		})
		// Burst of 10 requests, refilling at 10 requests per second
		limiters[name] = rate.NewLimiter(rate.Limit(10), 10)
	}

	return &AggregatorService{
		providers:       provs,
		cache:           c,
		providerTimeout: providerTimeout,
		priceWeight:     pWeight,
		durationWeight:  dWeight,
		breakers:        breakers,
		limiters:        limiters,
	}
}

// GenerateCacheKey creates a unique string cache key based on search params
func GenerateCacheKey(req *models.SearchRequest) string {
	return fmt.Sprintf("%s_%s_%s_%s_%d_%s",
		req.Origin, req.Destination, req.DepartureDate, req.ReturnDate, req.Passengers, req.CabinClass)
}

func (s *AggregatorService) Search(ctx context.Context, req *models.SearchRequest) (*models.SearchResponse, error) {
	slog.InfoContext(ctx, "Starting Aggregator search",
		"origin", req.Origin,
		"destination", req.Destination,
		"date", req.DepartureDate)
	start := time.Now()

	// 1. Check Cache
	cacheKey := GenerateCacheKey(req)
	if cachedData, found := s.cache.Get(cacheKey); found {
		slog.InfoContext(ctx, "Serving from Memory Cache", "cache_key", cacheKey)
		resp := cachedData.(*models.SearchResponse)
		// Update cache hit explicitly for this request payload
		respCopy := *resp
		respCopy.Metadata.CacheHit = true
		respCopy.Metadata.SearchTimeMs = time.Since(start).Milliseconds()
		return &respCopy, nil
	}

	// Generate Flight Legs (resolves Round-Trip / Multi-City logic)
	legs, err := req.GetLegs()
	if err != nil {
		return nil, fmt.Errorf("failed to process legs: %v", err)
	}

	// 2. Fetch from Providers Concurrently with Timeout Context across all Legs
	fetchCtx, cancel := context.WithTimeout(ctx, s.providerTimeout)
	defer cancel()

	var wg sync.WaitGroup
	numTasks := len(s.providers) * len(legs)
	resultsChan := make(chan []models.Flight, numTasks)
	successCountChan := make(chan int, numTasks)

	for _, reqLeg := range legs {
		l := reqLeg
		for _, p := range s.providers {
			wg.Add(1)
			go func(prov providers.FlightProvider, currentLeg models.SearchLeg) {
				defer wg.Done()

				cb := s.breakers[prov.Name()]
				limiter := s.limiters[prov.Name()]

				// Secure execution with Resource Throttling
				if limiter != nil {
					if err := limiter.Wait(fetchCtx); err != nil {
						slog.WarnContext(ctx, "Rate limit exceeded or timeout waiting", "provider", prov.Name(), "error", err.Error())
						successCountChan <- 0
						return
					}
				}

				var result interface{}
				var err error

				if cb != nil {
					// Execute through Circuit Breaker
					result, err = cb.Execute(func() (interface{}, error) {
						flights, success := s.fetchWithRetry(fetchCtx, prov, &currentLeg)
						if !success {
							return nil, fmt.Errorf("provider %s failed or timed out", prov.Name())
						}
						return flights, nil
					})
				} else {
					// Bypass CB
					flights, success := s.fetchWithRetry(fetchCtx, prov, &currentLeg)
					if !success {
						err = fmt.Errorf("provider %s failed or timed out", prov.Name())
					} else {
						result = flights
					}
				}

				if err == nil {
					successCountChan <- 1
					resultsChan <- result.([]models.Flight)
				} else {
					slog.WarnContext(ctx, "Provider skip or failure",
						"provider", prov.Name(),
						"error", err.Error())
					successCountChan <- 0
				}
			}(p, l)
		}
	}

	wg.Wait()
	close(resultsChan)
	close(successCountChan)

	// Collect results
	var allFlights []models.Flight
	for flights := range resultsChan {
		allFlights = append(allFlights, flights...)
	}

	successful := 0
	for count := range successCountChan {
		successful += count
	}
	failed := numTasks - successful

	slog.InfoContext(ctx, "Gathered flights phase complete",
		"successful_apis", successful,
		"failed_apis", failed,
		"total_raw_flights", len(allFlights))

	// 3. Post-Process (Filter / Normalize / Sort)
	allFlights = s.FilterAndSortFlights(allFlights, req, ctx)

	// 4. Construct Response
	resp := &models.SearchResponse{
		SearchCriteria: models.SearchCriteria{
			Origin:        req.Origin,
			Destination:   req.Destination,
			DepartureDate: req.DepartureDate,
			ReturnDate:    req.ReturnDate,
			Passengers:    req.Passengers,
			CabinClass:    req.CabinClass,
		},
		Metadata: models.Metadata{
			TotalResults:       len(allFlights),
			TotalLegs:          len(legs),
			ProvidersQueried:   len(s.providers),
			ProvidersSucceeded: successful,
			ProvidersFailed:    failed,
			SearchTimeMs:       time.Since(start).Milliseconds(),
			CacheHit:           false,
		},
		Flights: allFlights,
	}

	// Ensure empty slice is initialized as [] instead of null in JSON payload
	if resp.Flights == nil {
		resp.Flights = []models.Flight{}
	}

	// Save to cache
	s.cache.Set(cacheKey, resp, cache.DefaultExpiration)

	return resp, nil
}

// fetchWithRetry executes provider search, handling explicit retries with exp backoff for failing providers (e.g. AirAsia)
func (s *AggregatorService) fetchWithRetry(ctx context.Context, p providers.FlightProvider, leg *models.SearchLeg) ([]models.Flight, bool) {
	maxRetries := 3
	baseBackoff := 50 * time.Millisecond

	for i := 0; i <= maxRetries; i++ {
		flights, err := p.SearchFlights(ctx, leg)
		if err == nil {
			return flights, true // success
		}

		// Context timeout or cancellation means we abort retries immediately
		if ctx.Err() != nil {
			return nil, false
		}

		// Prepare exponential backoff
		if i < maxRetries {
			backoffDelay := baseBackoff * (1 << i) // 50, 100, 200 ms
			select {
			case <-time.After(backoffDelay):
			case <-ctx.Done():
				return nil, false
			}
		}
	}

	// Failed completely
	return nil, false
}

// FilterAndSortFlights applies search filters and sorting based on the request.
func (s *AggregatorService) FilterAndSortFlights(flights []models.Flight, req *models.SearchRequest, ctx context.Context) []models.Flight {
	slog.InfoContext(ctx, "Starting FilterAndSortFlights", "initial_count", len(flights))
	var filtered []models.Flight

	// 1. Filtering
	for _, f := range flights {
		if req.MinPrice > 0 && f.Price.Amount < req.MinPrice {
			continue
		}
		if req.MaxPrice > 0 && f.Price.Amount > req.MaxPrice {
			continue
		}
		if req.MaxStops != nil && f.Stops > *req.MaxStops {
			continue
		}
		if req.MaxDuration > 0 && f.Duration.TotalMinutes > req.MaxDuration {
			continue
		}
		if len(req.Airlines) > 0 {
			matchAirline := false
			for _, a := range req.Airlines {
				if f.Airline.Code == a || f.Airline.Name == a || f.Provider == a {
					matchAirline = true
					break
				}
			}
			if !matchAirline {
				continue
			}
		}

		// IDR formatting injection
		f.Price.FormattedAmount = fmt.Sprintf("Rp %s", formatThousands(f.Price.Amount))

		filtered = append(filtered, f)
	}

	// 2. Sorting
	sortBy := "best_value"
	if req.SortBy != "" {
		sortBy = req.SortBy
	}

	switch sortBy {
	case "price_lowest":
		sort.SliceStable(filtered, func(i, j int) bool {
			return filtered[i].Price.Amount < filtered[j].Price.Amount
		})
	case "price_highest":
		sort.SliceStable(filtered, func(i, j int) bool {
			return filtered[i].Price.Amount > filtered[j].Price.Amount
		})
	case "duration_shortest":
		sort.SliceStable(filtered, func(i, j int) bool {
			return filtered[i].Duration.TotalMinutes < filtered[j].Duration.TotalMinutes
		})
	case "duration_longest":
		sort.SliceStable(filtered, func(i, j int) bool {
			return filtered[i].Duration.TotalMinutes > filtered[j].Duration.TotalMinutes
		})
	case "departure_earliest":
		sort.SliceStable(filtered, func(i, j int) bool {
			return filtered[i].Departure.Timestamp < filtered[j].Departure.Timestamp
		})
	case "arrival_earliest":
		sort.SliceStable(filtered, func(i, j int) bool {
			return filtered[i].Arrival.Timestamp < filtered[j].Arrival.Timestamp
		})
	default: // best_value
		s.applyBestValueSort(filtered)
	}

	slog.InfoContext(ctx, "FilterAndSortFlights complete",
		"post_filter_count", len(filtered),
		"sorted_by", sortBy)

	return filtered
}

func formatThousands(n int) string {
	in := fmt.Sprintf("%d", n)
	parts := []string{}

	start := 0
	if in[0] == '-' {
		start = 1
	}

	length := len(in) - start
	for i := 0; i < length; i++ {
		if i > 0 && i%3 == 0 {
			parts = append([]string{"."}, parts...)
		}
		parts = append([]string{string(in[len(in)-1-i])}, parts...)
	}

	res := strings.Join(parts, "")
	if start == 1 {
		res = "-" + res
	}
	return res
}

// applyBestValueSort applies a combined normalized score for price and duration.
func (s *AggregatorService) applyBestValueSort(flights []models.Flight) {
	if len(flights) == 0 {
		return
	}

	// Find min/max boundaries to normalize scores
	minPrice, maxPrice := float64(^uint(0)>>1), float64(0)
	minDur, maxDur := float64(^uint(0)>>1), float64(0)

	for _, f := range flights {
		p := float64(f.Price.Amount)
		d := float64(f.Duration.TotalMinutes)

		if p < minPrice {
			minPrice = p
		}
		if p > maxPrice {
			maxPrice = p
		}
		if d < minDur {
			minDur = d
		}
		if d > maxDur {
			maxDur = d
		}
	}

	// Function to calculate best value score. Value from 0.0 (Best) to 1.0 (Worst).
	// We want cheap and fast -> Low score.
	calculateScore := func(f models.Flight) float64 {
		p := float64(f.Price.Amount)
		d := float64(f.Duration.TotalMinutes)

		// Normalize Price (0.0=minPrice, 1.0=maxPrice)
		nPrice := 0.0
		if maxPrice > minPrice {
			nPrice = (p - minPrice) / (maxPrice - minPrice)
		}

		// Normalize Duration (0.0=minDur, 1.0=maxDur)
		nDur := 0.0
		if maxDur > minDur {
			nDur = (d - minDur) / (maxDur - minDur)
		}

		// Weightings from configuration
		return (s.priceWeight * nPrice) + (s.durationWeight * nDur)
	}

	sort.SliceStable(flights, func(i, j int) bool {
		return calculateScore(flights[i]) < calculateScore(flights[j])
	})
}
