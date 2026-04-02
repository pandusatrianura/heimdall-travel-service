package models

import (
	"encoding/json"
	"fmt"
)

const (
	DirectionOutbound = "outbound"
	DirectionInbound  = "inbound"
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

func (fsa FlexStringArray) MarshalJSON() ([]byte, error) {
	if len(fsa) == 1 {
		return json.Marshal(fsa[0])
	}

	values := []string(fsa)
	if values == nil {
		values = []string{}
	}

	return json.Marshal(values)
}

// NullableStringArray allows JSON to be specified as null, a single string, or an array of strings/nulls.
// Null entries are normalized to empty strings so each index can still represent one trip item.
type NullableStringArray []string

func (nsa *NullableStringArray) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*nsa = nil
		return nil
	}

	var single *string
	if err := json.Unmarshal(data, &single); err == nil {
		if single == nil {
			*nsa = nil
			return nil
		}

		*nsa = []string{*single}
		return nil
	}

	var multi []*string
	if err := json.Unmarshal(data, &multi); err == nil {
		values := make([]string, len(multi))
		for index, value := range multi {
			if value != nil {
				values[index] = *value
			}
		}

		*nsa = values
		return nil
	}

	return fmt.Errorf("expected string, null, or array of strings/null")
}

func (nsa NullableStringArray) MarshalJSON() ([]byte, error) {
	if len(nsa) == 1 {
		if nsa[0] == "" {
			return []byte("null"), nil
		}

		return json.Marshal(nsa[0])
	}

	values := make([]*string, len(nsa))
	for index, value := range nsa {
		if value == "" {
			continue
		}

		current := value
		values[index] = &current
	}

	return json.Marshal(values)
}

// SearchRequest represents the incoming JSON request
type SearchRequest struct {
	Origin        FlexStringArray     `json:"origin"`
	Origins       FlexStringArray     `json:"origins,omitempty"`
	Destination   FlexStringArray     `json:"destination"`
	Destinations  FlexStringArray     `json:"destinations,omitempty"`
	DepartureDate FlexStringArray     `json:"departureDate"`
	ReturnDate    NullableStringArray `json:"returnDate,omitempty"`
	Passengers    int                 `json:"passengers"`
	CabinClass    string              `json:"cabinClass"`

	// Query parameters for post-fetch filtering
	SortBy      string   `json:"sort_by,omitempty"` // default "best_value"
	MinPrice    int      `json:"min_price,omitempty"`
	MaxPrice    int      `json:"max_price,omitempty"`
	MaxStops    *int     `json:"max_stops,omitempty"` // pointer to allow 0 value check
	Airlines    []string `json:"airlines,omitempty"`
	MaxDuration int      `json:"max_duration,omitempty"`
}

type TripItem struct {
	Index         int
	Origin        string
	Destination   string
	DepartureDate string
	ReturnDate    string
}

type SearchLeg struct {
	TripIndex     int
	Direction     string
	Origin        string
	Destination   string
	DepartureDate string
}

func (r *SearchRequest) normalizedOrigins() FlexStringArray {
	if len(r.Origin) > 0 {
		return r.Origin
	}

	return r.Origins
}

func (r *SearchRequest) normalizedDestinations() FlexStringArray {
	if len(r.Destination) > 0 {
		return r.Destination
	}

	return r.Destinations
}

func (r *SearchRequest) GetTripItems() ([]TripItem, error) {
	origins := r.normalizedOrigins()
	destinations := r.normalizedDestinations()
	departureDates := r.DepartureDate
	returnDates := r.ReturnDate

	tripCount := len(origins)
	if tripCount == 0 || len(destinations) == 0 || len(departureDates) == 0 {
		return nil, fmt.Errorf("origin, destination, and departureDate are required")
	}

	if len(destinations) != tripCount || len(departureDates) != tripCount {
		return nil, fmt.Errorf("origins, destinations, and departureDate must have the same number of items")
	}

	if len(returnDates) > 0 && len(returnDates) != tripCount {
		return nil, fmt.Errorf("returnDate must be empty or have the same number of items as the route arrays")
	}

	items := make([]TripItem, 0, tripCount)
	for index := range origins {
		origin := origins[index]
		destination := destinations[index]
		departureDate := departureDates[index]
		returnDate := ""
		if len(returnDates) == tripCount {
			returnDate = returnDates[index]
		}

		if origin == "" || destination == "" || departureDate == "" {
			return nil, fmt.Errorf("origin, destination, and departureDate are required for each trip item")
		}

		if origin == destination {
			return nil, fmt.Errorf("origin and destination cannot be the same for trip item %d", index)
		}

		if returnDate != "" && returnDate < departureDate {
			return nil, fmt.Errorf("returnDate cannot be earlier than departureDate for trip item %d", index)
		}

		items = append(items, TripItem{
			Index:         index,
			Origin:        origin,
			Destination:   destination,
			DepartureDate: departureDate,
			ReturnDate:    returnDate,
		})
	}

	return items, nil
}

// GetLegs processes the incoming parameters and returns trip-indexed flight segments.
func (r *SearchRequest) GetLegs() ([]SearchLeg, error) {
	items, err := r.GetTripItems()
	if err != nil {
		return nil, err
	}

	legs := make([]SearchLeg, 0, len(items)*2)
	for _, item := range items {
		legs = append(legs, SearchLeg{
			TripIndex:     item.Index,
			Direction:     DirectionOutbound,
			Origin:        item.Origin,
			Destination:   item.Destination,
			DepartureDate: item.DepartureDate,
		})

		if item.ReturnDate != "" {
			legs = append(legs, SearchLeg{
				TripIndex:     item.Index,
				Direction:     DirectionInbound,
				Origin:        item.Destination,
				Destination:   item.Origin,
				DepartureDate: item.ReturnDate,
			})
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
	Origin        FlexStringArray     `json:"origin"`
	Destination   FlexStringArray     `json:"destination"`
	DepartureDate FlexStringArray     `json:"departure_date"`
	ReturnDate    NullableStringArray `json:"return_date,omitempty"`
	Passengers    int                 `json:"passengers"`
	CabinClass    string              `json:"cabin_class"`
}

type Metadata struct {
	TotalResults       int   `json:"total_results"`
	ProvidersQueried   int   `json:"providers_queried"`
	ProvidersSucceeded int   `json:"providers_succeeded"`
	ProvidersFailed    int   `json:"providers_failed"`
	SearchTimeMs       int64 `json:"search_time_ms"`
	CacheHit           bool  `json:"cache_hit"`
}
