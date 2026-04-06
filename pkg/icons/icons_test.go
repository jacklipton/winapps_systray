package icons

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupCreatesIconFiles(t *testing.T) {
	dir := t.TempDir()
	mgr, err := Setup(dir)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Should create running, stopped, 4 starting frames, 4 stopping frames, and dark variants
	expectedFiles := []string{
		"winapps-running.svg",
		"winapps-stopped.svg",
		"winapps-starting-0.svg",
		"winapps-starting-1.svg",
		"winapps-starting-2.svg",
		"winapps-starting-3.svg",
		"winapps-stopping-0.svg",
		"winapps-stopping-1.svg",
		"winapps-stopping-2.svg",
		"winapps-stopping-3.svg",
		"winapps-running-dark.svg",
		"winapps-stopped-dark.svg",
	}
	for _, name := range expectedFiles {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected icon file %s to exist", name)
		}
	}

	if mgr.RunningName() != "winapps-running" {
		t.Errorf("unexpected running name: %s", mgr.RunningName())
	}
	if mgr.StoppedName() != "winapps-stopped" {
		t.Errorf("unexpected stopped name: %s", mgr.StoppedName())
	}
	frames := mgr.StartingFrames()
	if len(frames) != 4 {
		t.Errorf("expected 4 starting frames, got %d", len(frames))
	}
	stopFrames := mgr.StoppingFrames()
	if len(stopFrames) != 4 {
		t.Errorf("expected 4 stopping frames, got %d", len(stopFrames))
	}
}

func TestSVGContent(t *testing.T) {
	dir := t.TempDir()
	_, err := Setup(dir)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "winapps-running.svg"))
	if err != nil {
		t.Fatal(err)
	}
	svg := string(data)
	if !strings.Contains(svg, "<svg") {
		t.Error("running icon should be valid SVG")
	}
	if !strings.Contains(svg, "#0078D4") {
		t.Error("running icon should use blue background")
	}
}
