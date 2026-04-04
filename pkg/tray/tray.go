package tray

import (
	"fmt"
	"time"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/jacklipton/winapps_systray/pkg/config"
	"github.com/jacklipton/winapps_systray/pkg/container"
	"github.com/jacklipton/winapps_systray/pkg/icons"
	"github.com/jacklipton/winapps_systray/pkg/indicator"
	"github.com/jacklipton/winapps_systray/pkg/notify"
	"github.com/jacklipton/winapps_systray/pkg/ui"
)

// OnDashboard is called when the user clicks "Details...".
// Set by main before calling Setup.
type OnDashboardFunc func()

type TrayManager struct {
	ctrl         *container.Controller
	cfg          *config.Settings
	settingsPath string
	iconMgr      *icons.Manager
	ind          *indicator.Indicator

	// Menu items (need references for dynamic updates)
	mStatus  *gtk.MenuItem
	mUptime  *gtk.MenuItem
	mMemory  *gtk.MenuItem
	mEngine  *gtk.MenuItem
	mToggle  *gtk.MenuItem
	mKill    *gtk.MenuItem
	mDetails *gtk.MenuItem
	mSettings *gtk.MenuItem

	lastState container.State
	startedAt time.Time
	animFrame int
	animTimer glib.SourceHandle
	pollTimer glib.SourceHandle

	OnDashboard OnDashboardFunc
	Dashboard   *ui.Dashboard
}

func NewTrayManager(ctrl *container.Controller, cfg *config.Settings, settingsPath string, iconMgr *icons.Manager) *TrayManager {
	return &TrayManager{
		ctrl:         ctrl,
		cfg:          cfg,
		settingsPath: settingsPath,
		iconMgr:      iconMgr,
	}
}

func (t *TrayManager) Setup() {
	// Load CSS for status header coloring
	loadCSS()

	// Build GTK menu
	menu, _ := gtk.MenuNew()

	// Refresh stats every time the menu is shown
	menu.Connect("show", func() { go t.pollAndUpdate() })

	t.mStatus = addMenuItem(menu, "WinApps — Unknown", nil)
	t.mStatus.SetSensitive(false)

	addSeparator(menu)

	t.mUptime = addMenuItem(menu, "Uptime        —", nil)
	t.mUptime.SetSensitive(false)
	t.mMemory = addMenuItem(menu, "Memory        —", nil)
	t.mMemory.SetSensitive(false)
	t.mEngine = addMenuItem(menu, fmt.Sprintf("Engine        %s", t.ctrl.Engine()), nil)
	t.mEngine.SetSensitive(false)

	addSeparator(menu)

	t.mToggle = addMenuItem(menu, "Start Windows", func() { go t.onToggle() })
	t.mKill = addMenuItem(menu, "Force Kill", func() { go t.ctrl.Kill() })
	t.mKill.SetSensitive(false)

	addSeparator(menu)

	t.mDetails = addMenuItem(menu, "Details...", func() {
		if t.OnDashboard != nil {
			t.OnDashboard()
		}
	})

	t.mSettings = addMenuItem(menu, "Settings...", func() {
		sw := ui.NewSettingsWindow(t.cfg, t.settingsPath, t.ctrl.Engine())
		sw.OnSave = func() { t.restartPollTimer() }
		sw.Show()
	})

	addMenuItem(menu, "Quit", func() { gtk.MainQuit() })

	menu.ShowAll()

	// Create AppIndicator
	t.ind = indicator.New("winapps-systray", t.iconMgr.StoppedName(), t.iconMgr.Dir())
	t.ind.SetMenu(menu.Native())

	// Start status polling via GTK timer
	t.restartPollTimer()
}

func (t *TrayManager) restartPollTimer() {
	if t.pollTimer != 0 {
		glib.SourceRemove(t.pollTimer)
	}
	t.pollTimer = glib.TimeoutAdd(uint(t.cfg.PollIntervalSeconds*1000), func() bool {
		go t.pollAndUpdate()
		return true
	})
}

func (t *TrayManager) pollAndUpdate() {
	status, err := t.ctrl.GetStatus()
	if err != nil {
		return
	}

	var stats *container.Stats
	if status == container.StateRunning {
		stats = t.ctrl.GetStats()
	}

	glib.IdleAdd(func() {
		prev := t.lastState
		t.lastState = status
		t.updateUI(status, stats)

		// Send notifications on state transitions
		if t.cfg.Notifications && prev != "" && prev != status {
			t.notifyTransition(prev, status)
		}

		// Track uptime start — use real container start time when possible
		if status == container.StateRunning && prev != container.StateRunning {
			if startTime, err := t.ctrl.GetStartTime(); err == nil {
				t.startedAt = startTime
			} else {
				t.startedAt = time.Now()
			}
			if t.Dashboard != nil {
				t.Dashboard.SetStartedAt(t.startedAt)
			}
		}
	})
}

func (t *TrayManager) updateUI(state container.State, stats *container.Stats) {
	// Stop any running animation
	t.stopAnimation()

	switch state {
	case container.StateRunning:
		t.ind.SetIcon(t.iconMgr.RunningName())
		t.mStatus.SetLabel("● WinApps — Running")
		t.setStatusClass("status-running")
		t.mToggle.SetLabel("Stop Windows")
		t.mToggle.SetSensitive(true)
		t.mKill.SetSensitive(false)
		t.mDetails.SetSensitive(true)

		// Update stats
		if stats != nil {
			elapsed := time.Since(t.startedAt)
			t.mUptime.SetLabel(fmt.Sprintf("Uptime        %s", formatDuration(elapsed)))
			t.mMemory.SetLabel(fmt.Sprintf("Memory        %s", stats.MemUsage))
		}

	case container.StateStopped:
		t.ind.SetIcon(t.iconMgr.StoppedName())
		t.mStatus.SetLabel("● WinApps — Stopped")
		t.setStatusClass("status-stopped")
		t.mToggle.SetLabel("Start Windows")
		t.mToggle.SetSensitive(true)
		t.mKill.SetSensitive(false)
		t.mDetails.SetSensitive(false)
		t.mUptime.SetLabel("Uptime        —")
		t.mMemory.SetLabel("Memory        —")

	case container.StateStarting:
		t.mStatus.SetLabel("● WinApps — Starting...")
		t.setStatusClass("status-transition")
		t.mToggle.SetLabel("Starting...")
		t.mToggle.SetSensitive(false)
		t.mKill.SetSensitive(false)
		t.mDetails.SetSensitive(false)
		t.startAnimation(t.iconMgr.StartingFrames())

	case container.StateStopping:
		t.mStatus.SetLabel("● WinApps — Stopping...")
		t.setStatusClass("status-transition")
		t.mToggle.SetLabel("Stopping...")
		t.mToggle.SetSensitive(false)
		t.mKill.SetSensitive(true)
		t.startAnimation(t.iconMgr.StoppingFrames())
	}
}

func (t *TrayManager) startAnimation(frames []string) {
	t.animFrame = 0
	t.animTimer = glib.TimeoutAdd(150, func() bool {
		t.ind.SetIcon(frames[t.animFrame%len(frames)])
		t.animFrame++
		return true
	})
}

func (t *TrayManager) stopAnimation() {
	if t.animTimer != 0 {
		glib.SourceRemove(t.animTimer)
		t.animTimer = 0
	}
}

func (t *TrayManager) onToggle() {
	status, _ := t.ctrl.GetStatus()
	iconPath := t.iconMgr.Dir() + "/winapps-running.svg"
	stoppedIcon := t.iconMgr.Dir() + "/winapps-stopped.svg"

	if status == container.StateRunning {
		glib.IdleAdd(func() bool { t.updateUI(container.StateStopping, nil); return false })
		if err := t.ctrl.Stop(); err != nil && t.cfg.Notifications {
			notify.Send("WinApps", fmt.Sprintf("Failed to stop Windows VM: %v", err), stoppedIcon)
			return
		}
		if err := t.ctrl.WaitUntilState(container.StateStopped, time.Duration(t.cfg.StopTimeoutSeconds)*time.Second); err != nil && t.cfg.Notifications {
			notify.Send("WinApps", "Windows VM is taking longer than expected to stop", stoppedIcon)
		}
	} else if status == container.StateStopped {
		glib.IdleAdd(func() bool { t.updateUI(container.StateStarting, nil); return false })
		if err := t.ctrl.Start(); err != nil && t.cfg.Notifications {
			notify.Send("WinApps", fmt.Sprintf("Failed to start Windows VM: %v", err), iconPath)
			return
		}
		if err := t.ctrl.WaitUntilState(container.StateRunning, time.Duration(t.cfg.StartTimeoutSeconds)*time.Second); err != nil && t.cfg.Notifications {
			notify.Send("WinApps", "Windows VM is taking longer than expected to start", iconPath)
		}
	}
}

func (t *TrayManager) notifyTransition(prev, curr container.State) {
	iconPath := ""
	running := t.iconMgr.Dir() + "/winapps-running.svg"
	stopped := t.iconMgr.Dir() + "/winapps-stopped.svg"

	switch curr {
	case container.StateRunning:
		iconPath = running
		notify.Send("WinApps", "Windows VM is now running", iconPath)
	case container.StateStopped:
		if prev == container.StateRunning || prev == container.StateStopping {
			iconPath = stopped
			notify.Send("WinApps", "Windows VM has stopped", iconPath)
		}
	}
}

func addMenuItem(menu *gtk.Menu, label string, onClick func()) *gtk.MenuItem {
	item, _ := gtk.MenuItemNewWithLabel(label)
	if onClick != nil {
		item.Connect("activate", onClick)
	}
	item.Show()
	menu.Append(item)
	return item
}

func addSeparator(menu *gtk.Menu) {
	sep, _ := gtk.SeparatorMenuItemNew()
	sep.Show()
	menu.Append(sep)
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%ds", int(d.Seconds())%60)
}

// loadCSS injects CSS for the status header background coloring.
func loadCSS() {
	css, _ := gtk.CssProviderNew()
	css.LoadFromData(`
		.status-running { background-color: rgba(76, 175, 80, 0.15); }
		.status-stopped { background-color: rgba(158, 158, 158, 0.15); }
		.status-transition { background-color: rgba(255, 152, 0, 0.15); }
	`)
	screen, _ := gdk.ScreenGetDefault()
	gtk.AddProviderForScreen(screen, css, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}

// setStatusClass updates the CSS class on the status menu item.
func (t *TrayManager) setStatusClass(class string) {
	ctx, _ := t.mStatus.GetStyleContext()
	ctx.RemoveClass("status-running")
	ctx.RemoveClass("status-stopped")
	ctx.RemoveClass("status-transition")
	ctx.AddClass(class)
}
