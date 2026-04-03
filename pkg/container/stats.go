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

type rawStats struct {
	Name     string `json:"Name"`
	CPUPerc  string `json:"CPUPerc"`
	MemUsage string `json:"MemUsage"`
	MemPerc  string `json:"MemPerc"`
}

func parseStats(data []byte) (*Stats, error) {
	var raw rawStats
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	cpu, _ := strconv.ParseFloat(strings.TrimSuffix(raw.CPUPerc, "%"), 64)
	memPct, _ := strconv.ParseFloat(strings.TrimSuffix(raw.MemPerc, "%"), 64)

	// MemUsage is "4.1GiB / 16GiB" — take the first part
	memUsage := raw.MemUsage
	if parts := strings.SplitN(raw.MemUsage, " / ", 2); len(parts) == 2 {
		memUsage = parts[0]
	}

	return &Stats{
		Name:       raw.Name,
		CPUPercent: cpu,
		MemUsage:   memUsage,
		MemPercent: memPct,
	}, nil
}

func parseIPOutput(output string) string {
	return strings.TrimSpace(output)
}

// GetStats returns live resource usage for the container.
// Returns nil if the container is not running or stats are unavailable.
func (c *Controller) GetStats() *Stats {
	// docker/podman stats --no-stream --format json <container>
	cmd := exec.Command(c.cfg.Engine, "stats", "--no-stream", "--format", "json", c.cfg.ContainerName)
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
		c.cfg.ContainerName)
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
