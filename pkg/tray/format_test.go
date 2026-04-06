package tray

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"zero", 0, "0s"},
		{"seconds", 45 * time.Second, "45s"},
		{"one minute", 60 * time.Second, "1m"},
		{"minutes", 5*time.Minute + 30*time.Second, "5m"},
		{"one hour", 60 * time.Minute, "1h 0m"},
		{"hours and minutes", 2*time.Hour + 15*time.Minute, "2h 15m"},
		{"large", 48*time.Hour + 30*time.Minute, "48h 30m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}
