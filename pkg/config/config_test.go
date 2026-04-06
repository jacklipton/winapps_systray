package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Notifications {
		t.Error("expected notifications enabled by default")
	}
	if cfg.PollIntervalSeconds != 5 {
		t.Errorf("expected poll interval 5, got %d", cfg.PollIntervalSeconds)
	}
	if cfg.StartTimeoutSeconds != 60 {
		t.Errorf("expected start timeout 60, got %d", cfg.StartTimeoutSeconds)
	}
	if cfg.StopTimeoutSeconds != 120 {
		t.Errorf("expected stop timeout 120, got %d", cfg.StopTimeoutSeconds)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(`{"notifications":false,"poll_interval_seconds":10}`), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Notifications {
		t.Error("expected notifications disabled")
	}
	if cfg.PollIntervalSeconds != 10 {
		t.Errorf("expected poll interval 10, got %d", cfg.PollIntervalSeconds)
	}
	// Unset fields should get defaults
	if cfg.StartTimeoutSeconds != 60 {
		t.Errorf("expected start timeout default 60, got %d", cfg.StartTimeoutSeconds)
	}
	if cfg.StopTimeoutSeconds != 120 {
		t.Errorf("expected stop timeout default 120, got %d", cfg.StopTimeoutSeconds)
	}
}

func TestLoadClampsNegativeValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(`{"poll_interval_seconds":-5,"start_timeout_seconds":-1,"stop_timeout_seconds":0}`), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.PollIntervalSeconds < 1 {
		t.Errorf("expected poll interval >= 1, got %d", cfg.PollIntervalSeconds)
	}
	if cfg.StartTimeoutSeconds < 1 {
		t.Errorf("expected start timeout >= 1, got %d", cfg.StartTimeoutSeconds)
	}
	if cfg.StopTimeoutSeconds < 1 {
		t.Errorf("expected stop timeout >= 1, got %d", cfg.StopTimeoutSeconds)
	}
}

func TestLoadClampsExcessiveValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(`{"poll_interval_seconds":3600,"start_timeout_seconds":99999}`), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.PollIntervalSeconds > 300 {
		t.Errorf("expected poll interval <= 300, got %d", cfg.PollIntervalSeconds)
	}
	if cfg.StartTimeoutSeconds > 600 {
		t.Errorf("expected start timeout <= 600, got %d", cfg.StartTimeoutSeconds)
	}
}

func TestSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	cfg := &Settings{
		Notifications:       false,
		PollIntervalSeconds: 15,
		WinAppsDir:          "/home/user/winapps",
		PrimaryService:      "windows",
	}

	if err := cfg.Save(path); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Load it back and verify
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if loaded.Notifications {
		t.Error("expected notifications to be false")
	}
	if loaded.PollIntervalSeconds != 15 {
		t.Errorf("expected poll interval 15, got %d", loaded.PollIntervalSeconds)
	}
	if loaded.WinAppsDir != "/home/user/winapps" {
		t.Errorf("expected winapps dir /home/user/winapps, got %s", loaded.WinAppsDir)
	}
	if loaded.PrimaryService != "windows" {
		t.Errorf("expected primary service windows, got %s", loaded.PrimaryService)
	}
}
