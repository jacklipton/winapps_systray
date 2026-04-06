package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
)

type Settings struct {
	Notifications       bool   `json:"notifications"`
	PollIntervalSeconds int    `json:"poll_interval_seconds"`
	StartTimeoutSeconds int    `json:"start_timeout_seconds"`
	StopTimeoutSeconds  int    `json:"stop_timeout_seconds"`
	WinAppsDir          string `json:"winapps_dir"`
	PrimaryService      string `json:"primary_service"`
	VNCPort             int    `json:"vnc_port"`
}

func defaults() Settings {
	return Settings{
		Notifications:       true,
		PollIntervalSeconds: 5,
		StartTimeoutSeconds: 60,
		StopTimeoutSeconds:  120,
		VNCPort:             8006,
	}
}

// Save marshals settings to JSON and writes it to path.
func (s *Settings) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// Load reads settings from path. If the file doesn't exist, writes defaults
// and returns them. Fields missing from the JSON get default values.
func Load(path string) (*Settings, error) {
	cfg := defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Write defaults on first run
			writeDefaults(path, &cfg)
			return &cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		slog.Warn("invalid settings file, using defaults", "path", path, "error", err)
		d := defaults()
		return &d, nil
	}

	// Apply defaults for zero-value fields
	d := defaults()
	if cfg.PollIntervalSeconds == 0 {
		cfg.PollIntervalSeconds = d.PollIntervalSeconds
	}
	if cfg.StartTimeoutSeconds == 0 {
		cfg.StartTimeoutSeconds = d.StartTimeoutSeconds
	}
	if cfg.StopTimeoutSeconds == 0 {
		cfg.StopTimeoutSeconds = d.StopTimeoutSeconds
	}
	if cfg.VNCPort == 0 {
		cfg.VNCPort = d.VNCPort
	}

	cfg.validate()

	return &cfg, nil
}

// validate clamps settings to safe ranges.
func (s *Settings) validate() {
	s.PollIntervalSeconds = clamp(s.PollIntervalSeconds, 1, 300)
	s.StartTimeoutSeconds = clamp(s.StartTimeoutSeconds, 1, 600)
	s.StopTimeoutSeconds = clamp(s.StopTimeoutSeconds, 1, 600)
	s.VNCPort = clamp(s.VNCPort, 1, 65535)
}

func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// writeDefaults creates the settings file with default values.
// Errors are logged but not fatal — config is non-critical.
func writeDefaults(path string, cfg *Settings) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		slog.Warn("cannot create config dir", "dir", dir, "error", err)
		return
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(path, data, 0600); err != nil {
		slog.Warn("cannot write default settings", "path", path, "error", err)
	}
}
