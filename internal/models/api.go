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
	Origins       FlexStringArray `json:"origins,omitempty"`
	Destination   FlexStringArray `json:"destination"`
	Destinations  FlexStringArray `json:"destinations,omitempty"`
	DepartureDate FlexStringArray `json:"departureDate"`
	ReturnDate    FlexStringArray `json:"returnDate,omitempty"` // for simple round trips or matrix search
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
	// Consolidate aliases
	origins := r.Origin
	if len(origins) == 0 {
		origins = r.Origins
	}
	destinations := r.Destination
	if len(destinations) == 0 {
		destinations = r.Destinations
	}

	lenO := len(origins)
	lenD := len(destinations)
	lenDates := len(r.DepartureDate)

	if lenO == 0 || lenD == 0 || lenDates == 0 {
		return nil, fmt.Errorf("origin, destination, and departureDate are required")
	}

	var legs []SearchLeg

	// 1. Generate Outbound Matrix: Origin x Destination x Date
	for _, o := range origins {
		for _, d := range destinations {
			if o == d {
				continue // Skip identity routes
			}
			for _, date := range r.DepartureDate {
				legs = append(legs, SearchLeg{
					Origin:        o,
					Destination:   d,
					DepartureDate: date,
				})
			}
		}
	}

	// 2. Generate Inbound Matrix: Destination x Origin x ReturnDate
	for _, d := range destinations {
		for _, o := range origins {
			if d == o {
				continue // Skip identity routes
			}
			for _, rDate := range r.ReturnDate {
				// Basic chronological validation against first outbound date (simplified)
				if rDate != "" && len(r.DepartureDate) > 0 && rDate < r.DepartureDate[0] {
					continue
				}
				legs = append(legs, SearchLeg{
					Origin:        d,
					Destination:   o,
					DepartureDate: rDate,
				})
			}
		}
	}

	if len(legs) == 0 {
		return nil, fmt.Errorf("no valid flight legs generated from the given criteria")
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
	ReturnDate    FlexStringArray `json:"return_date,omitempty"`
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
