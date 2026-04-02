package models

import (
	"encoding/json"
	"fmt"
)

// FlexStringArray allows JSON to be specified as a single string or an array of strings seamlessly.
type FlexStringArray []string

func (fsa *FlexStringArray) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*fsa = []string{single}
		return nil
	}

	var multi []string
	if err := json.Unmarshal(data, &multi); err == nil {
		*fsa = multi
		return nil
	}

	return fmt.Errorf("expected string or array of strings")
}

// SearchRequest represents the incoming JSON request
type SearchRequest struct {
	Origin        FlexStringArray `json:"origin"`
	Destination   FlexStringArray `json:"destination"`
	DepartureDate FlexStringArray `json:"departureDate"`
	ReturnDate    string          `json:"returnDate,omitempty"` // for simple round trips
	Passengers    int             `json:"passengers"`
	CabinClass    string          `json:"cabinClass"`

	// Query parameters for post-fetch filtering
	SortBy      string   `json:"sort_by,omitempty"` // default "best_value"
	MinPrice    int      `json:"min_price,omitempty"`
	MaxPrice    int      `json:"max_price,omitempty"`
	MaxStops    *int     `json:"max_stops,omitempty"` // pointer to allow 0 value check
	Airlines    []string `json:"airlines,omitempty"`
	MaxDuration int      `json:"max_duration,omitempty"`
}

type SearchLeg struct {
	Origin        string
	Destination   string
	DepartureDate string
}

// GetLegs processes the incoming parameters and returns a list of individual flight segments.
// It throws an error if multi-city locations/dates are mismatched, solving the 3-origins/1-date problem.
func (r *SearchRequest) GetLegs() ([]SearchLeg, error) {
	lenO := len(r.Origin)
	lenD := len(r.Destination)
	lenDates := len(r.DepartureDate)

	if lenO == 0 || lenD == 0 || lenDates == 0 {
		return nil, fmt.Errorf("origin, destination, and departureDate are required")
	}

	// 1. Simple One-Way or Round-Trip:
	if lenO == 1 && lenD == 1 {
		var legs []SearchLeg

		// Optional: user might mistakenly send an array of 2 departure dates instead of ReturnDate
		if lenDates > 1 {
			legs = append(legs, SearchLeg{Origin: r.Origin[0], Destination: r.Destination[0], DepartureDate: r.DepartureDate[0]})
			legs = append(legs, SearchLeg{Origin: r.Destination[0], Destination: r.Origin[0], DepartureDate: r.DepartureDate[1]}) // Assuming array means round trip
		} else {
			legs = append(legs, SearchLeg{Origin: r.Origin[0], Destination: r.Destination[0], DepartureDate: r.DepartureDate[0]})
			if r.ReturnDate != "" {
				legs = append(legs, SearchLeg{Origin: r.Destination[0], Destination: r.Origin[0], DepartureDate: r.ReturnDate})
			}
		}

		return legs, nil
	}

	// 2. Multi-City configuration
	// Validate length of origins and destinations match
	if lenO != lenD {
		return nil, fmt.Errorf("length mismatch: provided %d origins but %d destinations", lenO, lenD)
	}

	// Validate dates match leg count
	if lenDates != lenO {
		return nil, fmt.Errorf("multi-city validation failed: provided %d legs but %d departure dates. Number of dates MUST match number of legs", lenO, lenDates)
	}

	var legs []SearchLeg
	for i := 0; i < lenO; i++ {
		legs = append(legs, SearchLeg{
			Origin:        r.Origin[i],
			Destination:   r.Destination[i],
			DepartureDate: r.DepartureDate[i],
		})
	}

	// 3. Optional: Add Circuit Return for Multi-City if ReturnDate is provided
	if r.ReturnDate != "" {
		// Basic chronolgical validation
		lastLeg := legs[len(legs)-1]
		if r.ReturnDate < lastLeg.DepartureDate {
			return nil, fmt.Errorf("returnDate (%s) cannot be before last departureDate (%s)", r.ReturnDate, lastLeg.DepartureDate)
		}

		legs = append(legs, SearchLeg{
			Origin:        r.Destination[lenD-1],
			Destination:   r.Origin[0],
			DepartureDate: r.ReturnDate,
		})
	}

	return legs, nil
}

// SearchResponse represents the unified response format
type SearchResponse struct {
	SearchCriteria SearchCriteria `json:"search_criteria"`
	Metadata       Metadata       `json:"metadata"`
	Flights        []Flight       `json:"flights"`
}

type SearchCriteria struct {
	Origin        FlexStringArray `json:"origin"`
	Destination   FlexStringArray `json:"destination"`
	DepartureDate FlexStringArray `json:"departure_date"`
	ReturnDate    string          `json:"return_date,omitempty"`
	Passengers    int             `json:"passengers"`
	CabinClass    string          `json:"cabin_class"`
}

type Metadata struct {
	TotalResults       int   `json:"total_results"`
	TotalLegs          int   `json:"total_legs"`
	ProvidersQueried   int   `json:"providers_queried"`
	ProvidersSucceeded int   `json:"providers_succeeded"`
	ProvidersFailed    int   `json:"providers_failed"`
	SearchTimeMs       int64 `json:"search_time_ms"`
	CacheHit           bool  `json:"cache_hit"`
}
