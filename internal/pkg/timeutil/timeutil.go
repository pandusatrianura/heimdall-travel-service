package timeutil

import (
	"fmt"
	"time"
)

// ParseTime parses a datetime string into a time.Time object.
// It tries multiple common formats and supports timezone offsets and implicit loc names.
func ParseTime(timeStr string, locName string) (time.Time, error) {
	// Try parsing standard RFC3339 first (like 2025-12-15T04:45:00+07:00)
	t, err := time.Parse(time.RFC3339, timeStr)
	if err == nil {
		return t, nil
	}

	// Try parsing format commonly used without colon in offset (2025-12-15T07:15:00+0700)
	t, err = time.Parse("2006-01-02T15:04:05-0700", timeStr)
	if err == nil {
		return t, nil
	}

	// Try format without timezone attached (e.g. 2025-12-15T05:30:00)
	// We need to apply the location if locName is provided.
	if locName != "" {
		loc, err := time.LoadLocation(locName)
		if err != nil {
			// Fallback if loc not found: assume UTC and let the user handle it
			loc = time.UTC
		}
		t, err = time.ParseInLocation("2006-01-02T15:04:05", timeStr, loc)
		if err == nil {
			return t, nil
		}
	} else {
		// Just parse without zone as UTC
		t, err = time.Parse("2006-01-02T15:04:05", timeStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %s", timeStr)
}

// FormatDuration formats total minutes into '4h 20m' format
func FormatDuration(minutes int) string {
	hours := minutes / 60
	mins := minutes % 60
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}
