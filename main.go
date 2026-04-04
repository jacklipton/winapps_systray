package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gotk3/gotk3/glib"
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

	// Single-instance lock
	lockFile, err := acquireLock()
	if err != nil {
		ui.ShowError(nil, "WinApps Systray is already running.")
		return
	}
	defer lockFile.Close()

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
	isInitialSetup := false
	if err != nil {
		log.Printf("Discovery failed: %v", err)
		if ui.ShowSetupRequired(nil, "WinApps directory not found or engine missing.\n\nWould you like to open Settings to configure it manually?") {
			isInitialSetup = true
			discoveryCfg = &discovery.Config{Engine: "docker"} // Default for setup UI
		} else {
			return
		}
	} else if cfg.PrimaryService == "" {
		// Found directory but no primary service selected
		isInitialSetup = true
	}

	ctrl := container.NewController(discoveryCfg, cfg)

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
	tm := tray.NewTrayManager(ctrl, cfg, settingsPath, iconMgr)
	tm.OnDashboard = func() { dashboard.Show() }
	tm.Dashboard = dashboard
	tm.Setup()

	if isInitialSetup {
		sw := ui.NewSettingsWindow(cfg, settingsPath, ctrl.Engine())
		sw.Show()
	}

	// Handle signals for clean shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		glib.IdleAdd(func() { gtk.MainQuit() })
	}()

	// Run GTK main loop (blocks until Quit)
	gtk.Main()
}

// acquireLock tries to obtain an exclusive file lock to prevent multiple instances.
func acquireLock() (*os.File, error) {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = os.TempDir()
	}
	lockPath := filepath.Join(runtimeDir, "winapps-systray.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("another instance is already running")
	}
	return f, nil
}
