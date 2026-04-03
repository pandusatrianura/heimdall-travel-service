package models

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestFlexStringArrayUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    FlexStringArray
		wantErr bool
	}{
		{name: "single string", input: `"CGK"`, want: FlexStringArray{"CGK"}},
		{name: "array", input: `["CGK","DPS"]`, want: FlexStringArray{"CGK", "DPS"}},
		{name: "empty array", input: `[]`, want: FlexStringArray{}},
		{name: "null becomes single empty string", input: `null`, want: FlexStringArray{""}},
		{name: "number is invalid", input: `123`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got FlexStringArray
			err := json.Unmarshal([]byte(tt.input), &got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("json.Unmarshal() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestFlexStringArrayMarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input FlexStringArray
		want  string
	}{
		{name: "single element marshals as string", input: FlexStringArray{"CGK"}, want: `"CGK"`},
		{name: "multiple elements marshal as array", input: FlexStringArray{"CGK", "DPS"}, want: `["CGK","DPS"]`},
		{name: "empty marshals as empty array", input: FlexStringArray{}, want: `[]`},
		{name: "nil marshals as empty array", input: nil, want: `[]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			if string(got) != tt.want {
				t.Fatalf("json.Marshal() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestNullableStringArrayUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    NullableStringArray
		wantErr bool
	}{
		{name: "null becomes nil", input: `null`, want: nil},
		{name: "single string", input: `"2025-12-18"`, want: NullableStringArray{"2025-12-18"}},
		{name: "array with nulls", input: `["2025-12-18",null,"2025-12-20"]`, want: NullableStringArray{"2025-12-18", "", "2025-12-20"}},
		{name: "all nulls", input: `[null,null]`, want: NullableStringArray{"", ""}},
		{name: "number is invalid", input: `123`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got NullableStringArray
			err := json.Unmarshal([]byte(tt.input), &got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("json.Unmarshal() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestNullableStringArrayMarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input NullableStringArray
		want  string
	}{
		{name: "single non empty marshals as string", input: NullableStringArray{"2025-12-18"}, want: `"2025-12-18"`},
		{name: "single empty marshals as null", input: NullableStringArray{""}, want: `null`},
		{name: "array with empty string marshals to null entry", input: NullableStringArray{"2025-12-18", "", "2025-12-20"}, want: `["2025-12-18",null,"2025-12-20"]`},
		{name: "nil marshals as empty array", input: nil, want: `[]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			if string(got) != tt.want {
				t.Fatalf("json.Marshal() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestSearchRequest_GetLegs_PositionalTripItems(t *testing.T) {
	t.Run("one way emits one outbound leg", func(t *testing.T) {
		req := &SearchRequest{
			Origins:       []string{"CGK"},
			Destinations:  []string{"DPS"},
			DepartureDate: []string{"2025-12-15"},
			ReturnDate:    nil,
		}

		legs, err := req.GetLegs()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(legs) != 1 {
			t.Fatalf("expected 1 leg, got %d", len(legs))
		}

		if legs[0].TripIndex != 0 {
			t.Fatalf("expected trip index 0, got %d", legs[0].TripIndex)
		}

		if legs[0].Direction != "outbound" {
			t.Fatalf("expected outbound direction, got %s", legs[0].Direction)
		}

		if legs[0].Origin != "CGK" || legs[0].Destination != "DPS" || legs[0].DepartureDate != "2025-12-15" {
			t.Fatalf("unexpected leg generated: %#v", legs[0])
		}
	})

	t.Run("round trip emits outbound and inbound for same item", func(t *testing.T) {
		req := &SearchRequest{
			Origins:       []string{"CGK"},
			Destinations:  []string{"DPS"},
			DepartureDate: []string{"2025-12-15"},
			ReturnDate:    []string{"2025-12-18"},
		}

		legs, err := req.GetLegs()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(legs) != 2 {
			t.Fatalf("expected 2 legs, got %d", len(legs))
		}

		if legs[0].TripIndex != 0 || legs[1].TripIndex != 0 {
			t.Fatalf("expected both legs to belong to trip index 0, got %#v", legs)
		}

		if legs[0].Direction != "outbound" || legs[1].Direction != "inbound" {
			t.Fatalf("expected outbound/inbound directions, got %#v", legs)
		}

		if legs[1].Origin != "DPS" || legs[1].Destination != "CGK" || legs[1].DepartureDate != "2025-12-18" {
			t.Fatalf("unexpected inbound leg generated: %#v", legs[1])
		}
	})

	t.Run("multi city uses positional pairing instead of matrix expansion", func(t *testing.T) {
		req := &SearchRequest{
			Origins:       []string{"CGK", "SUB"},
			Destinations:  []string{"DPS", "SIN"},
			DepartureDate: []string{"2025-12-15", "2025-12-20"},
			ReturnDate:    []string{"2025-12-25", "2025-12-26"},
		}

		legs, err := req.GetLegs()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(legs) != 4 {
			t.Fatalf("expected 4 legs from 2 round-trip items, got %d", len(legs))
		}

		expected := []SearchLeg{
			{TripIndex: 0, Direction: "outbound", Origin: "CGK", Destination: "DPS", DepartureDate: "2025-12-15"},
			{TripIndex: 0, Direction: "inbound", Origin: "DPS", Destination: "CGK", DepartureDate: "2025-12-25"},
			{TripIndex: 1, Direction: "outbound", Origin: "SUB", Destination: "SIN", DepartureDate: "2025-12-20"},
			{TripIndex: 1, Direction: "inbound", Origin: "SIN", Destination: "SUB", DepartureDate: "2025-12-26"},
		}

		for index, leg := range expected {
			if legs[index] != leg {
				t.Fatalf("unexpected leg at index %d: got %#v want %#v", index, legs[index], leg)
			}
		}
	})

	t.Run("mixed one way and round trip keeps per item semantics", func(t *testing.T) {
		req := &SearchRequest{
			Origins:       []string{"CGK", "SUB"},
			Destinations:  []string{"DPS", "SIN"},
			DepartureDate: []string{"2025-12-15", "2025-12-20"},
			ReturnDate:    []string{"", "2025-12-26"},
		}

		legs, err := req.GetLegs()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(legs) != 3 {
			t.Fatalf("expected 3 legs, got %d", len(legs))
		}

		if legs[0].TripIndex != 0 || legs[0].Direction != "outbound" {
			t.Fatalf("unexpected first leg: %#v", legs[0])
		}

		if legs[1].TripIndex != 1 || legs[1].Direction != "outbound" {
			t.Fatalf("unexpected second leg: %#v", legs[1])
		}

		if legs[2].TripIndex != 1 || legs[2].Direction != "inbound" {
			t.Fatalf("unexpected third leg: %#v", legs[2])
		}
	})

	t.Run("mismatched positional array lengths fail", func(t *testing.T) {
		req := &SearchRequest{
			Origins:       []string{"CGK", "SUB"},
			Destinations:  []string{"DPS"},
			DepartureDate: []string{"2025-12-15", "2025-12-20"},
		}

		_, err := req.GetLegs()
		if err == nil {
			t.Fatalf("expected mismatched length error")
		}
	})

	t.Run("identity route fails validation", func(t *testing.T) {
		req := &SearchRequest{
			Origins:       []string{"CGK"},
			Destinations:  []string{"CGK"},
			DepartureDate: []string{"2025-12-15"},
		}

		_, err := req.GetLegs()
		if err == nil {
			t.Fatalf("expected identity route validation error")
		}
	})
}
