package notify

import "testing"

func TestBuildArgs(t *testing.T) {
	args := buildArgs("WinApps", "VM is running", "/tmp/icon.svg")
	expected := []string{"-i", "/tmp/icon.svg", "WinApps", "VM is running"}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d", len(expected), len(args))
	}
	for i, v := range expected {
		if args[i] != v {
			t.Errorf("arg[%d]: expected %q, got %q", i, v, args[i])
		}
	}
}

func TestBuildArgsNoIcon(t *testing.T) {
	args := buildArgs("WinApps", "VM is running", "")
	expected := []string{"WinApps", "VM is running"}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d", len(expected), len(args))
	}
}
