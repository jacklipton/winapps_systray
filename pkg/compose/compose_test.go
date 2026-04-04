package compose

import (
        "os"
        "path/filepath"
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
