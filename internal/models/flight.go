package models

type Airline struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type FlightPoint struct {
	Airport   string `json:"airport"`
	City      string `json:"city"`
	Datetime  string `json:"datetime"`
	Timestamp int64  `json:"timestamp"` // Unix timestamp
}

type Duration struct {
	TotalMinutes int    `json:"total_minutes"`
	Formatted    string `json:"formatted"`
}

type Price struct {
	Amount          int    `json:"amount"` // Base price + taxes if available
	FormattedAmount string `json:"formatted_amount,omitempty"`
	Currency        string `json:"currency"`
}

type Baggage struct {
	CarryOn string `json:"carry_on"`
	Checked string `json:"checked"`
}

// Flight is the normalized flight model that is returned in the SearchResponse
type Flight struct {
	ID             string      `json:"id"`
	Provider       string      `json:"provider"`
	Airline        Airline     `json:"airline"`
	FlightNumber   string      `json:"flight_number"`
	Departure      FlightPoint `json:"departure"`
	Arrival        FlightPoint `json:"arrival"`
	Duration       Duration    `json:"duration"`
	Stops          int         `json:"stops"`
	Price          Price       `json:"price"`
	AvailableSeats int         `json:"available_seats"`
	CabinClass     string      `json:"cabin_class"`
	Aircraft       *string     `json:"aircraft"`  // pointer to handle null
	Amenities      []string    `json:"amenities"` // can be empty slice
	Baggage        Baggage     `json:"baggage"`
}
