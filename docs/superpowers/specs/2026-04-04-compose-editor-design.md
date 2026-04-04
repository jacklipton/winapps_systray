# Compose File Editor — Design Spec

**Date:** 2026-04-04
**Status:** Approved

## Overview

Add a GUI for editing Docker Compose VM configuration (RAM, CPU, disk size, Windows version, username, password) from the existing Settings window. Changes are saved to the compose YAML file preserving comments and formatting. A VM restart is required for changes to take effect (user-initiated, not automatic).

## Scope

### Editable Fields

| Field | Env Var | UI Widget | Validation |
|-------|---------|-----------|------------|
| RAM Size | `RAM_SIZE` | Dropdown with presets (2G, 4G, 8G, 16G) + custom entry | Must match `\d+[GM]` |
| CPU Cores | `CPU_CORES` | Spin button (1-64) | Positive integer 1-64 |
| Disk Size | `DISK_SIZE` | Text entry | Must match `\d+[GM]` |
| Windows Version | `VERSION` | Text entry | Non-empty string |
| Username | `USERNAME` | Text entry | Non-empty, no YAML-breaking chars |
| Password | `PASSWORD` | Password entry with show/hide toggle | Non-empty, no YAML-breaking chars |

### Out of Scope

- Port mappings, volumes, devices, `stop_grace_period`
- Automatic VM restart after save
- Adding new environment keys that don't already exist in the file

## Architecture

### New Package: `pkg/compose`

A focused package for reading and writing compose file VM settings using `yaml.v3`'s node-based API.

```go
// pkg/compose/compose.go

type VMConfig struct {
    RAMSize    string // e.g. "4G"
    CPUCores   string // e.g. "4"
    DiskSize   string // e.g. "64G"
    Version    string // e.g. "11"
    Username   string
    Password   string
}

// Load reads a compose file and extracts VM env vars from the
// specified service (default "windows") using yaml.v3 node API.
func Load(path, service string) (*VMConfig, error)

// Save backs up the file to <path>.bak, then writes the modified
// YAML tree back, preserving comments and structure.
func Save(path, service string, cfg *VMConfig) error

// Validate checks that all VMConfig fields meet their constraints.
func Validate(cfg *VMConfig) error
```

**Internal approach:**
- `Load` parses the file into a `yaml.Node` tree, walks to `services.<service>.environment`, and reads the six target keys.
- `Save` walks the same path, updates only the nodes whose values changed, and writes the full tree back. Unknown keys and comments are untouched.
- Before writing, `Save` copies the original file to `<path>.bak`. If the backup fails, `Save` returns an error and does not write.

**Validation rules:**
- `RAMSize`: must match pattern `\d+[GM]` (e.g. `1G`, `512M`, `16G`)
- `CPUCores`: positive integer 1-64
- `DiskSize`: must match pattern `\d+[GM]`
- `Version`: non-empty string
- `Username`/`Password`: non-empty, no characters that would break YAML quoting

### Settings Window Changes: `pkg/ui/settings.go`

Refactor the existing Settings window to use a `gtk.Notebook` with two tabs:

**Tab 1: "App Settings"** — exactly what exists today (WinApps directory, primary service, poll interval, notifications) with its own Save button.

**Tab 2: "VM Configuration"** — new tab with compose file fields:

```
+-------------------------------------+
|  App Settings | VM Configuration |  |
+-------------------------------------+
|                                     |
|  RAM Size        [ 4G        v ]   |
|  CPU Cores       [ 4         v ]   |
|  Disk Size       [ 64G         ]   |
|                                     |
|  Windows Version [ 11          ]   |
|  Username        [ MyWindowsUser ] |
|  Password        [ ************ ]  |
|                                     |
|  ! Restart VM for changes to apply  |
|                                     |
|                       [ Save ]      |
+-------------------------------------+
```

- RAM dropdown: common presets (2G, 4G, 8G, 16G) plus custom entry
- CPU cores: spin button (1-64)
- Disk size, version, username: text entries
- Password: GTK entry with visibility toggle
- Warning label shown after save: "Restart VM for changes to take effect"
- Each tab has its own Save button
- VM tab disabled with explanatory label if no compose file found

### `SettingsWindow` Struct Changes

Add `composeFilePath string` field. Passed in from callers.

## Integration & Data Flow

### Startup

No change. The compose file is not read at startup. The VM Configuration tab loads the compose file lazily when the user switches to it.

### Settings Window Construction

`NewSettingsWindow` receives an additional `composeFilePath string` parameter:
- Derived from `discovery.Config.WinAppsDir + "/" + discovery.Config.ComposeFile`
- If empty (no compose file found), VM tab shows "No compose file found" with all fields disabled

The `service` parameter passed to `compose.Load`/`compose.Save` uses `config.Settings.PrimaryService` (falling back to `"windows"` if empty), consistent with how the rest of the app identifies the target service.

### Save Flow (VM Tab)

1. Validate all fields via `compose.Validate(cfg)`
2. If invalid, show inline error labels next to offending fields
3. Call `compose.Save(path, service, &vmCfg)` — creates `.bak` then writes
4. Show "Saved. Restart VM for changes to take effect." info banner
5. No restart triggered

### Files Changed

| File | Change |
|------|--------|
| `pkg/ui/settings.go` | Refactor `Show()` to use notebook, move current content to tab 1, add VM tab 2. Add `composeFilePath` field. |
| `pkg/tray/tray.go` | Pass compose file path when creating SettingsWindow (line ~106) |
| `main.go` | Pass compose file path when creating SettingsWindow for initial setup (line ~89) |
| `go.mod` | Add `gopkg.in/yaml.v3` dependency |

### New Files

| File | Purpose |
|------|---------|
| `pkg/compose/compose.go` | Load, Save, Validate functions |
| `pkg/compose/compose_test.go` | Unit tests with embedded YAML fixtures |

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Compose file not found | VM tab disabled: "No compose file found in WinApps directory." |
| Missing expected env keys | `Load` returns zero value for missing fields; UI shows empty. `Save` only updates keys that already exist — never injects new keys. |
| Backup failure | `Save` returns error, does not write modified file. UI shows error dialog. |
| Malformed YAML | `Load` returns error. VM tab shows "Could not parse compose file" with detail, fields disabled. |
| Service not found | Same as malformed — error message, fields disabled. |
| Concurrent external edits | No locking. Our save overwrites, but `.bak` preserves previous state. Acceptable for a desktop tool. |

## Testing

### `pkg/compose/compose_test.go`

- **TestLoad** — parse sample YAML, verify all six fields extracted
- **TestLoadMissingKeys** — env vars absent, verify zero values
- **TestLoadMissingService** — no "windows" service, verify error
- **TestSave** — modify values, save, re-load, verify. Confirm comments preserved in raw content
- **TestSaveCreatesBackup** — verify `.bak` exists with original content
- **TestValidate** — table-driven tests for valid/invalid RAM, CPU, disk, empty fields

### No UI Tests

GTK widget testing is impractical (no existing UI tests in the codebase). The compose package carries the testable logic; the UI is a thin layer.

### Test Fixtures

Embedded as string constants in the test file. One fixture mirrors the real WinApps compose file structure with comments.
