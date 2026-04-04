package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindWinAppsDir(t *testing.T) {
	tempHome := t.TempDir()
	winappsDir := filepath.Join(tempHome, "winapps")
	os.Mkdir(winappsDir, 0755)
	os.WriteFile(filepath.Join(winappsDir, "compose.yaml"), []byte("name: \"winapps\""), 0644)

	path, composeFile, err := findWinAppsDir(tempHome)
	if err != nil {
		t.Fatalf("Expected to find winapps dir, got error: %v", err)
	}
	if path != winappsDir {
		t.Errorf("Expected %s, got %s", winappsDir, path)
	}
	if composeFile != "compose.yaml" {
		t.Errorf("Expected compose.yaml, got %s", composeFile)
	}
}

func TestFindWinAppsDirEnvVar(t *testing.T) {
	tempDir := t.TempDir()
	os.WriteFile(filepath.Join(tempDir, "compose.yaml"), []byte("name: winapps"), 0644)

	t.Setenv("WINAPPS_DIR", tempDir)
	path, _, err := findWinAppsDir(t.TempDir())
	if err != nil {
		t.Fatalf("Expected env var path, got error: %v", err)
	}
	if path != tempDir {
		t.Errorf("Expected %s, got %s", tempDir, path)
	}
}

func TestFindWinAppsDirEnvVarInvalid(t *testing.T) {
	t.Setenv("WINAPPS_DIR", "/nonexistent/path")
	_, _, err := findWinAppsDir(t.TempDir())
	if err == nil {
		t.Fatal("Expected error for invalid WINAPPS_DIR")
	}
}

func TestFindWinAppsDirConfigFile(t *testing.T) {
	tempHome := t.TempDir()
	winappsDir := t.TempDir()
	os.WriteFile(filepath.Join(winappsDir, "compose.yaml"), []byte("name: winapps"), 0644)

	configDir := filepath.Join(tempHome, ".config", "winapps-systray")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "config"), []byte(winappsDir+"\n"), 0644)

	t.Setenv("WINAPPS_DIR", "")
	path, _, err := findWinAppsDir(tempHome)
	if err != nil {
		t.Fatalf("Expected config file path, got error: %v", err)
	}
	if path != winappsDir {
		t.Errorf("Expected %s, got %s", winappsDir, path)
	}
}

func TestFindWinAppsDirComposeYml(t *testing.T) {
	tempHome := t.TempDir()
	winappsDir := filepath.Join(tempHome, "winapps")
	os.Mkdir(winappsDir, 0755)
	os.WriteFile(filepath.Join(winappsDir, "compose.yml"), []byte("name: winapps"), 0644)

	t.Setenv("WINAPPS_DIR", "")
	path, composeFile, err := findWinAppsDir(tempHome)
	if err != nil {
		t.Fatalf("Expected to find dir with compose.yml, got error: %v", err)
	}
	if path != winappsDir {
		t.Errorf("Expected %s, got %s", winappsDir, path)
	}
	if composeFile != "compose.yml" {
		t.Errorf("Expected compose.yml, got %s", composeFile)
	}
}

func TestListServices(t *testing.T) {
	tempDir := t.TempDir()

	// Create a dummy "docker" script that outputs service names
	mockDocker := filepath.Join(tempDir, "docker")
	content := "#!/bin/sh\necho \"service1\nservice2\nservice3\""
	os.WriteFile(mockDocker, []byte(content), 0755)

	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	services, err := ListServices(tempDir, "docker")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"service1", "service2", "service3"}
	if len(services) != len(expected) {
		t.Fatalf("expected %d services, got %d", len(expected), len(services))
	}
	for i, s := range services {
		if s != expected[i] {
			t.Errorf("expected %s at index %d, got %s", expected[i], i, s)
		}
	}
}
