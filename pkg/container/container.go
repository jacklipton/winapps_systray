package container

import (
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/jacklipton/winapps_systray/pkg/discovery"
)

type State string

const (
	StateStopped  State = "Stopped"
	StateRunning  State = "Running"
	StateStarting State = "Starting"
	StateStopping State = "Stopping"
	StateError    State = "Error"
)

type Controller struct {
	cfg *discovery.Config
}

func NewController(cfg *discovery.Config) *Controller {
	return &Controller{cfg: cfg}
}

func (c *Controller) GetStatus() (State, error) {
	cmd := exec.Command(c.cfg.Engine, "compose", "-f", "compose.yaml", "ps", "--format", "json")
	cmd.Dir = c.cfg.WinAppsDir
	output, err := cmd.Output()
	if err != nil {
		// If compose isn't running, it might return non-zero or empty
		return StateStopped, nil
	}

	outStr := string(output)
	if strings.Contains(outStr, "\"State\":\"running\"") {
		return StateRunning, nil
	}
	if strings.Contains(outStr, "\"State\":\"exited\"") || outStr == "" || outStr == "[]\n" {
		return StateStopped, nil
	}
	return StateStopped, nil
}

func (c *Controller) Start() error {
	cmd := exec.Command(c.cfg.Engine, "compose", "up", "-d")
	cmd.Dir = c.cfg.WinAppsDir
	return cmd.Run()
}

func (c *Controller) Stop() error {
	cmd := exec.Command(c.cfg.Engine, "compose", "stop")
	cmd.Dir = c.cfg.WinAppsDir
	return cmd.Run()
}

func (c *Controller) Kill() error {
	cmd := exec.Command(c.cfg.Engine, "compose", "kill")
	cmd.Dir = c.cfg.WinAppsDir
	return cmd.Run()
}

func (c *Controller) WaitUntilState(target State, timeout time.Duration) error {
	start := time.Now()
	for time.Since(start) < timeout {
		status, err := c.GetStatus()
		if err == nil && status == target {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return errors.New("timeout waiting for container state")
}
