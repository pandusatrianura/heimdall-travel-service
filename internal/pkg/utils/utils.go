package utils

import (
	"strings"
)

var defaultMockFiles = []string{
	"airasia_search_response.json",
	"batik_air_search_response.json",
	"garuda_indonesia_search_response.json",
	"lion_air_search_response.json",
}

func DefaultMockFiles() []string {
	return append([]string(nil), defaultMockFiles...)
}

// ResolveMockFilenames locates matching files from an injected mock file list.
func ResolveMockFilenames(prefix string, files []string) []string {
	if len(files) == 0 {
		files = DefaultMockFiles()
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
