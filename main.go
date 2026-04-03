package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/gotk3/gotk3/gtk"
	"github.com/jacklipton/winapps_systray/pkg/config"
	"github.com/jacklipton/winapps_systray/pkg/container"
	"github.com/jacklipton/winapps_systray/pkg/discovery"
	"github.com/jacklipton/winapps_systray/pkg/icons"
	"github.com/jacklipton/winapps_systray/pkg/tray"
	"github.com/jacklipton/winapps_systray/pkg/ui"
)

func main() {
	gtk.Init(nil)

	// Load config
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	settingsPath := filepath.Join(configDir, "winapps-systray", "settings.json")
	cfg, err := config.Load(settingsPath)
	if err != nil {
		log.Printf("warning: failed to load settings: %v", err)
		defaults := config.Settings{
			Notifications:       true,
			PollIntervalSeconds: 5,
			StartTimeoutSeconds: 60,
			StopTimeoutSeconds:  120,
		}
		cfg = &defaults
	}

	// Discover winapps
	discoveryCfg, err := discovery.GetConfig()
	if err != nil {
		log.Fatalf("Discovery failed: %v", err)
	}

	ctrl := container.NewController(discoveryCfg)

	// Set up icons in temp directory
	iconDir, err := os.MkdirTemp("", "winapps-icons-")
	if err != nil {
		log.Fatalf("Failed to create icon temp dir: %v", err)
	}
	defer os.RemoveAll(iconDir)

	iconMgr, err := icons.Setup(iconDir)
	if err != nil {
		log.Fatalf("Failed to set up icons: %v", err)
	}

	// Create dashboard (lazy — window created on first Show)
	dashboardIcon := filepath.Join(iconDir, "winapps-running.svg")
	dashboard := ui.NewDashboard(ctrl, dashboardIcon)

	// Set up tray
	tm := tray.NewTrayManager(ctrl, cfg, iconMgr)
	tm.OnDashboard = func() { dashboard.Show() }
	tm.Setup()

	// Run GTK main loop (blocks until Quit)
	gtk.Main()
}
