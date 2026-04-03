package utils

import (
	"reflect"
	"testing"
)

func TestResolveMockFilenames(t *testing.T) {
	tests := []struct {
		name      string
		prefix    string
		mockFiles []string
		want      []string
	}{
		{
			name:   "uses default files when list is empty",
			prefix: "airasia",
			want:   []string{"airasia_search_response.json"},
		},
		{
			name:   "matches prefix case insensitively",
			prefix: "AirAsia",
			want:   []string{"airasia_search_response.json"},
		},
		{
			name:      "uses custom file list",
			prefix:    "custom",
			mockFiles: []string{"custom_provider.json", "other_provider.json"},
			want:      []string{"custom_provider.json"},
		},
		{
			name:      "returns multiple matches from provided list",
			prefix:    "test",
			mockFiles: []string{"test_one.json", "test_two.json", "other.json"},
			want:      []string{"test_one.json", "test_two.json"},
		},
		{
			name:      "falls back to default naming when no match exists",
			prefix:    "missing",
			mockFiles: []string{"other.json"},
			want:      []string{"missing_search_response.json"},
		},
		{
			name:   "uses defaults when list is nil",
			prefix: "garuda",
			want:   []string{"garuda_indonesia_search_response.json"},
		},
		{
			name:      "empty list falls back to default naming",
			prefix:    "test",
			mockFiles: []string{},
			want:      []string{"test_search_response.json"},
		},
		{
			name:   "empty prefix returns all matched defaults",
			prefix: "",
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
			got := ResolveMockFilenames(tt.prefix, tt.mockFiles)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ResolveMockFilenames(%q) = %v, want %v", tt.prefix, got, tt.want)
			}
		})
	}
}
