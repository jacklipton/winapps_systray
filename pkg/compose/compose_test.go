package compose

import (
        "os"
        "path/filepath"
        "strings"
        "testing"
)

func TestValidate(t *testing.T) {
        tests := []struct {
                name    string
                cfg     VMConfig
                wantErr bool
        }{
                {
                        name: "valid config",
                        cfg: VMConfig{
                                RAMSize:  "4G",
                                CPUCores: "4",
                                DiskSize: "64G",
                                Version:  "11",
                                Username: "MyUser",
                                Password: "MyPass",
                        },
                        wantErr: false,
                },
                {
                        name: "valid RAM in megabytes",
                        cfg: VMConfig{
                                RAMSize:  "512M",
                                CPUCores: "2",
                                DiskSize: "32G",
                                Version:  "10",
                                Username: "User",
                                Password: "Pass",
                        },
                        wantErr: false,
                },
                {
                        name:    "all empty fields",
                        cfg:     VMConfig{},
                        wantErr: true,
                },
                {
                        name: "invalid RAM format",
                        cfg: VMConfig{
                                RAMSize:  "4GB",
                                CPUCores: "4",
                                DiskSize: "64G",
                                Version:  "11",
                                Username: "User",
                                Password: "Pass",
                        },
                        wantErr: true,
                },
                {
                        name: "CPU cores zero",
                        cfg: VMConfig{
                                RAMSize:  "4G",
                                CPUCores: "0",
                                DiskSize: "64G",
                                Version:  "11",
                                Username: "User",
                                Password: "Pass",
                        },
                        wantErr: true,
                },
                {
                        name: "CPU cores too high",
                        cfg: VMConfig{
                                RAMSize:  "4G",
                                CPUCores: "65",
                                DiskSize: "64G",
                                Version:  "11",
                                Username: "User",
                                Password: "Pass",
                        },
                        wantErr: true,
                },
                {
                        name: "CPU cores not a number",
                        cfg: VMConfig{
                                RAMSize:  "4G",
                                CPUCores: "abc",
                                DiskSize: "64G",
                                Version:  "11",
                                Username: "User",
                                Password: "Pass",
                        },
                        wantErr: true,
                },
                {
                        name: "invalid disk format",
                        cfg: VMConfig{
                                RAMSize:  "4G",
                                CPUCores: "4",
                                DiskSize: "big",
                                Version:  "11",
                                Username: "User",
                                Password: "Pass",
                        },
                        wantErr: true,
                },
                {
                        name: "empty version",
                        cfg: VMConfig{
                                RAMSize:  "4G",
                                CPUCores: "4",
                                DiskSize: "64G",
                                Version:  "",
                                Username: "User",
                                Password: "Pass",
                        },
                        wantErr: true,
                },
                {
                        name: "empty username",
                        cfg: VMConfig{
                                RAMSize:  "4G",
                                CPUCores: "4",
                                DiskSize: "64G",
                                Version:  "11",
                                Username: "",
                                Password: "Pass",
                        },
                        wantErr: true,
                },
                {
                        name: "empty password",
                        cfg: VMConfig{
                                RAMSize:  "4G",
                                CPUCores: "4",
                                DiskSize: "64G",
                                Version:  "11",
                                Username: "User",
                                Password: "",
                        },
                        wantErr: true,
                },
        }

        for _, tt := range tests {
                t.Run(tt.name, func(t *testing.T) {
                        err := Validate(&tt.cfg)
                        if (err != nil) != tt.wantErr {
                                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
                        }
                })
        }
}

const testComposeYAML = `# WinApps compose file
name: "winapps"
volumes:
  data:
services:
  windows:
    image: ghcr.io/dockur/windows:latest
    container_name: WinApps
    environment:
      VERSION: "11"
      RAM_SIZE: "4G" # RAM allocated to the Windows VM.
      CPU_CORES: "4" # CPU cores allocated to the Windows VM.
      DISK_SIZE: "64G" # Size of the primary hard disk.
      USERNAME: "MyWindowsUser"
      PASSWORD: "MyWindowsPassword"
      HOME: "${HOME}"
    ports:
      - 8006:8006
`

func TestLoad(t *testing.T) {
        dir := t.TempDir()
        path := filepath.Join(dir, "compose.yaml")
        os.WriteFile(path, []byte(testComposeYAML), 0644)

        cfg, err := Load(path, "windows")
        if err != nil {
                t.Fatalf("Load() error: %v", err)
        }
        if cfg.RAMSize != "4G" {
                t.Errorf("RAMSize = %q, want %q", cfg.RAMSize, "4G")
        }
        if cfg.CPUCores != "4" {
                t.Errorf("CPUCores = %q, want %q", cfg.CPUCores, "4")
        }
        if cfg.DiskSize != "64G" {
                t.Errorf("DiskSize = %q, want %q", cfg.DiskSize, "64G")
        }
        if cfg.Version != "11" {
                t.Errorf("Version = %q, want %q", cfg.Version, "11")
        }
        if cfg.Username != "MyWindowsUser" {
                t.Errorf("Username = %q, want %q", cfg.Username, "MyWindowsUser")
        }
        if cfg.Password != "MyWindowsPassword" {
                t.Errorf("Password = %q, want %q", cfg.Password, "MyWindowsPassword")
        }
}

const testComposeMissingKeys = `name: "winapps"
services:
  windows:
    image: ghcr.io/dockur/windows:latest
    environment:
      VERSION: "11"
      HOME: "${HOME}"
`

func TestLoadMissingKeys(t *testing.T) {
        dir := t.TempDir()
        path := filepath.Join(dir, "compose.yaml")
        os.WriteFile(path, []byte(testComposeMissingKeys), 0644)

        cfg, err := Load(path, "windows")
        if err != nil {
                t.Fatalf("Load() error: %v", err)
        }
        if cfg.Version != "11" {
                t.Errorf("Version = %q, want %q", cfg.Version, "11")
        }
        // Missing keys should be empty strings
        if cfg.RAMSize != "" {
                t.Errorf("RAMSize = %q, want empty", cfg.RAMSize)
        }
        if cfg.CPUCores != "" {
                t.Errorf("CPUCores = %q, want empty", cfg.CPUCores)
        }
        if cfg.DiskSize != "" {
                t.Errorf("DiskSize = %q, want empty", cfg.DiskSize)
        }
}

func TestLoadMissingService(t *testing.T) {
        dir := t.TempDir()
        path := filepath.Join(dir, "compose.yaml")
        os.WriteFile(path, []byte(testComposeMissingKeys), 0644)

        _, err := Load(path, "nonexistent")
        if err == nil {
                t.Error("Load() expected error for missing service, got nil")
        }
}

func TestLoadMalformedYAML(t *testing.T) {
        dir := t.TempDir()
        path := filepath.Join(dir, "compose.yaml")
        os.WriteFile(path, []byte("not: valid: yaml: [[[ "), 0644)

        _, err := Load(path, "windows")
        if err == nil {
                t.Error("Load() expected error for malformed YAML, got nil")
        }
}

func TestLoadFileNotFound(t *testing.T) {
        _, err := Load("/nonexistent/compose.yaml", "windows")
        if err == nil {
                t.Error("Load() expected error for missing file, got nil")
        }
}

func TestSave(t *testing.T) {
        dir := t.TempDir()
        path := filepath.Join(dir, "compose.yaml")
        os.WriteFile(path, []byte(testComposeYAML), 0644)

        newCfg := &VMConfig{
                RAMSize:  "8G",
                CPUCores: "8",
                DiskSize: "128G",
                Version:  "10",
                Username: "NewUser",
                Password: "NewPass",
        }

        if err := Save(path, "windows", newCfg); err != nil {
                t.Fatalf("Save() error: %v", err)
        }

        // Re-load and verify values changed
        loaded, err := Load(path, "windows")
        if err != nil {
                t.Fatalf("Load() after Save error: %v", err)
        }
        if loaded.RAMSize != "8G" {
                t.Errorf("RAMSize = %q, want %q", loaded.RAMSize, "8G")
        }
        if loaded.CPUCores != "8" {
                t.Errorf("CPUCores = %q, want %q", loaded.CPUCores, "8")
        }
        if loaded.DiskSize != "128G" {
                t.Errorf("DiskSize = %q, want %q", loaded.DiskSize, "128G")
        }
        if loaded.Version != "10" {
                t.Errorf("Version = %q, want %q", loaded.Version, "10")
        }
        if loaded.Username != "NewUser" {
                t.Errorf("Username = %q, want %q", loaded.Username, "NewUser")
        }
        if loaded.Password != "NewPass" {
                t.Errorf("Password = %q, want %q", loaded.Password, "NewPass")
        }
}

func TestSavePreservesComments(t *testing.T) {
        dir := t.TempDir()
        path := filepath.Join(dir, "compose.yaml")
        os.WriteFile(path, []byte(testComposeYAML), 0644)

        newCfg := &VMConfig{
                RAMSize:  "8G",
                CPUCores: "4",
                DiskSize: "64G",
                Version:  "11",
                Username: "MyWindowsUser",
                Password: "MyWindowsPassword",
        }

        if err := Save(path, "windows", newCfg); err != nil {
                t.Fatalf("Save() error: %v", err)
        }

        data, _ := os.ReadFile(path)
        content := string(data)

        // Comments should be preserved
        if !strings.Contains(content, "# RAM allocated to the Windows VM.") {
                t.Error("expected RAM comment to be preserved")
        }
        if !strings.Contains(content, "# CPU cores allocated to the Windows VM.") {
                t.Error("expected CPU comment to be preserved")
        }
        // The changed value should be present
        if !strings.Contains(content, "8G") {
                t.Error("expected new RAM_SIZE value 8G in output")
        }
}

func TestSaveCreatesBackup(t *testing.T) {
        dir := t.TempDir()
        path := filepath.Join(dir, "compose.yaml")
        original := []byte(testComposeYAML)
        os.WriteFile(path, original, 0644)

        newCfg := &VMConfig{
                RAMSize:  "8G",
                CPUCores: "4",
                DiskSize: "64G",
                Version:  "11",
                Username: "MyWindowsUser",
                Password: "MyWindowsPassword",
        }

        if err := Save(path, "windows", newCfg); err != nil {
                t.Fatalf("Save() error: %v", err)
        }

        bakPath := path + ".bak"
        bakData, err := os.ReadFile(bakPath)
        if err != nil {
                t.Fatalf("backup file not found: %v", err)
        }

        if string(bakData) != string(original) {
                t.Error("backup content does not match original")
        }
}

func TestSaveSkipsMissingKeys(t *testing.T) {
        dir := t.TempDir()
        path := filepath.Join(dir, "compose.yaml")
        os.WriteFile(path, []byte(testComposeMissingKeys), 0644)

        newCfg := &VMConfig{
                RAMSize:  "8G",
                CPUCores: "4",
                DiskSize: "64G",
                Version:  "10",
                Username: "NewUser",
                Password: "NewPass",
        }

        if err := Save(path, "windows", newCfg); err != nil {
                t.Fatalf("Save() error: %v", err)
        }

        // VERSION existed and should be updated
        loaded, _ := Load(path, "windows")
        if loaded.Version != "10" {
                t.Errorf("Version = %q, want %q", loaded.Version, "10")
        }

        // RAM_SIZE did NOT exist in original — should NOT be injected
        data, _ := os.ReadFile(path)
        if strings.Contains(string(data), "RAM_SIZE") {
                t.Error("Save should not inject RAM_SIZE when it didn't exist in original")
        }
}
