package utils

import (
	"reflect"
	"strings"
	"testing"
)

func TestResolveMockFilenames(t *testing.T) {
	tests := []struct {
		name        string
		prefix      string
		mockDataEnv string
		want        []string
	}{
		{
			name:        "uses default files when env is empty",
			prefix:      "airasia",
			mockDataEnv: "",
			want:        []string{"airasia_search_response.json"},
		},
		{
			name:        "matches prefix case insensitively",
			prefix:      "AirAsia",
			mockDataEnv: "",
			want:        []string{"airasia_search_response.json"},
		},
		{
			name:        "uses custom env file list",
			prefix:      "custom",
			mockDataEnv: `["custom_provider.json","other_provider.json"]`,
			want:        []string{"custom_provider.json"},
		},
		{
			name:        "returns multiple matches from env list",
			prefix:      "test",
			mockDataEnv: `["test_one.json","test_two.json","other.json"]`,
			want:        []string{"test_one.json", "test_two.json"},
		},
		{
			name:        "falls back to default naming when no match exists",
			prefix:      "missing",
			mockDataEnv: `["other.json"]`,
			want:        []string{"missing_search_response.json"},
		},
		{
			name:        "invalid env json falls back to defaults",
			prefix:      "garuda",
			mockDataEnv: `{invalid json}`,
			want:        []string{"garuda_indonesia_search_response.json"},
		},
		{
			name:        "empty json array falls back to default naming",
			prefix:      "test",
			mockDataEnv: `[]`,
			want:        []string{"test_search_response.json"},
		},
		{
			name:        "empty prefix returns all matched defaults",
			prefix:      "",
			mockDataEnv: "",
			want: []string{
				"airasia_search_response.json",
				"batik_air_search_response.json",
				"garuda_indonesia_search_response.json",
				"lion_air_search_response.json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MOCK_DATA_PROVIDER", tt.mockDataEnv)

			got := ResolveMockFilenames(tt.prefix)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ResolveMockFilenames(%q) = %v, want %v", tt.prefix, got, tt.want)
			}
		})
	}
}

func TestResolveDelayMS(t *testing.T) {
	tests := []struct {
		name       string
		prefix     string
		defaultVal int
		envValue   string
		want       int
	}{
		{
			name:       "returns configured delay",
			prefix:     "airasia",
			defaultVal: 100,
			envValue:   "250",
			want:       250,
		},
		{
			name:       "uses uppercase env key from mixed case prefix",
			prefix:     "GaRuDa",
			defaultVal: 100,
			envValue:   "75",
			want:       75,
		},
		{
			name:       "returns default when env is empty",
			prefix:     "lion_air",
			defaultVal: 150,
			envValue:   "",
			want:       150,
		},
		{
			name:       "returns default when env is invalid",
			prefix:     "batik_air",
			defaultVal: 200,
			envValue:   "abc",
			want:       200,
		},
		{
			name:       "accepts zero delay",
			prefix:     "instant",
			defaultVal: 300,
			envValue:   "0",
			want:       0,
		},
		{
			name:       "preserves negative values when parse succeeds",
			prefix:     "test",
			defaultVal: 300,
			envValue:   "-50",
			want:       -50,
		},
		{
			name:       "returns default on integer overflow",
			prefix:     "overflow",
			defaultVal: 300,
			envValue:   "9999999999999999999",
			want:       300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envKey := tt.prefix
			t.Setenv(strings.ToUpper(envKey)+"_DELAY_MS", tt.envValue)

			got := ResolveDelayMS(tt.prefix, tt.defaultVal)
			if got != tt.want {
				t.Fatalf("ResolveDelayMS(%q, %d) = %d, want %d", tt.prefix, tt.defaultVal, got, tt.want)
			}
		})
	}
}

func TestResolveFailureRate(t *testing.T) {
	tests := []struct {
		name       string
		prefix     string
		defaultVal int
		envValue   string
		want       int
	}{
		{
			name:       "returns configured failure rate",
			prefix:     "airasia",
			defaultVal: 10,
			envValue:   "35",
			want:       35,
		},
		{
			name:       "returns default when env is empty",
			prefix:     "garuda_indonesia",
			defaultVal: 5,
			envValue:   "",
			want:       5,
		},
		{
			name:       "returns default when env is invalid",
			prefix:     "lion_air",
			defaultVal: 15,
			envValue:   "not-a-number",
			want:       15,
		},
		{
			name:       "accepts zero failure rate",
			prefix:     "batik_air",
			defaultVal: 20,
			envValue:   "0",
			want:       0,
		},
		{
			name:       "accepts values above 100 when parse succeeds",
			prefix:     "test",
			defaultVal: 10,
			envValue:   "150",
			want:       150,
		},
		{
			name:       "preserves negative values when parse succeeds",
			prefix:     "negative",
			defaultVal: 10,
			envValue:   "-5",
			want:       -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(strings.ToUpper(tt.prefix)+"_FAILURE_RATE", tt.envValue)

			got := ResolveFailureRate(tt.prefix, tt.defaultVal)
			if got != tt.want {
				t.Fatalf("ResolveFailureRate(%q, %d) = %d, want %d", tt.prefix, tt.defaultVal, got, tt.want)
			}
		})
	}
}
