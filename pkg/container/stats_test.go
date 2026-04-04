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

func TestParseStatsRobust(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantCPU float64
		wantMem float64
	}{
		{
			name:    "Docker string CPUPerc",
			json:    `{"Name":"WinApps","CPUPerc":"12.34%","MemUsage":"4.1GiB / 16GiB","MemPerc":"25.63%"}`,
			wantCPU: 12.34,
			wantMem: 25.63,
		},
		{
			name:    "Docker float CPUPercent",
			json:    `{"Name":"WinApps","CPUPercent":5.67,"MemUsage":"4.1GiB / 16GiB","MemPercent":10.5}`,
			wantCPU: 5.67,
			wantMem: 10.5,
		},
		{
			name:    "Podman string CPUPerc",
			json:    `{"Name":"WinApps","CPUPerc":"1.2%","MemUsage":"100MB / 1GB","MemPerc":"10.0%"}`,
			wantCPU: 1.2,
			wantMem: 10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := parseStats([]byte(tt.json))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if stats.CPUPercent != tt.wantCPU {
				t.Errorf("expected CPU %f, got %f", tt.wantCPU, stats.CPUPercent)
			}
			if stats.MemPercent != tt.wantMem {
				t.Errorf("expected mem%% %f, got %f", tt.wantMem, stats.MemPercent)
			}
		})
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
