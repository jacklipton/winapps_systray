# UI Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate from `getlantern/systray` to a full GTK3 stack with animated tray icons, information-dense menu, GTK dashboard window, and configurable desktop notifications.

**Architecture:** Replace the systray library with `gotk3/gotk3` for GTK bindings and a custom CGo wrapper around `libayatana-appindicator3` for the tray icon. SVG icon variants are written to a temp directory at startup and referenced by name. The GTK main loop replaces `systray.Run()`, with `glib.TimeoutAdd` for polling and `glib.IdleAdd` for thread-safe UI updates from goroutines.

**Tech Stack:** Go, gotk3/gotk3 (GTK3 bindings), libayatana-appindicator3 (CGo), notify-send (exec), Docker/Podman CLI

---

### Task 1: Add gotk3 dependency and verify GTK3 build

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Add gotk3 dependency**

```bash
go get github.com/gotk3/gotk3@latest
```

- [ ] **Step 2: Create a minimal GTK build test**

Create `cmd/gtktest/main.go`:

```go
package main

import "github.com/gotk3/gotk3/gtk"

func main() {
	gtk.Init(nil)
}
```

- [ ] **Step 3: Verify it builds**

Run: `go build ./cmd/gtktest/`
Expected: builds without errors (requires `libgtk-3-dev` / `gtk3-devel` installed)

- [ ] **Step 4: Remove the test file and commit**

```bash
rm -rf cmd/
git add go.mod go.sum
git commit -m "chore: add gotk3 dependency"
```

---

### Task 2: Config package (settings.json)

**Files:**
- Create: `pkg/config/config.go`
- Create: `pkg/config/config_test.go`

- [ ] **Step 1: Write failing test for loading default config**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Notifications {
		t.Error("expected notifications enabled by default")
	}
	if cfg.PollIntervalSeconds != 5 {
		t.Errorf("expected poll interval 5, got %d", cfg.PollIntervalSeconds)
	}
	if cfg.StartTimeoutSeconds != 60 {
		t.Errorf("expected start timeout 60, got %d", cfg.StartTimeoutSeconds)
	}
	if cfg.StopTimeoutSeconds != 120 {
		t.Errorf("expected stop timeout 120, got %d", cfg.StopTimeoutSeconds)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/config/...`
Expected: FAIL — `Load` not defined

- [ ] **Step 3: Write failing test for loading from file**

```go
func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	os.WriteFile(path, []byte(`{"notifications":false,"poll_interval_seconds":10}`), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Notifications {
		t.Error("expected notifications disabled")
	}
	if cfg.PollIntervalSeconds != 10 {
		t.Errorf("expected poll interval 10, got %d", cfg.PollIntervalSeconds)
	}
	// Unset fields should get defaults
	if cfg.StartTimeoutSeconds != 60 {
		t.Errorf("expected start timeout default 60, got %d", cfg.StartTimeoutSeconds)
	}
	if cfg.StopTimeoutSeconds != 120 {
		t.Errorf("expected stop timeout default 120, got %d", cfg.StopTimeoutSeconds)
	}
}
```

- [ ] **Step 4: Implement config.go**

```go
package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
)

type Settings struct {
	Notifications       bool `json:"notifications"`
	PollIntervalSeconds int  `json:"poll_interval_seconds"`
	StartTimeoutSeconds int  `json:"start_timeout_seconds"`
	StopTimeoutSeconds  int  `json:"stop_timeout_seconds"`
}

func defaults() Settings {
	return Settings{
		Notifications:       true,
		PollIntervalSeconds: 5,
		StartTimeoutSeconds: 60,
		StopTimeoutSeconds:  120,
	}
}

// Load reads settings from path. If the file doesn't exist, returns defaults.
// Fields missing from the JSON get default values.
func Load(path string) (*Settings, error) {
	cfg := defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("warning: invalid settings file %s: %v (using defaults)", path, err)
		d := defaults()
		return &d, nil
	}

	// Apply defaults for zero-value fields
	d := defaults()
	if cfg.PollIntervalSeconds == 0 {
		cfg.PollIntervalSeconds = d.PollIntervalSeconds
	}
	if cfg.StartTimeoutSeconds == 0 {
		cfg.StartTimeoutSeconds = d.StartTimeoutSeconds
	}
	if cfg.StopTimeoutSeconds == 0 {
		cfg.StopTimeoutSeconds = d.StopTimeoutSeconds
	}

	return &cfg, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./pkg/config/... -v`
Expected: PASS (both tests)

- [ ] **Step 6: Commit**

```bash
git add pkg/config/
git commit -m "feat: add config package for settings.json with defaults"
```

---

### Task 3: Notify package (desktop notifications)

**Files:**
- Create: `pkg/notify/notify.go`
- Create: `pkg/notify/notify_test.go`

- [ ] **Step 1: Write failing test**

```go
package notify

import "testing"

func TestBuildArgs(t *testing.T) {
	args := buildArgs("WinApps", "VM is running", "/tmp/icon.svg")
	expected := []string{"-i", "/tmp/icon.svg", "WinApps", "VM is running"}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d", len(expected), len(args))
	}
	for i, v := range expected {
		if args[i] != v {
			t.Errorf("arg[%d]: expected %q, got %q", i, v, args[i])
		}
	}
}

func TestBuildArgsNoIcon(t *testing.T) {
	args := buildArgs("WinApps", "VM is running", "")
	expected := []string{"WinApps", "VM is running"}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d", len(expected), len(args))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/notify/...`
Expected: FAIL — `buildArgs` not defined

- [ ] **Step 3: Implement notify.go**

```go
package notify

import "os/exec"

// Send sends a desktop notification via notify-send.
// Failures are silently ignored — notifications are non-critical.
func Send(title, body, iconPath string) {
	args := buildArgs(title, body, iconPath)
	_ = exec.Command("notify-send", args...).Run()
}

func buildArgs(title, body, iconPath string) []string {
	if iconPath != "" {
		return []string{"-i", iconPath, title, body}
	}
	return []string{title, body}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/notify/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/notify/
git commit -m "feat: add notify package wrapping notify-send"
```

---

### Task 4: Icons package (SVG generation and temp dir management)

**Files:**
- Create: `pkg/icons/icons.go`
- Create: `pkg/icons/icons_test.go`

- [ ] **Step 1: Write failing test for SVG generation**

```go
package icons

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupCreatesIconFiles(t *testing.T) {
	dir := t.TempDir()
	mgr, err := Setup(dir)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Should create running, stopped, and 4 transition frames
	expectedFiles := []string{
		"winapps-running.svg",
		"winapps-stopped.svg",
		"winapps-starting-0.svg",
		"winapps-starting-1.svg",
		"winapps-starting-2.svg",
		"winapps-starting-3.svg",
	}
	for _, name := range expectedFiles {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected icon file %s to exist", name)
		}
	}

	if mgr.RunningName() != "winapps-running" {
		t.Errorf("unexpected running name: %s", mgr.RunningName())
	}
	if mgr.StoppedName() != "winapps-stopped" {
		t.Errorf("unexpected stopped name: %s", mgr.StoppedName())
	}
	frames := mgr.StartingFrames()
	if len(frames) != 4 {
		t.Errorf("expected 4 starting frames, got %d", len(frames))
	}
}

func TestSVGContent(t *testing.T) {
	dir := t.TempDir()
	_, err := Setup(dir)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "winapps-running.svg"))
	if err != nil {
		t.Fatal(err)
	}
	svg := string(data)
	if !strings.Contains(svg, "<svg") {
		t.Error("running icon should be valid SVG")
	}
	if !strings.Contains(svg, "#0078D4") {
		t.Error("running icon should use blue background")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/icons/...`
Expected: FAIL — `Setup` not defined

- [ ] **Step 3: Implement icons.go**

```go
package icons

import (
	"fmt"
	"os"
	"path/filepath"
)

// Manager manages tray icon SVG files in a directory.
type Manager struct {
	dir            string
	startingFrames []string
}

// Setup writes all icon SVG files to dir and returns a Manager.
func Setup(dir string) (*Manager, error) {
	icons := map[string]string{
		"winapps-running.svg": svgIcon("#0078D4", [4]float64{0.95, 0.85, 0.85, 0.7}),
		"winapps-stopped.svg": svgIcon("#555555", [4]float64{0.4, 0.3, 0.3, 0.2}),
	}

	// 4 animation frames: each highlights one pane clockwise
	// Pane order: 0=top-left, 1=top-right, 2=bottom-right, 3=bottom-left
	startingFrames := make([]string, 4)
	for i := 0; i < 4; i++ {
		name := fmt.Sprintf("winapps-starting-%d.svg", i)
		opacities := [4]float64{0.3, 0.3, 0.3, 0.3}
		opacities[i] = 0.95
		icons[name] = svgIcon("#0078D4", opacities)
		startingFrames[i] = fmt.Sprintf("winapps-starting-%d", i)
	}

	for name, content := range icons {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("write icon %s: %w", name, err)
		}
	}

	return &Manager{dir: dir, startingFrames: startingFrames}, nil
}

func (m *Manager) Dir() string              { return m.dir }
func (m *Manager) RunningName() string       { return "winapps-running" }
func (m *Manager) StoppedName() string       { return "winapps-stopped" }
func (m *Manager) StartingFrames() []string  { return m.startingFrames }

// StoppingFrames returns the starting frames in reverse order.
func (m *Manager) StoppingFrames() []string {
	frames := m.startingFrames
	reversed := make([]string, len(frames))
	for i, f := range frames {
		reversed[len(frames)-1-i] = f
	}
	return reversed
}

// svgIcon generates an SVG string for the winapps icon.
// bgColor is the background fill. opacities are for panes:
// [0]=top-left, [1]=top-right, [2]=bottom-right, [3]=bottom-left.
func svgIcon(bgColor string, opacities [4]float64) string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">
  <rect width="64" height="64" rx="12" fill="%s"/>
  <rect x="14" y="16" width="16" height="16" rx="1" fill="#fff" opacity="%.2f"/>
  <rect x="34" y="16" width="16" height="16" rx="1" fill="#fff" opacity="%.2f"/>
  <rect x="14" y="36" width="16" height="16" rx="1" fill="#fff" opacity="%.2f"/>
  <rect x="34" y="36" width="16" height="16" rx="1" fill="#fff" opacity="%.2f"/>
</svg>`, bgColor, opacities[0], opacities[1], opacities[3], opacities[2])
}
```

Note: The SVG pane order in the template is top-left, top-right, bottom-left, bottom-right (reading the rect positions). The opacities array uses clockwise order (TL, TR, BR, BL), so index 3 maps to bottom-left rect and index 2 maps to bottom-right rect.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/icons/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/icons/
git commit -m "feat: add icons package with SVG generation and animation frames"
```

---

### Task 5: AppIndicator CGo wrapper

**Files:**
- Create: `pkg/indicator/indicator.go`

- [ ] **Step 1: Implement the CGo wrapper**

```go
package indicator

/*
#cgo pkg-config: ayatana-appindicator3-0.1 gtk+-3.0
#include <libayatana-appindicator/app-indicator.h>
*/
import "C"
import "unsafe"

// Indicator wraps a libayatana-appindicator AppIndicator.
type Indicator struct {
	native *C.AppIndicator
}

// New creates an AppIndicator. iconName is the filename stem (no extension).
// iconThemePath is the directory containing the SVG/PNG files.
func New(id, iconName, iconThemePath string) *Indicator {
	cID := C.CString(id)
	defer C.free(unsafe.Pointer(cID))
	cIcon := C.CString(iconName)
	defer C.free(unsafe.Pointer(cIcon))
	cPath := C.CString(iconThemePath)
	defer C.free(unsafe.Pointer(cPath))

	native := C.app_indicator_new_with_path(
		cID, cIcon,
		C.APP_INDICATOR_CATEGORY_APPLICATION_STATUS,
		cPath,
	)
	C.app_indicator_set_status(native, C.APP_INDICATOR_STATUS_ACTIVE)
	return &Indicator{native: native}
}

// SetIcon changes the displayed icon by name (stem only, no extension).
func (ind *Indicator) SetIcon(iconName string) {
	cName := C.CString(iconName)
	defer C.free(unsafe.Pointer(cName))
	C.app_indicator_set_icon_full(ind.native, cName, cName)
}

// SetMenu attaches a GtkMenu to the indicator.
// menuPtr should be obtained from gotk3's menu.Native().
func (ind *Indicator) SetMenu(menuPtr uintptr) {
	C.app_indicator_set_menu(ind.native, (*C.GtkMenu)(unsafe.Pointer(menuPtr)))
}
```

- [ ] **Step 2: Verify it builds**

Run: `go build ./pkg/indicator/`
Expected: builds without errors (requires `libayatana-appindicator-gtk3-devel` installed)

- [ ] **Step 3: Commit**

```bash
git add pkg/indicator/
git commit -m "feat: add AppIndicator CGo wrapper for tray icon"
```

---

### Task 6: Container stats (add Stats method)

**Files:**
- Modify: `pkg/container/container.go`
- Create: `pkg/container/stats.go`
- Create: `pkg/container/stats_test.go`

- [ ] **Step 1: Write failing test for stats parsing**

```go
package container

import "testing"

func TestParseStats(t *testing.T) {
	raw := `{"Name":"WinApps","CPUPerc":"12.34%","MemUsage":"4.1GiB / 16GiB","MemPerc":"25.63%"}`
	stats, err := parseStats([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Name != "WinApps" {
		t.Errorf("expected name WinApps, got %s", stats.Name)
	}
	if stats.CPUPercent != 12.34 {
		t.Errorf("expected CPU 12.34, got %f", stats.CPUPercent)
	}
	if stats.MemUsage != "4.1GiB" {
		t.Errorf("expected mem 4.1GiB, got %s", stats.MemUsage)
	}
	if stats.MemPercent != 25.63 {
		t.Errorf("expected mem%% 25.63, got %f", stats.MemPercent)
	}
}

func TestParseIP(t *testing.T) {
	ip := parseIPOutput("172.21.0.2\n")
	if ip != "172.21.0.2" {
		t.Errorf("expected 172.21.0.2, got %s", ip)
	}
}

func TestParseIPEmpty(t *testing.T) {
	ip := parseIPOutput("")
	if ip != "" {
		t.Errorf("expected empty, got %s", ip)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/container/... -run TestParse`
Expected: FAIL — `parseStats` not defined

- [ ] **Step 3: Implement stats.go**

```go
package container

import (
	"encoding/json"
	"fmt"
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/container/... -run TestParse -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/container/stats.go pkg/container/stats_test.go
git commit -m "feat: add container stats (CPU, memory, IP) via docker stats/inspect"
```

---

### Task 7: Rewrite tray package (GTK AppIndicator + menu)

**Files:**
- Rewrite: `pkg/tray/tray.go`

- [ ] **Step 1: Rewrite tray.go with GTK menu and AppIndicator**

```go
package tray

import (
	"fmt"
	"time"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/jacklipton/winapps_systray/pkg/config"
	"github.com/jacklipton/winapps_systray/pkg/container"
	"github.com/jacklipton/winapps_systray/pkg/icons"
	"github.com/jacklipton/winapps_systray/pkg/indicator"
	"github.com/jacklipton/winapps_systray/pkg/notify"
)

// OnDashboard is called when the user clicks "Details...".
// Set by main before calling Setup.
type OnDashboardFunc func()

type TrayManager struct {
	ctrl    *container.Controller
	cfg     *config.Settings
	iconMgr *icons.Manager
	ind     *indicator.Indicator

	// Menu items (need references for dynamic updates)
	mStatus  *gtk.MenuItem
	mUptime  *gtk.MenuItem
	mMemory  *gtk.MenuItem
	mEngine  *gtk.MenuItem
	mToggle  *gtk.MenuItem
	mKill    *gtk.MenuItem
	mDetails *gtk.MenuItem

	lastState container.State
	startedAt time.Time
	animFrame int
	animTimer uint

	OnDashboard OnDashboardFunc
}

func NewTrayManager(ctrl *container.Controller, cfg *config.Settings, iconMgr *icons.Manager) *TrayManager {
	return &TrayManager{
		ctrl:    ctrl,
		cfg:     cfg,
		iconMgr: iconMgr,
	}
}

func (t *TrayManager) Setup() {
	// Build GTK menu
	menu, _ := gtk.MenuNew()

	t.mStatus = addMenuItem(menu, "WinApps — Unknown", nil)
	t.mStatus.SetSensitive(false)

	addSeparator(menu)

	t.mUptime = addMenuItem(menu, "Uptime        —", nil)
	t.mUptime.SetSensitive(false)
	t.mMemory = addMenuItem(menu, "Memory        —", nil)
	t.mMemory.SetSensitive(false)
	t.mEngine = addMenuItem(menu, fmt.Sprintf("Engine        %s", t.ctrl.Engine()), nil)
	t.mEngine.SetSensitive(false)

	addSeparator(menu)

	t.mToggle = addMenuItem(menu, "Start Windows", func() { go t.onToggle() })
	t.mKill = addMenuItem(menu, "Force Kill", func() { go t.ctrl.Kill() })
	t.mKill.SetSensitive(false)

	addSeparator(menu)

	t.mDetails = addMenuItem(menu, "Details...", func() {
		if t.OnDashboard != nil {
			t.OnDashboard()
		}
	})
	addMenuItem(menu, "Quit", func() { gtk.MainQuit() })

	menu.ShowAll()

	// Create AppIndicator
	t.ind = indicator.New("winapps-systray", t.iconMgr.StoppedName(), t.iconMgr.Dir())
	t.ind.SetMenu(menu.Native())

	// Start status polling via GTK timer
	glib.TimeoutAdd(uint(t.cfg.PollIntervalSeconds*1000), func() bool {
		t.pollAndUpdate()
		return true
	})
}

func (t *TrayManager) pollAndUpdate() {
	status, err := t.ctrl.GetStatus()
	if err != nil {
		return
	}

	prev := t.lastState
	t.lastState = status
	t.updateUI(status)

	// Send notifications on state transitions
	if t.cfg.Notifications && prev != "" && prev != status {
		t.notifyTransition(prev, status)
	}

	// Track uptime start
	if status == container.StateRunning && prev != container.StateRunning {
		t.startedAt = time.Now()
	}
}

func (t *TrayManager) updateUI(state container.State) {
	// Stop any running animation
	t.stopAnimation()

	switch state {
	case container.StateRunning:
		t.ind.SetIcon(t.iconMgr.RunningName())
		t.mStatus.SetLabel("● WinApps — Running")
		t.mToggle.SetLabel("Stop Windows")
		t.mToggle.SetSensitive(true)
		t.mKill.SetSensitive(false)
		t.mDetails.SetSensitive(true)

		// Update stats
		if stats := t.ctrl.GetStats(); stats != nil {
			elapsed := time.Since(t.startedAt)
			t.mUptime.SetLabel(fmt.Sprintf("Uptime        %s", formatDuration(elapsed)))
			t.mMemory.SetLabel(fmt.Sprintf("Memory        %s", stats.MemUsage))
		}

	case container.StateStopped:
		t.ind.SetIcon(t.iconMgr.StoppedName())
		t.mStatus.SetLabel("● WinApps — Stopped")
		t.mToggle.SetLabel("Start Windows")
		t.mToggle.SetSensitive(true)
		t.mKill.SetSensitive(false)
		t.mDetails.SetSensitive(false)
		t.mUptime.SetLabel("Uptime        —")
		t.mMemory.SetLabel("Memory        —")

	case container.StateStarting:
		t.mStatus.SetLabel("● WinApps — Starting...")
		t.mToggle.SetLabel("Starting...")
		t.mToggle.SetSensitive(false)
		t.mKill.SetSensitive(false)
		t.mDetails.SetSensitive(false)
		t.startAnimation(t.iconMgr.StartingFrames())

	case container.StateStopping:
		t.mStatus.SetLabel("● WinApps — Stopping...")
		t.mToggle.SetLabel("Stopping...")
		t.mToggle.SetSensitive(false)
		t.mKill.SetSensitive(true)
		t.startAnimation(t.iconMgr.StoppingFrames())
	}
}

func (t *TrayManager) startAnimation(frames []string) {
	t.animFrame = 0
	t.animTimer = glib.TimeoutAdd(150, func() bool {
		t.ind.SetIcon(frames[t.animFrame%len(frames)])
		t.animFrame++
		return true
	})
}

func (t *TrayManager) stopAnimation() {
	if t.animTimer != 0 {
		glib.SourceRemove(t.animTimer)
		t.animTimer = 0
	}
}

func (t *TrayManager) onToggle() {
	status, _ := t.ctrl.GetStatus()
	if status == container.StateRunning {
		glib.IdleAdd(func() bool { t.updateUI(container.StateStopping); return false })
		t.ctrl.Stop()
		t.ctrl.WaitUntilState(container.StateStopped, time.Duration(t.cfg.StopTimeoutSeconds)*time.Second)
	} else if status == container.StateStopped {
		glib.IdleAdd(func() bool { t.updateUI(container.StateStarting); return false })
		t.ctrl.Start()
		t.ctrl.WaitUntilState(container.StateRunning, time.Duration(t.cfg.StartTimeoutSeconds)*time.Second)
	}
}

func (t *TrayManager) notifyTransition(prev, curr container.State) {
	iconPath := ""
	running := t.iconMgr.Dir() + "/winapps-running.svg"
	stopped := t.iconMgr.Dir() + "/winapps-stopped.svg"

	switch curr {
	case container.StateRunning:
		iconPath = running
		notify.Send("WinApps", "Windows VM is now running", iconPath)
	case container.StateStopped:
		if prev == container.StateRunning || prev == container.StateStopping {
			iconPath = stopped
			notify.Send("WinApps", "Windows VM has stopped", iconPath)
		}
	}
}

func addMenuItem(menu *gtk.Menu, label string, onClick func()) *gtk.MenuItem {
	item, _ := gtk.MenuItemNewWithLabel(label)
	if onClick != nil {
		item.Connect("activate", onClick)
	}
	item.Show()
	menu.Append(item)
	return item
}

func addSeparator(menu *gtk.Menu) {
	sep, _ := gtk.SeparatorMenuItemNew()
	sep.Show()
	menu.Append(sep)
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
```

- [ ] **Step 2: Verify it builds**

Run: `go build ./pkg/tray/`
Expected: builds without errors

- [ ] **Step 3: Commit**

```bash
git add pkg/tray/tray.go
git commit -m "feat: rewrite tray package for GTK AppIndicator with animated icons"
```

---

### Task 8: Dashboard window (GTK)

**Files:**
- Create: `pkg/ui/dashboard.go`

- [ ] **Step 1: Implement dashboard.go**

```go
package ui

import (
	"fmt"
	"time"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/jacklipton/winapps_systray/pkg/container"
)

// Dashboard is the GTK details window.
type Dashboard struct {
	ctrl   *container.Controller
	window *gtk.Window

	// Labels for live updates
	lblStatus  *gtk.Label
	lblUptime  *gtk.Label
	lblMemory  *gtk.Label
	lblCPU     *gtk.Label
	lblName    *gtk.Label
	lblEngine  *gtk.Label
	lblIP      *gtk.Label
	lblCompose *gtk.Label

	btnToggle *gtk.Button
	btnKill   *gtk.Button

	startedAt time.Time
	timerID   uint
	uptimeID  uint
}

func NewDashboard(ctrl *container.Controller) *Dashboard {
	return &Dashboard{ctrl: ctrl}
}

// Show creates and shows the dashboard window. If already open, presents it.
func (d *Dashboard) Show() {
	if d.window != nil {
		d.window.Present()
		return
	}

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("WinApps Details")
	win.SetDefaultSize(450, 400)
	win.SetResizable(false)
	win.Connect("destroy", func() {
		d.cleanup()
		d.window = nil
	})
	d.window = win

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

	// Header
	header := d.buildHeader()
	box.PackStart(header, false, false, 0)

	// Stats grid
	grid := d.buildStatsGrid()
	box.PackStart(grid, false, false, 0)

	// Network section
	network := d.buildNetworkSection()
	box.PackStart(network, false, false, 0)

	// Action buttons
	actions := d.buildActions()
	box.PackEnd(actions, false, false, 8)

	win.Add(box)
	win.ShowAll()

	// Start live updates
	d.timerID = glib.TimeoutAdd(2500, func() bool {
		d.refresh()
		return true
	})
	d.uptimeID = glib.TimeoutAdd(1000, func() bool {
		d.updateUptime()
		return true
	})

	d.refresh()
}

func (d *Dashboard) buildHeader() *gtk.Box {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 12)
	box.SetMarginTop(16)
	box.SetMarginBottom(12)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)

	// Title and status
	titleBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)
	title, _ := gtk.LabelNew("")
	title.SetMarkup("<b><big>WinApps</big></b>")
	title.SetHAlign(gtk.ALIGN_START)
	titleBox.PackStart(title, false, false, 0)

	d.lblStatus, _ = gtk.LabelNew("Unknown")
	d.lblStatus.SetHAlign(gtk.ALIGN_START)
	titleBox.PackStart(d.lblStatus, false, false, 0)
	box.PackStart(titleBox, true, true, 0)

	// Uptime
	uptimeBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)
	uptimeLabel, _ := gtk.LabelNew("")
	uptimeLabel.SetMarkup("<small>Uptime</small>")
	uptimeLabel.SetHAlign(gtk.ALIGN_END)
	uptimeBox.PackStart(uptimeLabel, false, false, 0)

	d.lblUptime, _ = gtk.LabelNew("—")
	d.lblUptime.SetHAlign(gtk.ALIGN_END)
	uptimeBox.PackStart(d.lblUptime, false, false, 0)
	box.PackEnd(uptimeBox, false, false, 0)

	return box
}

func (d *Dashboard) buildStatsGrid() *gtk.Grid {
	grid, _ := gtk.GridNew()
	grid.SetRowHomogeneous(true)
	grid.SetColumnHomogeneous(true)
	grid.SetMarginStart(20)
	grid.SetMarginEnd(20)
	grid.SetMarginBottom(8)
	grid.SetRowSpacing(8)
	grid.SetColumnSpacing(8)

	d.lblMemory = d.addStatCell(grid, "MEMORY", "—", 0, 0)
	d.lblCPU = d.addStatCell(grid, "CPU", "—", 1, 0)
	d.lblName = d.addStatCell(grid, "CONTAINER", "—", 0, 1)
	d.lblEngine = d.addStatCell(grid, "ENGINE", "—", 1, 1)

	return grid
}

func (d *Dashboard) addStatCell(grid *gtk.Grid, title, value string, col, row int) *gtk.Label {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 4)
	box.SetMarginTop(8)
	box.SetMarginBottom(8)

	lbl, _ := gtk.LabelNew("")
	lbl.SetMarkup(fmt.Sprintf("<small>%s</small>", title))
	lbl.SetHAlign(gtk.ALIGN_START)
	box.PackStart(lbl, false, false, 0)

	val, _ := gtk.LabelNew(value)
	val.SetHAlign(gtk.ALIGN_START)
	box.PackStart(val, false, false, 0)

	grid.Attach(box, col, row, 1, 1)
	return val
}

func (d *Dashboard) buildNetworkSection() *gtk.Box {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 4)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)
	box.SetMarginTop(8)
	box.SetMarginBottom(8)

	header, _ := gtk.LabelNew("")
	header.SetMarkup("<small>NETWORK</small>")
	header.SetHAlign(gtk.ALIGN_START)
	box.PackStart(header, false, false, 4)

	ipRow, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	ipLabel, _ := gtk.LabelNew("IP Address")
	ipLabel.SetHAlign(gtk.ALIGN_START)
	ipRow.PackStart(ipLabel, true, true, 0)
	d.lblIP, _ = gtk.LabelNew("—")
	d.lblIP.SetHAlign(gtk.ALIGN_END)
	ipRow.PackEnd(d.lblIP, false, false, 0)
	box.PackStart(ipRow, false, false, 0)

	composeRow, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	composeLabel, _ := gtk.LabelNew("Compose File")
	composeLabel.SetHAlign(gtk.ALIGN_START)
	composeRow.PackStart(composeLabel, true, true, 0)
	d.lblCompose, _ = gtk.LabelNew("—")
	d.lblCompose.SetHAlign(gtk.ALIGN_END)
	composeRow.PackEnd(d.lblCompose, false, false, 0)
	box.PackStart(composeRow, false, false, 0)

	return box
}

func (d *Dashboard) buildActions() *gtk.Box {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)
	box.SetMarginBottom(14)
	box.SetHAlign(gtk.ALIGN_END)

	d.btnToggle, _ = gtk.ButtonNewWithLabel("Stop")
	d.btnToggle.Connect("clicked", func() {
		go func() {
			status, _ := d.ctrl.GetStatus()
			if status == container.StateRunning {
				d.ctrl.Stop()
			} else if status == container.StateStopped {
				d.ctrl.Start()
			}
		}()
	})
	box.PackStart(d.btnToggle, false, false, 0)

	d.btnKill, _ = gtk.ButtonNewWithLabel("Force Kill")
	d.btnKill.SetSensitive(false)
	d.btnKill.Connect("clicked", func() { go d.ctrl.Kill() })
	box.PackStart(d.btnKill, false, false, 0)

	return box
}

func (d *Dashboard) refresh() {
	status, _ := d.ctrl.GetStatus()

	switch status {
	case container.StateRunning:
		d.lblStatus.SetText("● Running")
		d.btnToggle.SetLabel("Stop")
		d.btnToggle.SetSensitive(true)
		d.btnKill.SetSensitive(false)

		if stats := d.ctrl.GetStats(); stats != nil {
			d.lblMemory.SetText(stats.MemUsage)
			d.lblCPU.SetText(fmt.Sprintf("%.1f%%", stats.CPUPercent))
			d.lblName.SetText(stats.Name)
			d.lblIP.SetText(stats.IPAddress)
		}

	case container.StateStopped:
		d.lblStatus.SetText("● Stopped")
		d.lblUptime.SetText("—")
		d.lblMemory.SetText("—")
		d.lblCPU.SetText("—")
		d.lblIP.SetText("—")
		d.btnToggle.SetLabel("Start")
		d.btnToggle.SetSensitive(true)
		d.btnKill.SetSensitive(false)

	case container.StateStarting:
		d.lblStatus.SetText("● Starting...")
		d.btnToggle.SetSensitive(false)
		d.btnKill.SetSensitive(false)

	case container.StateStopping:
		d.lblStatus.SetText("● Stopping...")
		d.btnToggle.SetSensitive(false)
		d.btnKill.SetSensitive(true)
	}

	d.lblEngine.SetText(d.ctrl.Engine())
	d.lblCompose.SetText(d.ctrl.ComposeFile())
}

func (d *Dashboard) updateUptime() {
	status, _ := d.ctrl.GetStatus()
	if status == container.StateRunning && !d.startedAt.IsZero() {
		elapsed := time.Since(d.startedAt)
		d.lblUptime.SetText(formatDuration(elapsed))
	}
}

// SetStartedAt sets the time the container started (tracked by tray manager).
func (d *Dashboard) SetStartedAt(t time.Time) {
	d.startedAt = t
}

func (d *Dashboard) cleanup() {
	if d.timerID != 0 {
		glib.SourceRemove(d.timerID)
		d.timerID = 0
	}
	if d.uptimeID != 0 {
		glib.SourceRemove(d.uptimeID)
		d.uptimeID = 0
	}
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
```

- [ ] **Step 2: Verify it builds**

Run: `go build ./pkg/ui/`
Expected: builds without errors

- [ ] **Step 3: Commit**

```bash
git add pkg/ui/
git commit -m "feat: add GTK dashboard window with live stats"
```

---

### Task 9: Rewrite main.go (GTK main loop)

**Files:**
- Rewrite: `main.go`

- [ ] **Step 1: Rewrite main.go**

```go
package main

import (
	"log"
	"os"
	"path/filepath"

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
	if err != nil {
		log.Fatalf("Discovery failed: %v", err)
	}

	ctrl := container.NewController(discoveryCfg)

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
	dashboard := ui.NewDashboard(ctrl)

	// Set up tray
	tm := tray.NewTrayManager(ctrl, cfg, iconMgr)
	tm.OnDashboard = func() { dashboard.Show() }
	tm.Setup()

	// Run GTK main loop (blocks until Quit)
	gtk.Main()
}
```

- [ ] **Step 2: Verify it builds**

Run: `go build -o winapps-systray .`
Expected: binary produced without errors

- [ ] **Step 3: Commit**

```bash
git add main.go
git commit -m "feat: rewrite main.go with GTK main loop"
```

---

### Task 10: Remove getlantern/systray and clean up

**Files:**
- Modify: `go.mod`
- Remove: `assets/icons.go` (old byte-slice PNGs)
- Modify: `assets/assets.go` (if needed)

- [ ] **Step 1: Remove old imports and assets**

```bash
rm assets/icons.go
```

If `assets/assets.go` is just `package assets` with no content, leave it or remove the whole `assets/` directory if nothing references it.

- [ ] **Step 2: Remove getlantern/systray dependency**

```bash
go mod tidy
```

This will remove `github.com/getlantern/systray` and all its transitive dependencies since nothing imports it anymore.

- [ ] **Step 3: Verify clean build**

Run: `go build -o winapps-systray .`
Expected: builds successfully with no reference to getlantern/systray

- [ ] **Step 4: Verify tests pass**

Run: `go test ./...`
Expected: all tests pass

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "chore: remove getlantern/systray and old placeholder icons"
```

---

### Task 11: Update build dependencies in README and nfpm.yaml

**Files:**
- Modify: `README.md`
- Modify: `nfpm.yaml`

- [ ] **Step 1: Update README build dependencies**

In `README.md`, update the build dependency sections:

Fedora:
```bash
sudo dnf install libayatana-appindicator-gtk3-devel gtk3-devel golang
```

Ubuntu/Debian:
```bash
sudo apt install libayatana-appindicator3-dev libgtk-3-dev golang
```

These are the same as the current README — no change needed for build deps. But add a note about runtime deps if not already present.

- [ ] **Step 2: Update nfpm.yaml if it lists dependencies**

Check if `nfpm.yaml` lists runtime dependencies and add `libayatana-appindicator3-1` and `libnotify` if they're not already there.

- [ ] **Step 3: Verify package builds**

Run: `make build` (or equivalent)
Expected: builds successfully

- [ ] **Step 4: Commit**

```bash
git add README.md nfpm.yaml
git commit -m "docs: update build deps for GTK migration"
```
