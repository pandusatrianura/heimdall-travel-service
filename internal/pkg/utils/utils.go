package utils

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

var defaultMockFiles = []string{
	"airasia_search_response.json",
	"batik_air_search_response.json",
	"garuda_indonesia_search_response.json",
	"lion_air_search_response.json",
}

// ResolveMockFilenames reads the MOCK_DATA_PROVIDER configuration and locates all matching files
func ResolveMockFilenames(prefix string) []string {
	rawEnv := os.Getenv("MOCK_DATA_PROVIDER")

	files := defaultMockFiles
	if rawEnv != "" {
		var envFiles []string
		if err := json.Unmarshal([]byte(rawEnv), &envFiles); err == nil {
			files = envFiles
		}
	}

	prefixLower := strings.ToLower(prefix)
	var matches []string
	for _, f := range files {
		if strings.HasPrefix(strings.ToLower(f), prefixLower) {
			matches = append(matches, f)
		}
	}

	if len(matches) == 0 {
		return []string{prefix + "_search_response.json"}
	}

	return matches
}

// ResolveDelayMS dynamically looks up the <PREFIX>_DELAY_MS env variable.
func ResolveDelayMS(prefix string, defaultVal int) int {
	envKey := strings.ToUpper(prefix) + "_DELAY_MS"
	raw := os.Getenv(envKey)
	if raw == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}
	return val
}

// ResolveFailureRate dynamically looks up the <PREFIX>_FAILURE_RATE env variable.
func ResolveFailureRate(prefix string, defaultVal int) int {
	envKey := strings.ToUpper(prefix) + "_FAILURE_RATE"
	raw := os.Getenv(envKey)
	if raw == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}
	return val
}
