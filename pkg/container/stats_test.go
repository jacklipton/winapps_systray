package container

import "testing"

func TestParseStats(t *testing.T) {
	raw := `{"Name":"WinApps","CPUPerc":"12.34%","MemUsage":"4.1GiB / 16GiB","MemPerc":"25.63%"}`
	stats, err := parseStats([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Name != "WinApps" {
		t.Errorf("expected name WinApps, got %s", stats.Name)
	}
	if stats.CPUPercent != 12.34 {
		t.Errorf("expected CPU 12.34, got %f", stats.CPUPercent)
	}
	if stats.MemUsage != "4.1GiB" {
		t.Errorf("expected mem 4.1GiB, got %s", stats.MemUsage)
	}
	if stats.MemPercent != 25.63 {
		t.Errorf("expected mem%% 25.63, got %f", stats.MemPercent)
	}
}

func TestParseIP(t *testing.T) {
	ip := parseIPOutput("172.21.0.2\n")
	if ip != "172.21.0.2" {
		t.Errorf("expected 172.21.0.2, got %s", ip)
	}
}

func TestParseIPEmpty(t *testing.T) {
	ip := parseIPOutput("")
	if ip != "" {
		t.Errorf("expected empty, got %s", ip)
	}
}
