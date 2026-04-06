package discovery

import (
	"bufio"
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
	RDPUser       string // from winapps.conf, for display
}

func GetConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// Read winapps.conf for engine hint and RDP user
	waConf := readWinappsConf(home)

	dir, composeFile, err := findWinAppsDir(home)
	if err != nil {
		return nil, err
	}

	engine, containerName := detectEngineAndName(dir, waConf)

	cfg := &Config{
		WinAppsDir:    dir,
		ComposeFile:   composeFile,
		ContainerName: containerName,
		Engine:        engine,
	}
	if waConf != nil {
		cfg.RDPUser = waConf.rdpUser
	}

	return cfg, nil
}

// winappsConf holds values parsed from ~/.config/winapps/winapps.conf.
type winappsConf struct {
	flavor  string // docker, podman, libvirt
	rdpUser string
}

func readWinappsConf(home string) *winappsConf {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = filepath.Join(home, ".config")
	}
	path := filepath.Join(configDir, "winapps", "winapps.conf")

	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	conf := &winappsConf{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		val = strings.Trim(val, "\"'")
		switch key {
		case "WAFLAVOR":
			conf.flavor = val
		case "RDP_USER":
			conf.rdpUser = val
		}
	}
	return conf
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

	// 3. Common locations (including ~/.config/winapps per upstream docs)
	candidates := []string{
		filepath.Join(configDir, "winapps"),
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

func detectEngineAndName(dir string, waConf *winappsConf) (string, string) {
	containerName := "WinApps" // Default

	// If winapps.conf specifies a flavor, prefer it
	if waConf != nil && (waConf.flavor == "docker" || waConf.flavor == "podman") {
		if _, err := exec.LookPath(waConf.flavor); err == nil {
			return waConf.flavor, containerName
		}
	}

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

// ListServices returns a list of service names defined in the compose file
// at the given directory using the specified container engine.
func ListServices(dir, engine string) ([]string, error) {
	cmd := exec.Command(engine, "compose", "config", "--services")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}

	raw := strings.TrimSpace(string(output))
	if raw == "" {
		return []string{}, nil
	}

	return strings.Split(raw, "\n"), nil
}
