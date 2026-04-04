package container

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
)

// Stats holds live container resource usage.
type Stats struct {
	Name       string
	CPUPercent float64
	MemUsage   string // e.g. "4.1GiB"
	MemPercent float64
	IPAddress  string
}

func parseStats(data []byte) (*Stats, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	stats := &Stats{
		Name: getString(raw["Name"]),
	}

	// CPUPercent (Docker v24+) or CPUPerc (older Docker/Podman)
	if val, ok := raw["CPUPercent"]; ok {
		stats.CPUPercent = parseFloat(val)
	} else if val, ok := raw["CPUPerc"]; ok {
		stats.CPUPercent = parseFloat(val)
	}

	// MemPerc (older Docker/Podman) or MemPercent (Docker v24+)
	if val, ok := raw["MemPercent"]; ok {
		stats.MemPercent = parseFloat(val)
	} else if val, ok := raw["MemPerc"]; ok {
		stats.MemPercent = parseFloat(val)
	}

	// MemUsage is "4.1GiB / 16GiB" or similar
	memUsage := getString(raw["MemUsage"])
	if parts := strings.SplitN(memUsage, " / ", 2); len(parts) == 2 {
		memUsage = parts[0]
	}
	stats.MemUsage = memUsage

	return stats, nil
}

func getString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func parseFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSuffix(val, "%"), 64)
		return f
	default:
		return 0
	}
}

func parseIPOutput(output string) string {
	return strings.TrimSpace(output)
}

// GetStats returns live resource usage for the container.
// Returns nil if the container is not running or stats are unavailable.
func (c *Controller) GetStats() *Stats {
	c.mu.Lock()
	containerName := c.containerName
	c.mu.Unlock()

	if containerName == "" {
		return nil
	}

	// docker/podman stats --no-stream --format json <container>
	cmd := exec.Command(c.cfg.Engine, "stats", "--no-stream", "--format", "json", containerName)
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return nil
	}

	stats, err := parseStats(output)
	if err != nil {
		return nil
	}

	// Get IP address via inspect
	ipCmd := exec.Command(c.cfg.Engine, "inspect", "--format",
		"{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}",
		containerName)
	if ipOut, err := ipCmd.Output(); err == nil {
		stats.IPAddress = parseIPOutput(string(ipOut))
	}

	return stats
}

// ComposeFile returns the compose file name from config.
func (c *Controller) ComposeFile() string {
	return c.cfg.ComposeFile
}

// Engine returns the container engine name from config.
func (c *Controller) Engine() string {
	return c.cfg.Engine
}

// ContainerName returns the container name from config.
func (c *Controller) ContainerName() string {
	return c.cfg.ContainerName
}

// PrimaryService returns the primary service name from settings.
func (c *Controller) PrimaryService() string {
	if c.settings.PrimaryService != "" {
		return c.settings.PrimaryService
	}
	return "WinApps (auto)"
}
