# WinApps Systray

A system tray application for managing [WinApps](https://github.com/winapps-org/winapps) containers on Linux.

Start/stop your Windows VM with a single click and free up 4GB+ of RAM when you're not using it.

## Features

- **One-click toggle** to start and stop your WinApps container
- **Animated tray icons** — smooth transitions during startup/shutdown
- **Information-dense menu** — live uptime, memory usage, and engine status
- **GTK Dashboard** — detailed resource stats and network info
- **Desktop notifications** — configurable alerts for state changes
- **Auto-discovery** of your winapps directory
- **Docker and Podman** support
- **Autostart** with your desktop session

## Install

### From package (Fedora/Ubuntu)

Download the latest `.rpm` or `.deb` from [Releases](https://github.com/jacklipton/winapps_systray/releases):

```bash
# Fedora
sudo dnf install winapps-systray-*.rpm

# Ubuntu/Debian
sudo dpkg -i winapps-systray-*.deb
```

### From source

Build dependencies (Fedora):

```bash
sudo dnf install libayatana-appindicator-gtk3-devel gtk3-devel golang
```

Build dependencies (Ubuntu/Debian):

```bash
sudo apt install libayatana-appindicator3-dev libgtk-3-dev golang
```

Then build and install:

```bash
git clone https://github.com/jacklipton/winapps_systray.git
cd winapps_systray

# System-wide (requires sudo)
make install-autostart

# User-local (no sudo needed)
make user-install
```

## Configuration

The app auto-discovers your winapps directory by checking these locations in order:

1. `WINAPPS_DIR` environment variable
2. `~/.config/winapps-systray/config` (a file containing the path)
3. `~/winapps`
4. `~/.winapps`
5. `~/Documents/winapps`

The directory must contain a `compose.yaml` (or `compose.yml` / `docker-compose.yaml`).

To set a custom path:

```bash
# Option A: environment variable
export WINAPPS_DIR=/path/to/your/winapps

# Option B: config file
mkdir -p ~/.config/winapps-systray
echo "/path/to/your/winapps" > ~/.config/winapps-systray/config
```

## Building packages

To build `.rpm` or `.deb` packages, install [nfpm](https://nfpm.goreleaser.com/install/) and run:

```bash
make rpm   # Fedora/RHEL
make deb   # Ubuntu/Debian
```

## Uninstall

```bash
# System-wide
sudo make uninstall

# User-local
make user-uninstall
```

## Project structure

- `pkg/discovery` — finds the winapps directory and detects docker/podman
- `pkg/container` — wraps compose commands and tracks container state
- `pkg/config` — user settings and defaults
- `pkg/notify` — desktop notifications via notify-send
- `pkg/icons` — SVG generation and animation management
- `pkg/indicator` — CGo wrapper for libayatana-appindicator
- `pkg/tray` — GTK system tray UI and event loop
- `pkg/ui` — GTK dashboard window

## License

MIT
