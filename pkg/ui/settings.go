package ui

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gotk3/gotk3/gtk"
	"github.com/jacklipton/winapps_systray/pkg/config"
	"github.com/jacklipton/winapps_systray/pkg/discovery"
)

type SettingsWindow struct {
	window   *gtk.Window
	settings *config.Settings
	path     string // path to settings.json

	// Widgets
	fileChooser  *gtk.FileChooserButton
	comboService *gtk.ComboBoxText
	spinPoll     *gtk.SpinButton
	chkNotify    *gtk.CheckButton
	lblDirStatus *gtk.Label

	engine string
	OnSave func() // called after settings are saved, for live-reload
}

func NewSettingsWindow(settings *config.Settings, path, engine string) *SettingsWindow {
	return &SettingsWindow{
		settings: settings,
		path:     path,
		engine:   engine,
	}
}

func (s *SettingsWindow) Show() {
	if s.window != nil {
		s.window.Present()
		return
	}

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("WinApps Settings")
	win.SetDefaultSize(400, 300)
	win.SetResizable(false)
	win.SetPosition(gtk.WIN_POS_CENTER)
	win.Connect("destroy", func() { s.window = nil })
	s.window = win

	mainBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 12)
	mainBox.SetMarginStart(18)
	mainBox.SetMarginEnd(18)
	mainBox.SetMarginTop(18)
	mainBox.SetMarginBottom(18)

	// --- WinApps Directory ---
	dirBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 4)
	lblDir, _ := gtk.LabelNew("WinApps Directory")
	lblDir.SetHAlign(gtk.ALIGN_START)
	dirBox.PackStart(lblDir, false, false, 0)

	s.fileChooser, _ = gtk.FileChooserButtonNew("Select WinApps Directory", gtk.FILE_CHOOSER_ACTION_SELECT_FOLDER)
	if s.settings.WinAppsDir != "" {
		s.fileChooser.SetFilename(s.settings.WinAppsDir)
	}
	s.fileChooser.Connect("file-set", s.onDirChanged)
	dirBox.PackStart(s.fileChooser, false, false, 0)

	s.lblDirStatus, _ = gtk.LabelNew("")
	s.lblDirStatus.SetHAlign(gtk.ALIGN_START)
	dirBox.PackStart(s.lblDirStatus, false, false, 0)
	s.validateDir()

	mainBox.PackStart(dirBox, false, false, 0)

	// --- Primary Service ---
	svcBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 4)
	lblSvc, _ := gtk.LabelNew("Primary Service")
	lblSvc.SetHAlign(gtk.ALIGN_START)
	svcBox.PackStart(lblSvc, false, false, 0)

	s.comboService, _ = gtk.ComboBoxTextNew()
	s.updateServices()
	svcBox.PackStart(s.comboService, false, false, 0)
	mainBox.PackStart(svcBox, false, false, 0)

	// --- Other Settings ---
	grid, _ := gtk.GridNew()
	grid.SetColumnSpacing(12)
	grid.SetRowSpacing(12)

	// Poll Interval
	lblPoll, _ := gtk.LabelNew("Poll Interval (sec)")
	lblPoll.SetHAlign(gtk.ALIGN_START)
	grid.Attach(lblPoll, 0, 0, 1, 1)

	adj, _ := gtk.AdjustmentNew(float64(s.settings.PollIntervalSeconds), 1, 60, 1, 5, 0)
	s.spinPoll, _ = gtk.SpinButtonNew(adj, 1, 0)
	grid.Attach(s.spinPoll, 1, 0, 1, 1)

	// Notifications
	s.chkNotify, _ = gtk.CheckButtonNewWithLabel("Enable Desktop Notifications")
	s.chkNotify.SetActive(s.settings.Notifications)
	grid.Attach(s.chkNotify, 0, 1, 2, 1)

	mainBox.PackStart(grid, false, false, 0)

	// --- Save Button ---
	btnBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	btnSave, _ := gtk.ButtonNewWithLabel("Save Settings")
	btnSave.SetHAlign(gtk.ALIGN_END)
	btnSave.Connect("clicked", s.onSave)
	btnBox.PackEnd(btnSave, false, false, 0)
	mainBox.PackEnd(btnBox, false, false, 0)

	win.Add(mainBox)
	win.ShowAll()
}

func (s *SettingsWindow) onDirChanged() {
	newDir := s.fileChooser.GetFilename()
	s.settings.WinAppsDir = newDir
	s.validateDir()
	s.updateServices()
}

func (s *SettingsWindow) validateDir() {
	dir := s.settings.WinAppsDir
	if dir == "" {
		s.lblDirStatus.SetMarkup("<small>No directory selected</small>")
		return
	}
	// Check for compose file
	for _, name := range []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			s.lblDirStatus.SetMarkup(fmt.Sprintf("<small>Found %s</small>", name))
			return
		}
	}
	s.lblDirStatus.SetMarkup("<small><span foreground='#cc0000'>No compose file found in this directory</span></small>")
}

func (s *SettingsWindow) updateServices() {
	if s.settings.WinAppsDir == "" {
		return
	}

	s.comboService.RemoveAll()
	services, err := discovery.ListServices(s.settings.WinAppsDir, s.engine)
	if err != nil {
		log.Printf("failed to list services: %v", err)
		return
	}

	activeIdx := -1
	for i, svc := range services {
		s.comboService.AppendText(svc)
		if svc == s.settings.PrimaryService {
			activeIdx = i
		}
	}

	if activeIdx != -1 {
		s.comboService.SetActive(activeIdx)
	} else if len(services) > 0 {
		s.comboService.SetActive(0)
	}
}

func (s *SettingsWindow) onSave() {
	s.settings.WinAppsDir = s.fileChooser.GetFilename()
	s.settings.PrimaryService = s.comboService.GetActiveText()
	s.settings.PollIntervalSeconds = int(s.spinPoll.GetValue())
	s.settings.Notifications = s.chkNotify.GetActive()

	if err := s.settings.Save(s.path); err != nil {
		log.Printf("failed to save settings: %v", err)
		msg := gtk.MessageDialogNew(s.window, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_OK, "Failed to save settings: %v", err)
		msg.Run()
		msg.Destroy()
		return
	}

	if s.OnSave != nil {
		s.OnSave()
	}

	msg := gtk.MessageDialogNew(s.window, gtk.DIALOG_MODAL, gtk.MESSAGE_INFO, gtk.BUTTONS_OK, "%s", "Settings saved.")
	msg.Run()
	msg.Destroy()
	s.window.Close()
}
