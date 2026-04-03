# WinApps Systray

A high-performance, minimal-overhead system tray application for managing [WinApps](https://github.com/winapps-org/winapps) containers on Linux (GNOME, KDE, etc.).

## Features

- **Toggle Windows:** Start and stop your WinApps container with a single click.
- **Resource Management:** Easily free up 4GB+ of RAM by shutting down the Windows VM when not in use.
- **Status Indicators:** Color-coded tray icons reflect the current state (Running, Stopped, Starting, Stopping).
- **Wait & Verify:** Gracefully handles the shutdown process with a 120s grace period.
- **Force Kill:** Option to forcefully terminate the container if it hangs during shutdown.
- **Auto-Discovery:** Automatically detects `~/winapps` and works with both **Docker** and **Podman**.

## Prerequisites

To build this application on Fedora, you need the following development headers:

```bash
sudo dnf install libayatana-appindicator-gtk3-devel gtk3-devel golang
```

## Installation & Building

1. **Clone or move to the project directory:**
   ```bash
   cd ~/code_projects/winapps_systray
   ```

2. **Build the binary:**
   ```bash
   make build
   ```

3. **Run the application:**
   ```bash
   ./winapps_systray &
   ```

## Configuration

The application automatically looks for your WinApps installation in `~/winapps`. It parses your `compose.yaml` to identify the container name and uses your system's default container engine (Docker or Podman).

## Development

The project is structured as follows:
- `pkg/discovery`: Logic for finding the WinApps directory and configuration.
- `pkg/container`: Wrapper for Docker/Podman CLI commands and status polling.
- `pkg/tray`: System tray UI management and event loop.
- `assets`: Embedded 16x16 icons for status representation.

## License
MIT
