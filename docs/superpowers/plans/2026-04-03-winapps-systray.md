# WinApps Systray Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a lightweight Go-based system tray app to start, stop, and monitor the WinApps Docker/Podman container with "Wait & Verify" and "Force Kill" support.

**Architecture:**
- **Discovery Engine:** Locate `~/winapps` and its `compose.yaml`.
- **Container Controller:** Wraps `docker compose` or `podman compose` to manage container lifecycle.
- **Tray Manager:** Native tray icon with dynamic menus and state-based icons.
- **Event Loop:** Background goroutine for status polling during transitions.

**Tech Stack:**
- **Go 1.20+**
- **getlantern/systray** (Native tray integration)
- **Container engines:** Docker or Podman (wrapped CLI)

---

### Task 1: Environment Setup & Dependencies

**Files:**
- Create: `go.mod` (already initialized)
- Create: `main.go`
- Create: `assets/assets.go`

- [ ] **Step 1: Install dependencies**

Run: `export PATH=$PATH:/usr/local/go/bin && go get github.com/getlantern/systray`
Expected: `systray` added to `go.mod`.

- [ ] **Step 2: Create a minimal `main.go` entry point**

```go
package main

import (
	"github.com/getlantern/systray"
	"log"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTitle("WinApps")
	systray.SetTooltip("WinApps Container Controller")
	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	go func() {
		for {
			select {
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	// Clean up here if needed
}
```

- [ ] **Step 3: Verify the minimal app runs**

Run: `export PATH=$PATH:/usr/local/go/bin && go build -o winapps_systray main.go`
Expected: `winapps_systray` binary created.
(Note: Running it in a headless environment might fail, so we verify build only for now).

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum main.go
git commit -m "chore: initial setup with systray skeleton"
```

---

### Task 2: Discovery Engine

**Files:**
- Create: `pkg/discovery/discovery.go`
- Test: `pkg/discovery/discovery_test.go`

- [ ] **Step 1: Write failing test for finding winapps directory**

```go
package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindWinAppsDir(t *testing.T) {
	tempHome := t.TempDir()
	winappsDir := filepath.Join(tempHome, "winapps")
	os.Mkdir(winappsDir, 0755)
	os.WriteFile(filepath.Join(winappsDir, "compose.yaml"), []byte("name: \"winapps\""), 0644)

	path, err := findWinAppsDir(tempHome)
	if err != nil {
		t.Fatalf("Expected to find winapps dir, got error: %v", err)
	}
	if path != winappsDir {
		t.Errorf("Expected %s, got %s", winappsDir, path)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `export PATH=$PATH:/usr/local/go/bin && go test ./pkg/discovery/...`
Expected: FAIL (undefined `findWinAppsDir`)

- [ ] **Step 3: Implement `findWinAppsDir` and `Config` struct**

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `export PATH=$PATH:/usr/local/go/bin && go test ./pkg/discovery/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/discovery
git commit -m "feat: implement discovery engine to find winapps directory"
```

---

### Task 3: Container Controller (Status & Status Mapping)

**Files:**
- Create: `pkg/container/container.go`
- Test: `pkg/container/container_test.go`

- [ ] **Step 1: Define `State` enum and `Controller` struct**

```go
package container

import (
	"os/exec"
	"strings"
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
```

- [ ] **Step 2: Run status check on real environment (if possible)**

Run: `export PATH=$PATH:/usr/local/go/bin && go build ./pkg/container/...`
Expected: Build success.

- [ ] **Step 3: Commit**

```bash
git add pkg/container
git commit -m "feat: implement container status detection"
```

---

### Task 4: Container Controller (Commands & Wait/Verify)

**Files:**
- Modify: `pkg/container/container.go`

- [ ] **Step 1: Add Start, Stop, and Kill methods**

```go
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
```

- [ ] **Step 2: Add WaitUntilState method for polling**

```go
import "time"

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
```

- [ ] **Step 3: Commit**

```bash
git add pkg/container/container.go
git commit -m "feat: add container lifecycle commands and polling"
```

---

### Task 5: Assets & Icons (Placeholder Generation)

**Files:**
- Create: `assets/generate_icons.go`
- Create: `assets/icons.go` (embedded)

- [ ] **Step 1: Create a simple script to generate monochrome icons if needed, or define base64 versions**

We will use small 16x16 PNGs. For this implementation, we will embed base64 strings of minimal colored squares as icons if real icons are missing.

```go
package assets

import _ "embed"

// Minimal 16x16 PNG icons as base64 or raw bytes
// For simplicity in the plan, we'll use placeholder bytes.

var (
	// Blue icon for Running
	IconRunning []byte = []byte{...} 
	// Gray icon for Stopped
	IconStopped []byte = []byte{...}
	// Yellow icon for Transition
	IconTransition []byte = []byte{...}
)
```

(Note: I will provide actual small PNG byte slices in the implementation phase).

- [ ] **Step 2: Commit placeholders**

```bash
git add assets
git commit -m "chore: add placeholder asset definitions"
```

---

### Task 6: Tray UI Implementation

**Files:**
- Modify: `main.go`
- Create: `pkg/tray/tray.go`

- [ ] **Step 1: Implement `TrayManager` in `pkg/tray/tray.go`**

```go
package tray

import (
	"fmt"
	"time"
	"github.com/getlantern/systray"
	"github.com/jacklipton/winapps_systray/pkg/container"
	"github.com/jacklipton/winapps_systray/assets"
)

type TrayManager struct {
	ctrl *container.Controller
	mStatus *systray.MenuItem
	mToggle *systray.MenuItem
	mKill   *systray.MenuItem
}

func NewTrayManager(ctrl *container.Controller) *TrayManager {
	return &TrayManager{ctrl: ctrl}
}

func (t *TrayManager) Setup() {
	t.mStatus = systray.AddMenuItem("Status: Unknown", "Current container status")
	t.mStatus.Disable()
	systray.AddSeparator()
	t.mToggle = systray.AddMenuItem("Start Windows", "Toggle container state")
	t.mKill = systray.AddMenuItem("Force Kill", "Forcefully stop the container")
	t.mKill.Hide()
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit")

	go t.statusLoop()
	go t.clickLoop(mQuit)
}

func (t *TrayManager) updateUI(state container.State) {
	t.mStatus.SetTitle(fmt.Sprintf("Status: %s", state))
	t.mKill.Hide()

	switch state {
	case container.StateRunning:
		systray.SetIcon(assets.IconRunning)
		t.mToggle.SetTitle("Stop Windows")
		t.mToggle.Enable()
	case container.StateStopped:
		systray.SetIcon(assets.IconStopped)
		t.mToggle.SetTitle("Start Windows")
		t.mToggle.Enable()
	case container.StateStarting, container.StateStopping:
		systray.SetIcon(assets.IconTransition)
		t.mToggle.Disable()
		if state == container.StateStopping {
			t.mKill.Show()
		}
	}
}

func (t *TrayManager) statusLoop() {
	for {
		status, _ := t.ctrl.GetStatus()
		t.updateUI(status)
		time.Sleep(5 * time.Second)
	}
}

func (t *TrayManager) clickLoop(mQuit *systray.MenuItem) {
	for {
		select {
		case <-t.mToggle.ClickedCh:
			status, _ := t.ctrl.GetStatus()
			if status == container.StateRunning {
				t.updateUI(container.StateStopping)
				go func() {
					t.ctrl.Stop()
					t.ctrl.WaitUntilState(container.StateStopped, 120*time.Second)
				}()
			} else if status == container.StateStopped {
				t.updateUI(container.StateStarting)
				go func() {
					t.ctrl.Start()
					t.ctrl.WaitUntilState(container.StateRunning, 60*time.Second)
				}()
			}
		case <-t.mKill.ClickedCh:
			t.ctrl.Kill()
		case <-mQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}
```

- [ ] **Step 2: Update `main.go` to use `TrayManager`**

```go
package main

import (
	"log"
	"github.com/getlantern/systray"
	"github.com/jacklipton/winapps_systray/pkg/discovery"
	"github.com/jacklipton/winapps_systray/pkg/container"
	"github.com/jacklipton/winapps_systray/pkg/tray"
)

func main() {
	systray.Run(onReady, func() {})
}

func onReady() {
	cfg, err := discovery.GetConfig()
	if err != nil {
		log.Fatalf("Discovery failed: %v", err)
	}

	ctrl := container.NewController(cfg)
	tm := tray.NewTrayManager(ctrl)
	tm.Setup()
}
```

- [ ] **Step 3: Commit**

```bash
git add pkg/tray main.go
git commit -m "feat: implement full tray UI and interaction logic"
```

---

### Task 7: Final Build & Verification

**Files:**
- Create: `Makefile`

- [ ] **Step 1: Create a Makefile for easy building**

```makefile
GO=go
BIN=winapps_systray

build:
	export PATH=$(PATH):/usr/local/go/bin && $(GO) build -o $(BIN) main.go

run: build
	./$(BIN)

clean:
	rm -f $(BIN)
```

- [ ] **Step 2: Perform final build**

Run: `export PATH=$PATH:/usr/local/go/bin && make build`
Expected: `winapps_systray` binary produced without errors.

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "chore: add Makefile and final build verification"
```
