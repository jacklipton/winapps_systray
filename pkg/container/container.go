package container

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
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
	cfg        *discovery.Config
	mu         sync.Mutex
	transition State // non-empty while a Start/Stop is in progress
}

func NewController(cfg *discovery.Config) *Controller {
	return &Controller{cfg: cfg}
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
	State string `json:"State"`
}

func (c *Controller) pollState() (State, error) {
	cmd := c.compose("ps", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return StateError, fmt.Errorf("compose ps: %w", err)
	}

	var containers []containerInfo
	if err := json.Unmarshal(output, &containers); err != nil {
		return StateError, fmt.Errorf("parse compose output: %w", err)
	}

	for _, info := range containers {
		if strings.EqualFold(info.State, "running") {
			return StateRunning, nil
		}
	}
	return StateStopped, nil
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
