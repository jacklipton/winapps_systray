package ui

import (
	"fmt"
	"time"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/jacklipton/winapps_systray/pkg/container"
)

// Dashboard is the GTK details window.
type Dashboard struct {
	ctrl     *container.Controller
	iconPath string // path to running SVG icon for header
	window   *gtk.Window

	// Labels for live updates
	lblStatus  *gtk.Label
	lblUptime  *gtk.Label
	lblMemory  *gtk.Label
	lblCPU     *gtk.Label
	lblName    *gtk.Label
	lblService *gtk.Label
	lblEngine  *gtk.Label
	lblUser    *gtk.Label
	lblIP      *gtk.Label
	lblCompose *gtk.Label

	btnToggle  *gtk.Button
	btnPause   *gtk.Button
	btnRestart *gtk.Button
	btnKill    *gtk.Button

	startedAt time.Time
	timerID   glib.SourceHandle
	uptimeID  glib.SourceHandle
}

func NewDashboard(ctrl *container.Controller, iconPath string) *Dashboard {
	return &Dashboard{ctrl: ctrl, iconPath: iconPath}
}

// Show creates and shows the dashboard window. If already open, presents it.
func (d *Dashboard) Show() {
	if d.window != nil {
		d.window.Present()
		return
	}

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("WinApps Details")
	win.SetDefaultSize(450, 400)
	win.SetResizable(false)
	win.Connect("destroy", func() {
		d.cleanup()
		d.window = nil
	})
	d.window = win

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

	// Header
	header := d.buildHeader()
	box.PackStart(header, false, false, 0)

	// Stats grid
	grid := d.buildStatsGrid()
	box.PackStart(grid, false, false, 0)

	// Info section
	network := d.buildInfoSection()
	box.PackStart(network, false, false, 0)

	// Action buttons
	actions := d.buildActions()
	box.PackEnd(actions, false, false, 8)

	win.Add(box)
	win.ShowAll()

	// Start live updates
	d.timerID = glib.TimeoutAdd(2500, func() bool {
		go d.refresh()
		return true
	})
	d.uptimeID = glib.TimeoutAdd(1000, func() bool {
		d.updateUptime()
		return true
	})

	go d.refresh()
}

func (d *Dashboard) buildHeader() *gtk.Box {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 12)
	box.SetMarginTop(16)
	box.SetMarginBottom(12)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)

	// App icon
	if d.iconPath != "" {
		pixbuf, err := gdk.PixbufNewFromFileAtScale(d.iconPath, 32, 32, true)
		if err == nil {
			img, _ := gtk.ImageNewFromPixbuf(pixbuf)
			box.PackStart(img, false, false, 0)
		}
	}

	// Title and status
	titleBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)
	title, _ := gtk.LabelNew("")
	title.SetMarkup("<b><big>WinApps</big></b>")
	title.SetHAlign(gtk.ALIGN_START)
	titleBox.PackStart(title, false, false, 0)

	d.lblStatus, _ = gtk.LabelNew("Unknown")
	d.lblStatus.SetHAlign(gtk.ALIGN_START)
	titleBox.PackStart(d.lblStatus, false, false, 0)
	box.PackStart(titleBox, true, true, 0)

	// Uptime
	uptimeBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)
	uptimeLabel, _ := gtk.LabelNew("")
	uptimeLabel.SetMarkup("<small>Uptime</small>")
	uptimeLabel.SetHAlign(gtk.ALIGN_END)
	uptimeBox.PackStart(uptimeLabel, false, false, 0)

	d.lblUptime, _ = gtk.LabelNew("—")
	d.lblUptime.SetHAlign(gtk.ALIGN_END)
	uptimeBox.PackStart(d.lblUptime, false, false, 0)
	box.PackEnd(uptimeBox, false, false, 0)

	return box
}

func (d *Dashboard) buildStatsGrid() *gtk.Grid {
	grid, _ := gtk.GridNew()
	grid.SetRowHomogeneous(true)
	grid.SetColumnHomogeneous(true)
	grid.SetMarginStart(20)
	grid.SetMarginEnd(20)
	grid.SetMarginBottom(8)
	grid.SetRowSpacing(8)
	grid.SetColumnSpacing(8)

	d.lblMemory = d.addStatCell(grid, "MEMORY", "—", 0, 0)
	d.lblCPU = d.addStatCell(grid, "CPU", "—", 1, 0)
	d.lblService = d.addStatCell(grid, "SERVICE", "—", 0, 1)
	d.lblName = d.addStatCell(grid, "CONTAINER", "—", 1, 1)
	d.lblEngine = d.addStatCell(grid, "ENGINE", "—", 0, 2)
	d.lblUser = d.addStatCell(grid, "RDP USER", "—", 1, 2)

	return grid
}

func (d *Dashboard) addStatCell(grid *gtk.Grid, title, value string, col, row int) *gtk.Label {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 4)
	box.SetMarginTop(8)
	box.SetMarginBottom(8)

	lbl, _ := gtk.LabelNew("")
	lbl.SetMarkup(fmt.Sprintf("<small>%s</small>", title))
	lbl.SetHAlign(gtk.ALIGN_START)
	box.PackStart(lbl, false, false, 0)

	val, _ := gtk.LabelNew(value)
	val.SetHAlign(gtk.ALIGN_START)
	box.PackStart(val, false, false, 0)

	grid.Attach(box, col, row, 1, 1)
	return val
}

func (d *Dashboard) buildInfoSection() *gtk.Box {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 4)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)
	box.SetMarginTop(8)
	box.SetMarginBottom(8)

	header, _ := gtk.LabelNew("")
	header.SetMarkup("<small>CONFIGURATION</small>")
	header.SetHAlign(gtk.ALIGN_START)
	box.PackStart(header, false, false, 4)

	composeRow, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	composeLabel, _ := gtk.LabelNew("Compose File")
	composeLabel.SetHAlign(gtk.ALIGN_START)
	composeRow.PackStart(composeLabel, true, true, 0)
	d.lblCompose, _ = gtk.LabelNew("—")
	d.lblCompose.SetHAlign(gtk.ALIGN_END)
	composeRow.PackEnd(d.lblCompose, false, false, 0)
	box.PackStart(composeRow, false, false, 0)

	ipRow, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	ipLabel, _ := gtk.LabelNew("IP Address")
	ipLabel.SetHAlign(gtk.ALIGN_START)
	ipRow.PackStart(ipLabel, true, true, 0)
	d.lblIP, _ = gtk.LabelNew("—")
	d.lblIP.SetHAlign(gtk.ALIGN_END)
	ipRow.PackEnd(d.lblIP, false, false, 0)
	box.PackStart(ipRow, false, false, 0)

	return box
}

func (d *Dashboard) buildActions() *gtk.Box {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)
	box.SetMarginBottom(14)
	box.SetHAlign(gtk.ALIGN_END)

	d.btnToggle, _ = gtk.ButtonNewWithLabel("Stop")
	d.btnToggle.Connect("clicked", func() {
		go func() {
			status, _ := d.ctrl.GetStatus()
			switch status {
			case container.StateRunning, container.StatePaused:
				_ = d.ctrl.Stop()
			case container.StateStopped:
				_ = d.ctrl.Start()
			}
		}()
	})
	box.PackStart(d.btnToggle, false, false, 0)

	d.btnPause, _ = gtk.ButtonNewWithLabel("Pause")
	d.btnPause.SetSensitive(false)
	d.btnPause.Connect("clicked", func() {
		go func() {
			status, _ := d.ctrl.GetStatus()
			switch status {
			case container.StateRunning:
				_ = d.ctrl.Pause()
			case container.StatePaused:
				_ = d.ctrl.Unpause()
			}
		}()
	})
	box.PackStart(d.btnPause, false, false, 0)

	d.btnRestart, _ = gtk.ButtonNewWithLabel("Restart")
	d.btnRestart.SetSensitive(false)
	d.btnRestart.Connect("clicked", func() { go func() { _ = d.ctrl.Restart() }() })
	box.PackStart(d.btnRestart, false, false, 0)

	d.btnKill, _ = gtk.ButtonNewWithLabel("Force Kill")
	d.btnKill.SetSensitive(false)
	d.btnKill.Connect("clicked", func() { go func() { _ = d.ctrl.Kill() }() })
	box.PackStart(d.btnKill, false, false, 0)

	return box
}

func (d *Dashboard) refresh() {
	status, _ := d.ctrl.GetStatus()
	var stats *container.Stats
	if status == container.StateRunning {
		stats = d.ctrl.GetStats()
	}

	engine := d.ctrl.Engine()
	compose := d.ctrl.ComposeFile()
	rdpUser := d.ctrl.RDPUser()

	glib.IdleAdd(func() {
		if d.window == nil {
			return
		}

		switch status {
		case container.StateRunning:
			d.lblStatus.SetText("● Running")
			d.btnToggle.SetLabel("Stop")
			d.btnToggle.SetSensitive(true)
			d.btnPause.SetLabel("Pause")
			d.btnPause.SetSensitive(true)
			d.btnRestart.SetSensitive(true)
			d.btnKill.SetSensitive(false)

			if stats != nil {
				d.lblMemory.SetMarkup(fmt.Sprintf("<b>%s</b>", stats.MemUsage))
				d.lblCPU.SetMarkup(fmt.Sprintf("<b>%.1f%%</b>", stats.CPUPercent))
				d.lblName.SetText(stats.Name)
				d.lblIP.SetText(stats.IPAddress)
			}

		case container.StatePaused:
			d.lblStatus.SetText("● Paused")
			d.btnToggle.SetLabel("Stop")
			d.btnToggle.SetSensitive(true)
			d.btnPause.SetLabel("Resume")
			d.btnPause.SetSensitive(true)
			d.btnRestart.SetSensitive(false)
			d.btnKill.SetSensitive(true)

		case container.StateStopped:
			d.lblStatus.SetText("● Stopped")
			d.lblUptime.SetText("—")
			d.lblMemory.SetMarkup("—")
			d.lblCPU.SetMarkup("—")
			d.lblIP.SetText("—")
			d.btnToggle.SetLabel("Start")
			d.btnToggle.SetSensitive(true)
			d.btnPause.SetSensitive(false)
			d.btnRestart.SetSensitive(false)
			d.btnKill.SetSensitive(false)

		case container.StateStarting:
			d.lblStatus.SetText("● Starting...")
			d.btnToggle.SetSensitive(false)
			d.btnPause.SetSensitive(false)
			d.btnRestart.SetSensitive(false)
			d.btnKill.SetSensitive(false)

		case container.StateStopping:
			d.lblStatus.SetText("● Stopping...")
			d.btnToggle.SetSensitive(false)
			d.btnPause.SetSensitive(false)
			d.btnRestart.SetSensitive(false)
			d.btnKill.SetSensitive(true)

		case container.StateError:
			d.lblStatus.SetText("● Error")
			d.lblUptime.SetText("—")
			d.lblMemory.SetMarkup("—")
			d.lblCPU.SetMarkup("—")
			d.lblIP.SetText("—")
			d.btnToggle.SetLabel("Start")
			d.btnToggle.SetSensitive(true)
			d.btnPause.SetSensitive(false)
			d.btnRestart.SetSensitive(false)
			d.btnKill.SetSensitive(true)
		}

		d.lblEngine.SetText(engine)
		d.lblService.SetText(d.ctrl.PrimaryService())
		d.lblCompose.SetText(compose)
		if rdpUser != "" {
			d.lblUser.SetText(rdpUser)
		}
	})
}

func (d *Dashboard) updateUptime() {
	status, _ := d.ctrl.GetStatus()
	if status == container.StateRunning && !d.startedAt.IsZero() {
		elapsed := time.Since(d.startedAt)
		glib.IdleAdd(func() {
			if d.window != nil {
				d.lblUptime.SetText(formatDuration(elapsed))
			}
		})
	}
}

// SetStartedAt sets the time the container started (tracked by tray manager).
func (d *Dashboard) SetStartedAt(t time.Time) {
	d.startedAt = t
}

func (d *Dashboard) cleanup() {
	if d.timerID != 0 {
		glib.SourceRemove(d.timerID)
		d.timerID = 0
	}
	if d.uptimeID != 0 {
		glib.SourceRemove(d.uptimeID)
		d.uptimeID = 0
	}
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
