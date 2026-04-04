package compose

import (
        "fmt"
        "os"
        "regexp"
        "strconv"

        "gopkg.in/yaml.v3"
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

// Load reads a compose file and extracts VM environment variables
// from the specified service using yaml.v3's node API.
func Load(path, service string) (*VMConfig, error) {
        data, err := os.ReadFile(path)
        if err != nil {
                return nil, fmt.Errorf("read compose file: %w", err)
        }

        var doc yaml.Node
        if err := yaml.Unmarshal(data, &doc); err != nil {
                return nil, fmt.Errorf("parse compose file: %w", err)
        }

        envNode, err := findEnvNode(&doc, service)
        if err != nil {
                return nil, err
        }

        cfg := &VMConfig{}
        envMap := readMappingNode(envNode)

        cfg.RAMSize = envMap["RAM_SIZE"]
        cfg.CPUCores = envMap["CPU_CORES"]
        cfg.DiskSize = envMap["DISK_SIZE"]
        cfg.Version = envMap["VERSION"]
        cfg.Username = envMap["USERNAME"]
        cfg.Password = envMap["PASSWORD"]

        return cfg, nil
}

// findEnvNode walks the YAML node tree to find
// services.<service>.environment and returns that mapping node.
func findEnvNode(doc *yaml.Node, service string) (*yaml.Node, error) {
        if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
                return nil, fmt.Errorf("invalid YAML document")
        }
        root := doc.Content[0]
        if root.Kind != yaml.MappingNode {
                return nil, fmt.Errorf("expected mapping at root")
        }

        servicesNode := findMapValue(root, "services")
        if servicesNode == nil {
                return nil, fmt.Errorf("no 'services' key in compose file")
        }

        serviceNode := findMapValue(servicesNode, service)
        if serviceNode == nil {
                return nil, fmt.Errorf("service %q not found in compose file", service)
        }

        envNode := findMapValue(serviceNode, "environment")
        if envNode == nil {
                return nil, fmt.Errorf("no 'environment' key in service %q", service)
        }

        return envNode, nil
}

// findMapValue finds the value node for a given key in a mapping node.
func findMapValue(mapping *yaml.Node, key string) *yaml.Node {
        if mapping.Kind != yaml.MappingNode {
                return nil
        }
        for i := 0; i < len(mapping.Content)-1; i += 2 {
                if mapping.Content[i].Value == key {
                        return mapping.Content[i+1]
                }
        }
        return nil
}

// readMappingNode reads a YAML mapping node into a string map.
func readMappingNode(node *yaml.Node) map[string]string {
        m := make(map[string]string)
        if node.Kind != yaml.MappingNode {
                return m
        }
        for i := 0; i < len(node.Content)-1; i += 2 {
                m[node.Content[i].Value] = node.Content[i+1].Value
        }
        return m
}
