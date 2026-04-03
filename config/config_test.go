package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pandusatrianura/heimdall-travel-service/internal/pkg/utils"
)

var configEnvKeys = []string{
	"PORT",
	"MOCK_DATA_PATH",
	"MOCK_DATA_PROVIDER",
	"CACHE_TTL_MINUTES",
	"CACHE_CLEANUP_MINUTES",
	"PROVIDER_TIMEOUT_MS",
	"BEST_VALUE_PRICE_WEIGHT",
	"BEST_VALUE_DURATION_WEIGHT",
	"AIRASIA_DELAY_MS",
	"AIRASIA_FAILURE_RATE",
	"BATIK_AIR_DELAY_MS",
	"GARUDA_INDONESIA_DELAY_MS",
	"LION_AIR_DELAY_MS",
}

func TestLoadDefaultsWhenUnset(t *testing.T) {
	prepareEnv(t)
	withWorkingDir(t, t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "8080" {
		t.Fatalf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.MockDataPath != filepath.Join(mustGetwd(t), "mock_provider") {
		t.Fatalf("MockDataPath = %q, want %q", cfg.MockDataPath, filepath.Join(mustGetwd(t), "mock_provider"))
	}
	if cfg.CacheTTL != 5*time.Minute {
		t.Fatalf("CacheTTL = %v, want %v", cfg.CacheTTL, 5*time.Minute)
	}
	if cfg.CacheCleanup != 10*time.Minute {
		t.Fatalf("CacheCleanup = %v, want %v", cfg.CacheCleanup, 10*time.Minute)
	}
	if cfg.ProviderTimeout != 1500*time.Millisecond {
		t.Fatalf("ProviderTimeout = %v, want %v", cfg.ProviderTimeout, 1500*time.Millisecond)
	}
	if cfg.BestValuePriceWeight != 0.6 {
		t.Fatalf("BestValuePriceWeight = %v, want %v", cfg.BestValuePriceWeight, 0.6)
	}
	if cfg.BestValueDurationWeight != 0.4 {
		t.Fatalf("BestValueDurationWeight = %v, want %v", cfg.BestValueDurationWeight, 0.4)
	}
	if len(cfg.ProviderRuntime.MockDataFiles) != len(utils.DefaultMockFiles()) {
		t.Fatalf("MockDataFiles len = %d, want %d", len(cfg.ProviderRuntime.MockDataFiles), len(utils.DefaultMockFiles()))
	}
	if cfg.ProviderRuntime.AirAsia.DelayMS != 100 || cfg.ProviderRuntime.AirAsia.FailureRate != 10 {
		t.Fatalf("AirAsia runtime config = %+v, want delay=100 failure=10", cfg.ProviderRuntime.AirAsia)
	}
}

func TestLoadReadsFromOSEnvWithoutDotEnv(t *testing.T) {
	prepareEnv(t)
	withWorkingDir(t, t.TempDir())

	t.Setenv("PORT", "9090")
	t.Setenv("MOCK_DATA_PATH", "custom_mocks")
	t.Setenv("CACHE_TTL_MINUTES", "15")
	t.Setenv("CACHE_CLEANUP_MINUTES", "20")
	t.Setenv("PROVIDER_TIMEOUT_MS", "2200")
	t.Setenv("BEST_VALUE_PRICE_WEIGHT", "0.7")
	t.Setenv("BEST_VALUE_DURATION_WEIGHT", "0.3")
	t.Setenv("MOCK_DATA_PROVIDER", `["custom_one.json","custom_two.json"]`)
	t.Setenv("AIRASIA_DELAY_MS", "0")
	t.Setenv("AIRASIA_FAILURE_RATE", "25")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "9090" {
		t.Fatalf("Port = %q, want %q", cfg.Port, "9090")
	}
	if cfg.MockDataPath != filepath.Join(mustGetwd(t), "custom_mocks") {
		t.Fatalf("MockDataPath = %q, want %q", cfg.MockDataPath, filepath.Join(mustGetwd(t), "custom_mocks"))
	}
	if cfg.CacheTTL != 15*time.Minute {
		t.Fatalf("CacheTTL = %v, want %v", cfg.CacheTTL, 15*time.Minute)
	}
	if cfg.CacheCleanup != 20*time.Minute {
		t.Fatalf("CacheCleanup = %v, want %v", cfg.CacheCleanup, 20*time.Minute)
	}
	if cfg.ProviderTimeout != 2200*time.Millisecond {
		t.Fatalf("ProviderTimeout = %v, want %v", cfg.ProviderTimeout, 2200*time.Millisecond)
	}
	if cfg.BestValuePriceWeight != 0.7 {
		t.Fatalf("BestValuePriceWeight = %v, want %v", cfg.BestValuePriceWeight, 0.7)
	}
	if cfg.BestValueDurationWeight != 0.3 {
		t.Fatalf("BestValueDurationWeight = %v, want %v", cfg.BestValueDurationWeight, 0.3)
	}
	if len(cfg.ProviderRuntime.MockDataFiles) != 2 || cfg.ProviderRuntime.MockDataFiles[0] != "custom_one.json" {
		t.Fatalf("MockDataFiles = %v, want custom files", cfg.ProviderRuntime.MockDataFiles)
	}
	if cfg.ProviderRuntime.AirAsia.DelayMS != 0 || cfg.ProviderRuntime.AirAsia.FailureRate != 25 {
		t.Fatalf("AirAsia runtime config = %+v, want delay=0 failure=25", cfg.ProviderRuntime.AirAsia)
	}
}

func TestLoadReadsProviderRuntimeFromDotEnv(t *testing.T) {
	prepareEnv(t)
	tempDir := t.TempDir()
	withWorkingDir(t, tempDir)

	dotEnv := "PORT=9000\nMOCK_DATA_PATH=temp_mocks\nMOCK_DATA_PROVIDER=[\"lion_air_search_response.json\"]\nCACHE_TTL_MINUTES=7\nAIRASIA_DELAY_MS=0\nAIRASIA_FAILURE_RATE=100\n"
	if err := os.WriteFile(filepath.Join(tempDir, ".env"), []byte(dotEnv), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "9000" {
		t.Fatalf("Port = %q, want %q", cfg.Port, "9000")
	}
	if !samePath(cfg.MockDataPath, filepath.Join(tempDir, "temp_mocks")) {
		t.Fatalf("MockDataPath = %q, want path equivalent to %q", cfg.MockDataPath, filepath.Join(tempDir, "temp_mocks"))
	}
	if cfg.CacheTTL != 7*time.Minute {
		t.Fatalf("CacheTTL = %v, want %v", cfg.CacheTTL, 7*time.Minute)
	}
	if len(cfg.ProviderRuntime.MockDataFiles) != 1 || cfg.ProviderRuntime.MockDataFiles[0] != "lion_air_search_response.json" {
		t.Fatalf("MockDataFiles = %v, want lion_air_search_response.json", cfg.ProviderRuntime.MockDataFiles)
	}
	if cfg.ProviderRuntime.AirAsia.DelayMS != 0 || cfg.ProviderRuntime.AirAsia.FailureRate != 100 {
		t.Fatalf("AirAsia runtime config = %+v, want delay=0 failure=100", cfg.ProviderRuntime.AirAsia)
	}
}

func TestLoadOSEnvOverridesDotEnv(t *testing.T) {
	prepareEnv(t)
	tempDir := t.TempDir()
	withWorkingDir(t, tempDir)

	dotEnv := "PORT=9000\nBEST_VALUE_PRICE_WEIGHT=0.8\n"
	if err := os.WriteFile(filepath.Join(tempDir, ".env"), []byte(dotEnv), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("PORT", "9100")
	t.Setenv("BEST_VALUE_PRICE_WEIGHT", "0.9")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "9100" {
		t.Fatalf("Port = %q, want %q", cfg.Port, "9100")
	}
	if cfg.BestValuePriceWeight != 0.9 {
		t.Fatalf("BestValuePriceWeight = %v, want %v", cfg.BestValuePriceWeight, 0.9)
	}
}

func TestLoadInvalidNumericValuesFallBackToDefaults(t *testing.T) {
	prepareEnv(t)
	withWorkingDir(t, t.TempDir())

	t.Setenv("CACHE_TTL_MINUTES", "invalid")
	t.Setenv("CACHE_CLEANUP_MINUTES", "invalid")
	t.Setenv("PROVIDER_TIMEOUT_MS", "invalid")
	t.Setenv("BEST_VALUE_PRICE_WEIGHT", "invalid")
	t.Setenv("BEST_VALUE_DURATION_WEIGHT", "invalid")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.CacheTTL != 5*time.Minute {
		t.Fatalf("CacheTTL = %v, want %v", cfg.CacheTTL, 5*time.Minute)
	}
	if cfg.CacheCleanup != 10*time.Minute {
		t.Fatalf("CacheCleanup = %v, want %v", cfg.CacheCleanup, 10*time.Minute)
	}
	if cfg.ProviderTimeout != 1500*time.Millisecond {
		t.Fatalf("ProviderTimeout = %v, want %v", cfg.ProviderTimeout, 1500*time.Millisecond)
	}
	if cfg.BestValuePriceWeight != 0.6 {
		t.Fatalf("BestValuePriceWeight = %v, want %v", cfg.BestValuePriceWeight, 0.6)
	}
	if cfg.BestValueDurationWeight != 0.4 {
		t.Fatalf("BestValueDurationWeight = %v, want %v", cfg.BestValueDurationWeight, 0.4)
	}
}

func TestLoadAllowsZeroWeights(t *testing.T) {
	prepareEnv(t)
	withWorkingDir(t, t.TempDir())

	t.Setenv("BEST_VALUE_PRICE_WEIGHT", "0")
	t.Setenv("BEST_VALUE_DURATION_WEIGHT", "0")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.BestValuePriceWeight != 0 {
		t.Fatalf("BestValuePriceWeight = %v, want %v", cfg.BestValuePriceWeight, 0.0)
	}
	if cfg.BestValueDurationWeight != 0 {
		t.Fatalf("BestValueDurationWeight = %v, want %v", cfg.BestValueDurationWeight, 0.0)
	}
}

func prepareEnv(t *testing.T) {
	t.Helper()
	for _, key := range configEnvKeys {
		t.Setenv(key, "")
	}
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Fatalf("restore Chdir() error = %v", err)
		}
	})
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	return wd
}

func samePath(actual string, expected string) bool {
	actual = canonicalizePath(actual)
	expected = canonicalizePath(expected)
	return filepath.Clean(actual) == filepath.Clean(expected)
}

func canonicalizePath(path string) string {
	resolvedParent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return filepath.Clean(path)
	}

	return filepath.Join(resolvedParent, filepath.Base(path))
}
