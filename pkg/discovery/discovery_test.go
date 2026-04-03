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

	path, err := findWinAppsDir(tempHome)
	if err != nil {
		t.Fatalf("Expected to find winapps dir, got error: %v", err)
	}
	if path != winappsDir {
		t.Errorf("Expected %s, got %s", winappsDir, path)
	}
}
