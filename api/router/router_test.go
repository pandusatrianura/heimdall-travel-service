package router

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pandusatrianura/heimdall-travel-service/config"
	"github.com/pandusatrianura/heimdall-travel-service/internal/pkg/utils"
	"github.com/pandusatrianura/heimdall-travel-service/internal/providers"
)

func TestBuildRegistersHealthRoute(t *testing.T) {
	mux := Build(testConfig(t))
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp := httptest.NewRecorder()

	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
	if contentType := resp.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", contentType, "application/json")
	}
	if body := resp.Body.String(); body != `{"status":"ok"}` {
		t.Fatalf("body = %q, want %q", body, `{"status":"ok"}`)
	}
}

func TestBuildRegistersSearchRoute(t *testing.T) {
	mux := Build(testConfig(t))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search", strings.NewReader("{"))
	resp := httptest.NewRecorder()

	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusBadRequest)
	}
}

func TestBuildRejectsWrongMethodForSearch(t *testing.T) {
	mux := Build(testConfig(t))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
	resp := httptest.NewRecorder()

	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusMethodNotAllowed)
	}
}

func testConfig(t *testing.T) config.Config {
	t.Helper()
	return config.Config{
		Port:                    "8080",
		MockDataPath:            filepath.Join(t.TempDir(), "mock_provider"),
		CacheTTL:                5 * time.Minute,
		CacheCleanup:            10 * time.Minute,
		ProviderTimeout:         1500 * time.Millisecond,
		BestValuePriceWeight:    0.6,
		BestValueDurationWeight: 0.4,
		ProviderRuntime: providers.RuntimeConfig{
			MockDataFiles:   utils.DefaultMockFiles(),
			AirAsia:         providers.ProviderRuntimeConfig{DelayMS: 100, FailureRate: 10},
			BatikAir:        providers.ProviderRuntimeConfig{DelayMS: 200},
			GarudaIndonesia: providers.ProviderRuntimeConfig{DelayMS: 50},
			LionAir:         providers.ProviderRuntimeConfig{DelayMS: 150},
		},
	}
}
