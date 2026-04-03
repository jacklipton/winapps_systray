# WinApps Systray Design Specification

## Overview
A high-performance, minimal-overhead system tray application for managing the [WinApps](https://github.com/winapps-org/winapps) Docker/Podman container. Designed for Fedora (GNOME) and KDE, this tool provides a native UI to toggle the Windows environment on and off, reducing manual CLI usage and optimizing system resources.

## Objectives
- **Resource Management:** Easily shut down the Windows VM container when not in use to free up 4GB+ of RAM.
- **Convenience:** Single-click start/stop from the system tray.
- **Visual Feedback:** Clear icon-based status indicators.
- **Safety:** Proper handling of "Wait & Verify" for graceful shutdowns, with a "Force Kill" fallback.

## Technology Stack
- **Language:** Go (1.20+)
- **UI Library:** `getlantern/systray` (Native system tray integration)
- **Container Interface:** Docker CLI or Podman CLI (wrapped via `os/exec` or SDK)
- **Build System:** Go modules, static binary compilation

## Core Components

### 1. Discovery Engine
The app will automatically locate the `winapps` installation by:
- Checking `~/winapps` (standard path).
- Parsing `compose.yaml` to identify the container name (default: `WinApps`) and project name.
- Detecting whether `docker` or `podman` is the active container engine.

### 2. State Controller
Manages the lifecycle of the Windows container with the following states:
- **Stopped:** Container is not running.
- **Starting:** `docker compose up -d` has been issued; waiting for container to report "running".
- **Running:** Container is active and healthy.
- **Stopping:** `docker compose stop` has been issued; waiting for exit (up to 120s grace period).

### 3. Tray UI Manager
The menu will include:
- **Status Header:** Displays current state (e.g., "Status: Running").
- **Toggle Action:** "Start Windows" or "Stop Windows" depending on state.
- **Force Kill (Conditional):** Enabled only during the "Stopping" state if the user needs to bypass the grace period.
- **Quit:** Cleanly exits the tray app.

## Icon State Mapping
- **Offline:** Gray/Monochrome Windows Logo.
- **Starting/Stopping:** Yellow/Orange or Animated (if supported) icon.
- **Online:** Blue/Colored Windows Logo.

## Error Handling
- **Missing WinApps:** Display a notification and a menu error if `~/winapps` or `compose.yaml` is not found.
- **Daemon Down:** Notify if Docker/Podman is not running.
- **Timeout:** If the container fails to stop within the grace period, provide a "Force Kill" prompt.

## Performance Targets
- **Memory Footprint:** < 15MB RAM.
- **Startup Time:** < 500ms.
- **CPU Usage:** Negligible (polling only during state transitions).
