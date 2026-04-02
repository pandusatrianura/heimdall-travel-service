package timeutil

import (
	"testing"
)

func TestParseTime(t *testing.T) {
	tests := []struct {
		name     string
		timeStr  string
		locName  string
		wantYear int
		wantHour int
		wantErr  bool
	}{
		{
			name:     "RFC3339 with offset",
			timeStr:  "2025-12-15T04:45:00+07:00",
			locName:  "",
			wantYear: 2025,
			wantHour: 4,
			wantErr:  false,
		},
		{
			name:     "Standard with offset no colon",
			timeStr:  "2025-12-15T07:15:00+0700",
			locName:  "",
			wantYear: 2025,
			wantHour: 7,
			wantErr:  false,
		},
		{
			name:     "No offset but location provided (Lion Air style)",
			timeStr:  "2025-12-15T05:30:00",
			locName:  "Asia/Jakarta",
			wantYear: 2025,
			wantHour: 5,
			wantErr:  false,
		},
		{
			name:    "Invalid format",
			timeStr: "15-12-2025 04:45:00",
			locName: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTime(tt.timeStr, tt.locName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Year() != tt.wantYear {
					t.Errorf("ParseTime() Year = %v, want %v", got.Year(), tt.wantYear)
				}
				if got.Hour() != tt.wantHour {
					t.Errorf("ParseTime() Hour = %v, want %v", got.Hour(), tt.wantHour)
				}
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		minutes int
		want    string
	}{
		{90, "1h 30m"},
		{120, "2h 0m"},
		{45, "45m"},
		{260, "4h 20m"},
	}

	for _, tt := range tests {
		got := FormatDuration(tt.minutes)
		if got != tt.want {
			t.Errorf("FormatDuration(%d) = %q, want %q", tt.minutes, got, tt.want)
		}
	}
}
