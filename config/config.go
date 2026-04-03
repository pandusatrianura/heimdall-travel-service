package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pandusatrianura/heimdall-travel-service/internal/pkg/utils"
	"github.com/pandusatrianura/heimdall-travel-service/internal/providers"
	"github.com/spf13/viper"
)

const (
	defaultPort                    = "8080"
	defaultMockDataPath            = "mock_provider"
	defaultCacheTTLMinutes         = 5
	defaultCacheCleanupMinutes     = 10
	defaultProviderTimeoutMS       = 1500
	defaultBestValuePriceWeight    = 0.6
	defaultBestValueDurationWeight = 0.4
)

type Config struct {
	Port                    string
	MockDataPath            string
	CacheTTL                time.Duration
	CacheCleanup            time.Duration
	ProviderTimeout         time.Duration
	BestValuePriceWeight    float64
	BestValueDurationWeight float64
	ProviderRuntime         providers.RuntimeConfig
}

func Load() (Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Config{}, fmt.Errorf("get working directory: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(filepath.Join(cwd, ".env"))
	v.SetConfigType("env")
	v.AutomaticEnv()
	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		var configFileNotFound viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFound) && !os.IsNotExist(err) {
			return Config{}, fmt.Errorf("read .env config: %w", err)
		}
	}

	mockDataPath := strings.TrimSpace(v.GetString("MOCK_DATA_PATH"))
	if !filepath.IsAbs(mockDataPath) {
		mockDataPath = filepath.Join(cwd, mockDataPath)
	}
	mockDataFiles := getStringSlice(v, "MOCK_DATA_PROVIDER", utils.DefaultMockFiles())

	return Config{
		Port:                    strings.TrimSpace(v.GetString("PORT")),
		MockDataPath:            mockDataPath,
		CacheTTL:                time.Duration(getNonZeroInt(v, "CACHE_TTL_MINUTES", defaultCacheTTLMinutes)) * time.Minute,
		CacheCleanup:            time.Duration(getNonZeroInt(v, "CACHE_CLEANUP_MINUTES", defaultCacheCleanupMinutes)) * time.Minute,
		ProviderTimeout:         time.Duration(getNonZeroInt(v, "PROVIDER_TIMEOUT_MS", defaultProviderTimeoutMS)) * time.Millisecond,
		BestValuePriceWeight:    getFloat(v, "BEST_VALUE_PRICE_WEIGHT", defaultBestValuePriceWeight),
		BestValueDurationWeight: getFloat(v, "BEST_VALUE_DURATION_WEIGHT", defaultBestValueDurationWeight),
		ProviderRuntime: providers.RuntimeConfig{
			MockDataFiles:   mockDataFiles,
			AirAsia:         providers.ProviderRuntimeConfig{DelayMS: getInt(v, "AIRASIA_DELAY_MS", 100), FailureRate: getInt(v, "AIRASIA_FAILURE_RATE", 10)},
			BatikAir:        providers.ProviderRuntimeConfig{DelayMS: getInt(v, "BATIK_AIR_DELAY_MS", 200)},
			GarudaIndonesia: providers.ProviderRuntimeConfig{DelayMS: getInt(v, "GARUDA_INDONESIA_DELAY_MS", 50)},
			LionAir:         providers.ProviderRuntimeConfig{DelayMS: getInt(v, "LION_AIR_DELAY_MS", 150)},
		},
	}, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("PORT", defaultPort)
	v.SetDefault("MOCK_DATA_PATH", defaultMockDataPath)
	v.SetDefault("CACHE_TTL_MINUTES", strconv.Itoa(defaultCacheTTLMinutes))
	v.SetDefault("CACHE_CLEANUP_MINUTES", strconv.Itoa(defaultCacheCleanupMinutes))
	v.SetDefault("PROVIDER_TIMEOUT_MS", strconv.Itoa(defaultProviderTimeoutMS))
	v.SetDefault("BEST_VALUE_PRICE_WEIGHT", strconv.FormatFloat(defaultBestValuePriceWeight, 'f', 1, 64))
	v.SetDefault("BEST_VALUE_DURATION_WEIGHT", strconv.FormatFloat(defaultBestValueDurationWeight, 'f', 1, 64))
	v.SetDefault("MOCK_DATA_PROVIDER", encodeDefaultMockFiles())
	v.SetDefault("AIRASIA_DELAY_MS", "100")
	v.SetDefault("AIRASIA_FAILURE_RATE", "10")
	v.SetDefault("BATIK_AIR_DELAY_MS", "200")
	v.SetDefault("GARUDA_INDONESIA_DELAY_MS", "50")
	v.SetDefault("LION_AIR_DELAY_MS", "150")
}

func getInt(v *viper.Viper, key string, defaultValue int) int {
	raw := strings.TrimSpace(v.GetString(key))
	value, err := strconv.Atoi(raw)
	if err != nil {
		return defaultValue
	}
	return value
}

func getNonZeroInt(v *viper.Viper, key string, defaultValue int) int {
	raw := strings.TrimSpace(v.GetString(key))
	value, err := strconv.Atoi(raw)
	if err != nil || value == 0 {
		return defaultValue
	}
	return value
}

func getFloat(v *viper.Viper, key string, defaultValue float64) float64 {
	raw := strings.TrimSpace(v.GetString(key))
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return defaultValue
	}
	return value
}

func getStringSlice(v *viper.Viper, key string, defaultValue []string) []string {
	raw := strings.TrimSpace(v.GetString(key))
	if raw == "" {
		return append([]string(nil), defaultValue...)
	}

	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil || len(values) == 0 {
		return append([]string(nil), defaultValue...)
	}

	return values
}

func encodeDefaultMockFiles() string {
	data, err := json.Marshal(utils.DefaultMockFiles())
	if err != nil {
		return "[]"
	}
	return string(data)
}
