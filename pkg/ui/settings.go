package ui

import (
        "fmt"
        "log"
        "os"
        "path/filepath"
        "strconv"

        "github.com/gotk3/gotk3/gtk"
        "github.com/jacklipton/winapps_systray/pkg/compose"
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

        // VM Configuration widgets
        comboRAM     *gtk.ComboBoxText
        spinCPU      *gtk.SpinButton
        entryDisk    *gtk.Entry
        entryVersion *gtk.Entry
        entryUser    *gtk.Entry
        entryPass    *gtk.Entry
        lblVMStatus  *gtk.Label

        engine          string
        composeFilePath string // full path to compose.yaml, empty if not found
        OnSave          func() // called after settings are saved, for live-reload
}
func NewSettingsWindow(settings *config.Settings, path, engine, composeFilePath string) *SettingsWindow {
        return &SettingsWindow{
                settings:        settings,
                path:            path,
                engine:          engine,
                composeFilePath: composeFilePath,
        }
}
func (s *SettingsWindow) Show() {
        if s.window != nil {
                s.window.Present()
                return
        }

        win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
        win.SetTitle("WinApps Settings")
        win.SetDefaultSize(420, 380)
        win.SetResizable(false)
        win.SetPosition(gtk.WIN_POS_CENTER)
        win.Connect("destroy", func() { s.window = nil })
        s.window = win

        notebook, _ := gtk.NotebookNew()

        // Tab 1: App Settings
        appTab := s.buildAppSettingsTab()
        appLabel, _ := gtk.LabelNew("App Settings")
        notebook.AppendPage(appTab, appLabel)

        // Tab 2: VM Configuration
        vmTab := s.buildVMConfigTab()
        vmLabel, _ := gtk.LabelNew("VM Configuration")
        notebook.AppendPage(vmTab, vmLabel)

        win.Add(notebook)
        win.ShowAll()
}

func (s *SettingsWindow) buildAppSettingsTab() *gtk.Box {
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

        lblPoll, _ := gtk.LabelNew("Poll Interval (sec)")
        lblPoll.SetHAlign(gtk.ALIGN_START)
        grid.Attach(lblPoll, 0, 0, 1, 1)

        adj, _ := gtk.AdjustmentNew(float64(s.settings.PollIntervalSeconds), 1, 60, 1, 5, 0)
        s.spinPoll, _ = gtk.SpinButtonNew(adj, 1, 0)
        grid.Attach(s.spinPoll, 1, 0, 1, 1)

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

        return mainBox
}

func (s *SettingsWindow) buildVMConfigTab() *gtk.Box {
        box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 12)
        box.SetMarginStart(18)
        box.SetMarginEnd(18)
        box.SetMarginTop(18)
        box.SetMarginBottom(18)

        // If no compose file, show disabled message
        if s.composeFilePath == "" {
                lbl, _ := gtk.LabelNew("No compose file found in WinApps directory.")
                box.PackStart(lbl, false, false, 0)
                return box
        }

        // Load current values
        service := s.settings.PrimaryService
        if service == "" {
                service = "windows"
        }
        vmCfg, err := compose.Load(s.composeFilePath, service)
        if err != nil {
                lbl, _ := gtk.LabelNew(fmt.Sprintf("Could not parse compose file:\n%v", err))
                box.PackStart(lbl, false, false, 0)
                return box
        }

        grid, _ := gtk.GridNew()
        grid.SetColumnSpacing(12)
        grid.SetRowSpacing(10)

        // RAM Size — dropdown with presets + editable
        lblRAM, _ := gtk.LabelNew("RAM Size")
        lblRAM.SetHAlign(gtk.ALIGN_START)
        grid.Attach(lblRAM, 0, 0, 1, 1)

        s.comboRAM, _ = gtk.ComboBoxTextNewWithEntry()
        for _, preset := range []string{"2G", "4G", "8G", "16G"} {
                s.comboRAM.AppendText(preset)
        }
        if vmCfg.RAMSize != "" {
                entry, _ := s.comboRAM.GetEntry()
                entry.SetText(vmCfg.RAMSize)
        }
        grid.Attach(s.comboRAM, 1, 0, 1, 1)

        // CPU Cores — spin button
        lblCPU, _ := gtk.LabelNew("CPU Cores")
        lblCPU.SetHAlign(gtk.ALIGN_START)
        grid.Attach(lblCPU, 0, 1, 1, 1)

        cpuVal := 4.0
        if vmCfg.CPUCores != "" {
                if v, err := strconv.ParseFloat(vmCfg.CPUCores, 64); err == nil {
                        cpuVal = v
                }
        }
        cpuAdj, _ := gtk.AdjustmentNew(cpuVal, 1, 64, 1, 4, 0)
        s.spinCPU, _ = gtk.SpinButtonNew(cpuAdj, 1, 0)
        grid.Attach(s.spinCPU, 1, 1, 1, 1)

        // Disk Size
        lblDisk, _ := gtk.LabelNew("Disk Size")
        lblDisk.SetHAlign(gtk.ALIGN_START)
        grid.Attach(lblDisk, 0, 2, 1, 1)

        s.entryDisk, _ = gtk.EntryNew()
        s.entryDisk.SetText(vmCfg.DiskSize)
        grid.Attach(s.entryDisk, 1, 2, 1, 1)

        // Windows Version
        lblVer, _ := gtk.LabelNew("Windows Version")
        lblVer.SetHAlign(gtk.ALIGN_START)
        grid.Attach(lblVer, 0, 3, 1, 1)

        s.entryVersion, _ = gtk.EntryNew()
        s.entryVersion.SetText(vmCfg.Version)
        grid.Attach(s.entryVersion, 1, 3, 1, 1)

        // Username
        lblUser, _ := gtk.LabelNew("Username")
        lblUser.SetHAlign(gtk.ALIGN_START)
        grid.Attach(lblUser, 0, 4, 1, 1)

        s.entryUser, _ = gtk.EntryNew()
        s.entryUser.SetText(vmCfg.Username)
        grid.Attach(s.entryUser, 1, 4, 1, 1)

        // Password
        lblPass, _ := gtk.LabelNew("Password")
        lblPass.SetHAlign(gtk.ALIGN_START)
        grid.Attach(lblPass, 0, 5, 1, 1)

        s.entryPass, _ = gtk.EntryNew()
        s.entryPass.SetText(vmCfg.Password)
        s.entryPass.SetVisibility(false)
        grid.Attach(s.entryPass, 1, 5, 1, 1)

        // Show/hide password toggle
        btnShowPass, _ := gtk.ToggleButtonNewWithLabel("Show")
        btnShowPass.Connect("toggled", func() {
                s.entryPass.SetVisibility(btnShowPass.GetActive())
        })
        grid.Attach(btnShowPass, 2, 5, 1, 1)

        box.PackStart(grid, false, false, 0)

        // Status label (for errors and success messages)
        s.lblVMStatus, _ = gtk.LabelNew("")
        s.lblVMStatus.SetHAlign(gtk.ALIGN_START)
        box.PackStart(s.lblVMStatus, false, false, 0)

        // Save button
        btnBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
        btnSave, _ := gtk.ButtonNewWithLabel("Save VM Settings")
        btnSave.SetHAlign(gtk.ALIGN_END)
        btnSave.Connect("clicked", s.onSaveVM)
        btnBox.PackEnd(btnSave, false, false, 0)
        box.PackEnd(btnBox, false, false, 0)

        return box
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
	windowsIdx := -1
	for i, svc := range services {
		s.comboService.AppendText(svc)
		if svc == s.settings.PrimaryService {
			activeIdx = i
		}
		if svc == "windows" {
			windowsIdx = i
		}
	}

	if activeIdx != -1 {
		s.comboService.SetActive(activeIdx)
	} else if windowsIdx != -1 {
		// Auto-select "windows" — the standard winapps service name
		s.comboService.SetActive(windowsIdx)
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

	func (s *SettingsWindow) onSaveVM() {
	entry, _ := s.comboRAM.GetEntry()
	ramText, _ := entry.GetText()
	diskText, _ := s.entryDisk.GetText()
	versionText, _ := s.entryVersion.GetText()
	userText, _ := s.entryUser.GetText()
	passText, _ := s.entryPass.GetText()

	vmCfg := &compose.VMConfig{
	        RAMSize:  ramText,
	        CPUCores: strconv.Itoa(int(s.spinCPU.GetValue())),
	        DiskSize: diskText,
	        Version:  versionText,
	        Username: userText,
	        Password: passText,
	}

	if err := compose.Validate(vmCfg); err != nil {
	        s.lblVMStatus.SetMarkup(fmt.Sprintf("<span foreground='#cc0000'>%s</span>", err.Error()))
	        return
	}

	service := s.settings.PrimaryService
	if service == "" {
	        service = "windows"
	}

	if err := compose.Save(s.composeFilePath, service, vmCfg); err != nil {
	        log.Printf("failed to save VM config: %v", err)
	        msg := gtk.MessageDialogNew(s.window, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_OK,
	                "Failed to save VM configuration: %v", err)
	        msg.Run()
	        msg.Destroy()
	        return
	}

	s.lblVMStatus.SetMarkup("<span foreground='#4CAF50'>Saved. Restart VM for changes to take effect.</span>")
	}

