# WinApps Systray UI Redesign

## Overview

Migrate the system tray UI from `getlantern/systray` (archived, limited) to a full GTK3 stack using `gotk3`. The redesign adds animated tray icons, an information-dense menu, a GTK dashboard window, and configurable desktop notifications. The goal is a polished, professional feel with power-user information density.

## Motivation

The current UI has three problems:
1. **Icons are placeholder 16x16 PNGs** — hard to distinguish at a glance.
2. **Menu is bare** — just "Status: Running" and Start/Stop with no detail.
3. **Transitions feel uncertain** — a static yellow icon and disabled button give no feedback.

Additionally, `getlantern/systray` is archived and unmaintained, making it a liability.

## Approach

**Full GTK migration (Approach B).** Replace `getlantern/systray` entirely with GTK's AppIndicator/StatusNotifier for the tray icon and GTK for the menu and dashboard. Single UI framework, native theme integration.

## Design

### 1. Tray Icon System

All icon states derive from the existing `winapps-systray.svg` (blue rounded square with 4-pane Windows grid).

**Static states:**
- **Running** — Full-color blue background (`#0078D4`), bright white panes (opacity 0.95/0.85/0.85/0.7).
- **Stopped** — Grey background (`#555`), dimmed panes (opacity 0.4/0.3/0.3/0.2). Immediately obvious the VM is off.

**Animated transitions (frame-swapping via GTK timer, ~150ms per frame):**
- **Starting** — 4 frames, each highlighting one pane at full brightness while others stay dim. Clockwise sweep: top-left → top-right → bottom-right → bottom-left. Blue background at reduced opacity (0.6). Feels like "booting up."
- **Stopping** — Same 4 frames in reverse order (bottom-left → bottom-right → top-right → top-left), with the background progressively dimming. Feels like "powering down."

Icons are pre-rendered as PNG byte slices from SVG at build time or embedded as SVG and rendered via GTK/GDK at runtime.

### 2. Tray Menu

GTK menu via AppIndicator with color-coded status header and quick stats.

**Menu structure:**

```
┌──────────────────────────────┐
│ ● WinApps — Running          │  ← colored header (green bg when running,
│                               │    grey when stopped, amber when transitioning)
│ Uptime        2h 34m         │  ← shown only when running
│ Memory        4.1 GB         │  ← shown only when running
│ Engine        docker         │  ← always shown
├──────────────────────────────┤
│ ⏹ Stop Windows               │  ← toggle: "Start Windows" / "Stop Windows" / "Starting..."
│ ⚡ Force Kill                 │  ← enabled only during Stopping state
├──────────────────────────────┤
│ 📊 Details...                │  ← opens GTK dashboard window
│ ✕ Quit                       │
└──────────────────────────────┘
```

**Behavior:**
- Status header uses a colored background + dot indicator (green=running, grey=stopped, amber=transitioning).
- Quick stats (uptime, memory) only shown when running. Engine always shown.
- Stats update when the menu is opened (not live while open — GTK menus don't support that well).
- Toggle action disabled during transitions.
- Force Kill only enabled during Stopping state (same as current).
- "Details..." greyed out when stopped (nothing to show).

### 3. Dashboard Window

A compact, fixed-size (~450x400px) GTK dialog opened from "Details..." menu item.

**Layout:**
- **Header** — App icon, "WinApps" title, status indicator (dot + text), live uptime counter (ticks every second).
- **Stats grid (2x2)** — Memory (large number + "GB"), CPU (percentage), Container name, Engine.
- **Network section** — IP address, compose file path.
- **Action buttons** — Stop/Start toggle + Force Kill, same enable/disable logic as tray menu.

**Behavior:**
- Live updates every 2-3 seconds via `docker/podman stats --no-stream --format json` for CPU/memory and `docker/podman inspect` for IP/network.
- Singleton window — clicking "Details..." again brings existing window to front.
- Auto-updates when container state changes. If container stops, stats clear and show "Container stopped." Window stays open.
- Uptime counter ticks via GTK timer (not polling the container).
- Not resizable. Follows system GTK theme.

### 4. Desktop Notifications

Sent via `libnotify` (Go binding or `notify-send` exec). Respects system Do Not Disturb. Uses the app's SVG icon.

**Notification events:**
- **VM started** — "Windows VM is now running"
- **VM stopped** — "Windows VM has stopped"
- **Start/stop failed** — "Failed to start Windows VM" (with brief error detail)
- **Timeout** — "Windows VM is taking longer than expected to start" (after configurable timeout)

Notifications are on by default, configurable via settings.

### 5. Configuration

Settings stored in `~/.config/winapps-systray/settings.json`. The existing `config` file (winapps directory path) remains separate and unchanged.

```json
{
  "notifications": true,
  "poll_interval_seconds": 5,
  "start_timeout_seconds": 60,
  "stop_timeout_seconds": 120
}
```

- **notifications** — enable/disable desktop notifications (default: `true`)
- **poll_interval_seconds** — container status polling interval (default: `5`)
- **start_timeout_seconds** — when to send "taking longer than expected" notification (default: `60`)
- **stop_timeout_seconds** — stop timeout before hinting at Force Kill (default: `120`)

No settings UI in v1 — just the JSON file. Sane defaults mean most users never need to touch it. File is created with defaults on first run if missing.

## Architecture

```
main.go              — GTK application init + main loop (replaces systray.Run)
pkg/discovery/       — unchanged (find winapps dir, detect engine)
pkg/container/       — add Stats() method for CPU/mem/IP via docker stats/inspect
pkg/tray/            — rewrite: GTK AppIndicator + menu (replaces getlantern/systray)
pkg/ui/              — new: GTK dashboard window
pkg/notify/          — new: libnotify wrapper
pkg/config/          — new: settings.json loader with defaults
pkg/icons/           — new: SVG-derived icon set + animation frame management
assets/              — replace byte-slice PNGs with SVG source files
```

**Key dependency changes:**
- Remove: `github.com/getlantern/systray`
- Add: `github.com/gotk3/gotk3` for GTK3 bindings
- Add: AppIndicator bindings for tray icon (via gotk3 CGo wrapper around libayatana-appindicator3)
- Build deps: `libayatana-appindicator3-dev`, `libgtk-3-dev`, `libnotify-dev`

## Error Handling

- **Missing WinApps dir** — same as current: log.Fatalf on startup.
- **Docker/Podman not running** — tray shows "Error" state, menu shows error detail, retry on next poll.
- **Stats command failure** — dashboard shows "—" for unavailable metrics, doesn't crash.
- **Notification failure** — silently ignored (non-critical).
- **Settings file corrupt** — log warning, use defaults.

## Performance Targets

Same as original spec:
- Memory footprint: < 15MB RAM (GTK may use slightly more than pure systray, but still well under)
- Startup time: < 500ms
- CPU usage: negligible (polling only, GTK idle when menu closed)

## Out of Scope (v1)

- Resource usage graphs (CPU/memory over time) — potential future GTK DrawingArea addition
- Settings UI — edit JSON manually for now
- Multiple container management — single WinApps container only
- Windows app launching — that's what winapps-launcher does (complementary project)
