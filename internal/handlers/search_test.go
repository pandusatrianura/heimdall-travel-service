package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pandusatrianura/heimdall-travel-service/internal/models"
	"github.com/pandusatrianura/heimdall-travel-service/internal/services"
)

func TestSearchHandler_HandleSearch(t *testing.T) {
	// Initialize real aggregator but pointing to local mock logic
	mockDataPath := filepath.Join("..", "..", "mock_provider")
	aggregatorSvc := services.NewAggregatorService(mockDataPath, 5*time.Minute, 10*time.Minute, 1500*time.Millisecond, 0.6, 0.4)
	handler := NewSearchHandler(aggregatorSvc)

	t.Run("Valid Request", func(t *testing.T) {
		reqBody := models.SearchRequest{
			Origin:        []string{"CGK"},
			Destination:   []string{"DPS"},
			DepartureDate: []string{"2025-12-15"},
			Passengers:    1,
			CabinClass:    "economy",
		}

		b, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/search?sort_by=price_lowest", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.HandleSearch(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		bodyBytes := rr.Body.Bytes()

		// check response body briefly
		var resp models.SearchResponse
		if err := json.Unmarshal(bodyBytes, &resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp.Metadata.TotalResults == 0 {
			t.Errorf("expected >0 total results")
		}

		if len(resp.Flights) == 0 {
			t.Fatalf("expected flights in flat response")
		}

		var raw map[string]any
		if err := json.Unmarshal(bodyBytes, &raw); err != nil {
			t.Fatalf("failed to decode raw response: %v", err)
		}

		searchCriteria, ok := raw["search_criteria"].(map[string]any)
		if !ok {
			t.Fatalf("expected search_criteria object in response")
		}

		if _, exists := raw["trips"]; exists {
			t.Fatalf("did not expect trips field in flat response")
		}

		if origin, ok := searchCriteria["origin"].(string); !ok || origin != "CGK" {
			t.Fatalf("expected scalar origin in search_criteria, got %#v", searchCriteria["origin"])
		}

		if destination, ok := searchCriteria["destination"].(string); !ok || destination != "DPS" {
			t.Fatalf("expected scalar destination in search_criteria, got %#v", searchCriteria["destination"])
		}

		// Check sorting applies
		if len(resp.Flights) > 1 {
			if resp.Flights[0].Price.Amount > resp.Flights[1].Price.Amount {
				t.Errorf("List not sorted by price_lowest correctly")
			}
		}

		t.Run("Invalid Positional Array Lengths Return Bad Request", func(t *testing.T) {
			reqBody := models.SearchRequest{
				Origins:       []string{"CGK", "SUB"},
				Destinations:  []string{"DPS"},
				DepartureDate: []string{"2025-12-15", "2025-12-20"},
			}

			b, _ := json.Marshal(reqBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBuffer(b))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.HandleSearch(rr, req)

			if status := rr.Code; status != http.StatusBadRequest {
				t.Fatalf("handler returned wrong status code for positional mismatch: got %v want %v", status, http.StatusBadRequest)
			}

			if !strings.Contains(rr.Body.String(), "must have the same number of items") {
				t.Fatalf("expected positional mismatch error message, got %s", rr.Body.String())
			}
		})

		t.Run("Identity Route Returns Bad Request", func(t *testing.T) {
			reqBody := models.SearchRequest{
				Origin:        []string{"CGK"},
				Destination:   []string{"CGK"},
				DepartureDate: []string{"2025-12-15"},
			}

			b, _ := json.Marshal(reqBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBuffer(b))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.HandleSearch(rr, req)

			if status := rr.Code; status != http.StatusBadRequest {
				t.Fatalf("handler returned wrong status code for identity route: got %v want %v", status, http.StatusBadRequest)
			}

			if !strings.Contains(rr.Body.String(), "origin and destination cannot be the same") {
				t.Fatalf("expected identity route error message, got %s", rr.Body.String())
			}
		})
	})

	t.Run("End-to-End Filter Integration", func(t *testing.T) {
		reqBody := models.SearchRequest{
			Origin:        []string{"CGK"},
			Destination:   []string{"DPS"},
			DepartureDate: []string{"2025-12-15"},
		}

		b, _ := json.Marshal(reqBody)
		// Request specific airlines, under max_price, and 0 stops
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/search?airlines=Garuda Indonesia&max_price=2000000&max_stops=0", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.HandleSearch(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var resp models.SearchResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Ensure payload Metadata validates Provider count accurately
		if resp.Metadata.ProvidersQueried != 4 {
			t.Errorf("Expected 4 providers queried in architecture, got %d", resp.Metadata.ProvidersQueried)
		}

		// If Garuda parsed correctly, we should have results
		if resp.Metadata.TotalResults == 0 {
			t.Log("Note: Zero results from Garuda mock - check if mock path is reachable from handler test.")
		} else {
			// Test IDR string formatting inside handler payload explicitly
			if resp.Flights[0].Price.FormattedAmount == "" || string(resp.Flights[0].Price.FormattedAmount[0:3]) != "Rp " {
				t.Errorf("IDR Formatting failed, got: %s", resp.Flights[0].Price.FormattedAmount)
			}
		}
	})

	t.Run("Invalid Missing Fields Request", func(t *testing.T) {
		// Missing Destination
		reqBody := models.SearchRequest{
			Origin:        []string{"CGK"},
			DepartureDate: []string{"2025-12-15"},
		}

		b, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBuffer(b))

		rr := httptest.NewRecorder()
		handler.HandleSearch(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code for invalid req: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("Invalid JSON Body", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBuffer([]byte("{invalid json}")))

		rr := httptest.NewRecorder()
		handler.HandleSearch(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code for invalid JSON: got %v want %v", status, http.StatusBadRequest)
		}
	})
}
