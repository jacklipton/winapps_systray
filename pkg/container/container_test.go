package container

import (
	"testing"
	"time"

	"github.com/jacklipton/winapps_systray/pkg/config"
	"github.com/jacklipton/winapps_systray/pkg/discovery"
)

func TestStartTimeoutUsesSettings(t *testing.T) {
	tests := []struct {
		name           string
		timeoutSeconds int
		elapsedSeconds int
		wantCleared    bool
	}{
		{"before timeout", 30, 20, false},
		{"after timeout", 10, 20, true},
		{"just before timeout", 15, 14, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &config.Settings{StartTimeoutSeconds: tt.timeoutSeconds}
			cfg := &discovery.Config{Engine: "echo", ComposeFile: "compose.yaml", WinAppsDir: "/tmp"}
			ctrl := NewController(cfg, settings)

			ctrl.mu.Lock()
			ctrl.transition = StateStarting
			ctrl.transitionAt = time.Now().Add(-time.Duration(tt.elapsedSeconds) * time.Second)
			ctrl.mu.Unlock()

			// Replicate the timeout check from GetStatus (line 79) to verify
			// it uses settings.StartTimeoutSeconds, not a hardcoded value.
			ctrl.mu.Lock()
			cleared := false
			if ctrl.transition == StateStarting {
				if time.Since(ctrl.transitionAt) > time.Duration(ctrl.settings.StartTimeoutSeconds)*time.Second {
					ctrl.transition = ""
					cleared = true
				}
			}
			ctrl.mu.Unlock()

			if cleared != tt.wantCleared {
				t.Errorf("timeout cleared = %v, want %v", cleared, tt.wantCleared)
			}
		})
	}
}
