package compose

import (
        "fmt"
        "regexp"
        "strconv"
)

// VMConfig holds the editable VM environment variables from a compose file.
type VMConfig struct {
        RAMSize  string // e.g. "4G"
        CPUCores string // e.g. "4"
        DiskSize string // e.g. "64G"
        Version  string // e.g. "11"
        Username string
        Password string
}

// envKeys maps VMConfig fields to compose environment variable names.
var envKeys = map[string]string{
        "RAMSize":  "RAM_SIZE",
        "CPUCores": "CPU_CORES",
        "DiskSize": "DISK_SIZE",
        "Version":  "VERSION",
        "Username": "USERNAME",
        "Password": "PASSWORD",
}

var sizePattern = regexp.MustCompile(`^\d+[GM]$`)

// Validate checks that all VMConfig fields meet their constraints.
func Validate(cfg *VMConfig) error {
        if !sizePattern.MatchString(cfg.RAMSize) {
                return fmt.Errorf("invalid RAM size %q: must be a number followed by G or M (e.g. 4G, 512M)", cfg.RAMSize)
        }

        cores, err := strconv.Atoi(cfg.CPUCores)
        if err != nil || cores < 1 || cores > 64 {
                return fmt.Errorf("invalid CPU cores %q: must be an integer between 1 and 64", cfg.CPUCores)
        }

        if !sizePattern.MatchString(cfg.DiskSize) {
                return fmt.Errorf("invalid disk size %q: must be a number followed by G or M (e.g. 64G)", cfg.DiskSize)
        }

        if cfg.Version == "" {
                return fmt.Errorf("Windows version must not be empty")
        }

        if cfg.Username == "" {
                return fmt.Errorf("username must not be empty")
        }

        if cfg.Password == "" {
                return fmt.Errorf("password must not be empty")
        }

        return nil
}
