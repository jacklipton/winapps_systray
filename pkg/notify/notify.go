package notify

import "os/exec"

// Send sends a desktop notification via notify-send.
// Failures are silently ignored — notifications are non-critical.
func Send(title, body, iconPath string) {
	args := buildArgs(title, body, iconPath)
	_ = exec.Command("notify-send", args...).Run()
}

func buildArgs(title, body, iconPath string) []string {
	if iconPath != "" {
		return []string{"-i", iconPath, title, body}
	}
	return []string{title, body}
}
