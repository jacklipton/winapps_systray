package icons

import (
	"fmt"
	"os"
	"path/filepath"
)

// Manager manages tray icon SVG files in a directory.
type Manager struct {
	dir             string
	startingFrames  []string
	stoppingFrames  []string
}

// Setup writes all icon SVG files to dir and returns a Manager.
func Setup(dir string) (*Manager, error) {
	icons := map[string]string{
		"winapps-running.svg": svgIcon("#0078D4", [4]float64{0.95, 0.85, 0.85, 0.7}),
		"winapps-stopped.svg": svgIcon("#555555", [4]float64{0.4, 0.3, 0.3, 0.2}),
	}

	// 4 starting frames: each highlights one pane clockwise
	// Pane order: 0=top-left, 1=top-right, 2=bottom-right, 3=bottom-left
	startingFrames := make([]string, 4)
	for i := 0; i < 4; i++ {
		name := fmt.Sprintf("winapps-starting-%d.svg", i)
		opacities := [4]float64{0.3, 0.3, 0.3, 0.3}
		opacities[i] = 0.95
		icons[name] = svgIcon("#0078D4", opacities)
		startingFrames[i] = fmt.Sprintf("winapps-starting-%d", i)
	}

	// 4 stopping frames: reverse order, background progressively dims
	// BL → BR → TR → TL, bg goes from #0078D4 → #3a5f8a → #555 → #444
	stoppingFrames := make([]string, 4)
	stoppingBgs := [4]string{"#0068B8", "#3a5f8a", "#555555", "#444444"}
	stoppingOrder := [4]int{3, 2, 1, 0} // BL, BR, TR, TL
	for i := 0; i < 4; i++ {
		name := fmt.Sprintf("winapps-stopping-%d.svg", i)
		opacities := [4]float64{0.25, 0.25, 0.25, 0.25}
		opacities[stoppingOrder[i]] = 0.85
		icons[name] = svgIcon(stoppingBgs[i], opacities)
		stoppingFrames[i] = fmt.Sprintf("winapps-stopping-%d", i)
	}

	for name, content := range icons {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("write icon %s: %w", name, err)
		}
	}

	return &Manager{dir: dir, startingFrames: startingFrames, stoppingFrames: stoppingFrames}, nil
}

func (m *Manager) Dir() string              { return m.dir }
func (m *Manager) RunningName() string       { return "winapps-running" }
func (m *Manager) StoppedName() string       { return "winapps-stopped" }
func (m *Manager) StartingFrames() []string  { return m.startingFrames }
func (m *Manager) StoppingFrames() []string   { return m.stoppingFrames }

// svgIcon generates an SVG string for the winapps icon.
// bgColor is the background fill. opacities are for panes:
// [0]=top-left, [1]=top-right, [2]=bottom-right, [3]=bottom-left.
func svgIcon(bgColor string, opacities [4]float64) string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">
  <rect width="64" height="64" rx="12" fill="%s"/>
  <rect x="14" y="16" width="16" height="16" rx="1" fill="#fff" opacity="%.2f"/>
  <rect x="34" y="16" width="16" height="16" rx="1" fill="#fff" opacity="%.2f"/>
  <rect x="14" y="36" width="16" height="16" rx="1" fill="#fff" opacity="%.2f"/>
  <rect x="34" y="36" width="16" height="16" rx="1" fill="#fff" opacity="%.2f"/>
</svg>`, bgColor, opacities[0], opacities[1], opacities[3], opacities[2])
}
