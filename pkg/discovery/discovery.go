package discovery

import (
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	WinAppsDir    string
	ContainerName string
	Engine        string // "docker" or "podman"
}

func GetConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir, err := findWinAppsDir(home)
	if err != nil {
		return nil, err
	}
    // Simple detection for now
    engine := "docker"
    if _, err := os.Stat("/usr/bin/podman"); err == nil {
        engine = "podman"
    }

	return &Config{
		WinAppsDir:    dir,
		ContainerName: "WinApps", // Default from compose.yaml in spec
		Engine:        engine,
	}, nil
}

func findWinAppsDir(home string) (string, error) {
	path := filepath.Join(home, "winapps")
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		if _, err := os.Stat(filepath.Join(path, "compose.yaml")); err == nil {
			return path, nil
		}
	}
	return "", errors.New("winapps directory not found with compose.yaml")
}
