package discovery

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Config struct {
	WinAppsDir    string
	ComposeFile   string // e.g. "compose.yaml", "docker-compose.yml"
	ContainerName string
	Engine        string // "docker" or "podman"
}

func GetConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dir, composeFile, err := findWinAppsDir(home)
	if err != nil {
		return nil, err
	}

	engine, containerName := detectEngineAndName(dir)

	return &Config{
		WinAppsDir:    dir,
		ComposeFile:   composeFile,
		ContainerName: containerName,
		Engine:        engine,
	}, nil
}

// validateDir cleans a user-supplied path and rejects non-absolute or
// obviously invalid values to prevent path traversal misuse.
func validateDir(path string) (string, error) {
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		return "", fmt.Errorf("path must be absolute: %q", clean)
	}
	return clean, nil
}

func findWinAppsDir(home string) (string, string, error) {
	// 1. Explicit override via environment variable
	if env := os.Getenv("WINAPPS_DIR"); env != "" {
		dir, err := validateDir(env)
		if err != nil {
			return "", "", fmt.Errorf("WINAPPS_DIR: %w", err)
		}
		if f := findComposeFile(dir); f != "" {
			return dir, f, nil
		}
		return "", "", errors.New("WINAPPS_DIR points to a directory with no compose file")
	}

	// 2. XDG config pointer: ~/.config/winapps-systray/config with a path
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = filepath.Join(home, ".config")
	}
	configFile := filepath.Join(configDir, "winapps-systray", "config")
	if data, err := os.ReadFile(configFile); err == nil {
		raw := strings.TrimSpace(string(data))
		if raw != "" {
			dir, err := validateDir(raw)
			if err != nil {
				return "", "", fmt.Errorf("config file %s: %w", configFile, err)
			}
			if f := findComposeFile(dir); f != "" {
				return dir, f, nil
			}
		}
	}

	// 3. Common locations
	candidates := []string{
		filepath.Join(home, "winapps"),
		filepath.Join(home, ".winapps"),
		filepath.Join(home, "Documents", "winapps"),
	}

	for _, path := range candidates {
		if f := findComposeFile(path); f != "" {
			return path, f, nil
		}
	}

	return "", "", errors.New("winapps directory not found; set WINAPPS_DIR or create ~/.config/winapps-systray/config")
}

// findComposeFile returns the compose filename found in dir, or "" if none.
func findComposeFile(dir string) string {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return ""
	}
	for _, name := range []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return name
		}
	}
	return ""
}

func detectEngineAndName(dir string) (string, string) {
	containerName := "WinApps" // Default

	// Try to see which engine has the container already
	engines := []string{"docker", "podman"}
	for _, e := range engines {
		if _, err := exec.LookPath(e); err == nil {
			cmd := exec.Command(e, "compose", "ps", "--format", "json")
			cmd.Dir = dir
			output, err := cmd.Output()
			if err == nil && len(output) > 0 && !strings.Contains(string(output), "[]") {
				return e, containerName
			}
		}
	}

	// Fallback to whichever is installed
	if _, err := exec.LookPath("docker"); err == nil {
		return "docker", containerName
	}
	if _, err := exec.LookPath("podman"); err == nil {
		return "podman", containerName
	}

	return "docker", containerName
}
