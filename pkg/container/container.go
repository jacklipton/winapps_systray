package container

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/jacklipton/winapps_systray/pkg/config"
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
	cfg           *discovery.Config
	settings      *config.Settings
	mu            sync.Mutex
	transition    State  // non-empty while a Start/Stop is in progress
	containerName string // dynamically discovered
}

func NewController(cfg *discovery.Config, settings *config.Settings) *Controller {
	return &Controller{
		cfg:      cfg,
		settings: settings,
	}
}

// compose builds an exec.Cmd for "engine compose -f <file> <args...>".
func (c *Controller) compose(args ...string) *exec.Cmd {
	full := append([]string{"compose", "-f", c.cfg.ComposeFile}, args...)
	cmd := exec.Command(c.cfg.Engine, full...)
	cmd.Dir = c.cfg.WinAppsDir
	return cmd
}

// GetStatus returns the current container state. During a transition it returns
// the transitional state (Starting/Stopping) until the actual state matches the
// target, preventing the status-poll loop from clobbering the UI.
func (c *Controller) GetStatus() (State, error) {
	actual, err := c.pollState()
	if err != nil {
		return StateError, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.transition == StateStarting {
		if actual == StateRunning {
			c.transition = ""
			return StateRunning, nil
		}
		return StateStarting, nil
	}
	if c.transition == StateStopping {
		if actual == StateStopped {
			c.transition = ""
			return StateStopped, nil
		}
		return StateStopping, nil
	}

	return actual, nil
}

type containerInfo struct {
	Name    string `json:"Name"`
	Service string `json:"Service"`
	State   string `json:"State"`
	Status  string `json:"Status"` // Detailed status like "Up 2 hours"
}

func (c *Controller) pollState() (State, error) {
	args := []string{"ps", "--format", "json"}
	if c.settings.PrimaryService != "" {
		args = append(args, c.settings.PrimaryService)
	}
	cmd := c.compose(args...)
	output, err := cmd.Output()
	if err != nil {
		return StateError, fmt.Errorf("compose ps: %w", err)
	}

	if len(output) == 0 {
		return StateStopped, nil
	}

	var containers []containerInfo
	dec := json.NewDecoder(strings.NewReader(string(output)))
	for dec.More() {
		var info containerInfo
		if err := dec.Decode(&info); err != nil {
			// Some versions output an array, some NDJSON. Try to handle both.
			if strings.HasPrefix(strings.TrimSpace(string(output)), "[") {
				if err := json.Unmarshal(output, &containers); err == nil {
					break
				}
			}
			return StateError, fmt.Errorf("parse compose output: %w", err)
		}
		containers = append(containers, info)
	}

	if len(containers) == 0 {
		return StateStopped, nil
	}

	// For now, we take the first container matching the service (usually only one)
	target := containers[0]
	c.mu.Lock()
	c.containerName = target.Name
	c.mu.Unlock()

	state := strings.ToLower(target.State)
	switch {
	case state == "running":
		return StateRunning, nil
	case state == "starting" || state == "restarting":
		return StateStarting, nil
	case state == "stopping" || state == "removing":
		return StateStopping, nil
	case state == "exited" || state == "created" || state == "dead":
		return StateStopped, nil
	default:
		// Fallback for custom statuses
		if strings.Contains(strings.ToLower(target.Status), "up") {
			return StateRunning, nil
		}
		return StateStopped, nil
	}
}

func (c *Controller) Start() error {
	c.mu.Lock()
	c.transition = StateStarting
	c.mu.Unlock()

	err := c.compose("up", "-d").Run()
	if err != nil {
		c.mu.Lock()
		c.transition = ""
		c.mu.Unlock()
	}
	return err
}

func (c *Controller) Stop() error {
	c.mu.Lock()
	c.transition = StateStopping
	c.mu.Unlock()

	err := c.compose("stop").Run()
	if err != nil {
		c.mu.Lock()
		c.transition = ""
		c.mu.Unlock()
	}
	return err
}

func (c *Controller) Kill() error {
	err := c.compose("kill").Run()
	c.mu.Lock()
	c.transition = ""
	c.mu.Unlock()
	return err
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
	// Timed out — clear the stuck transition so the UI recovers
	c.mu.Lock()
	c.transition = ""
	c.mu.Unlock()
	return errors.New("timeout waiting for container state")
}
