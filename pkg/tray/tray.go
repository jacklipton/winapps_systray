package tray

import (
	"fmt"
	"log"
	"time"
	"github.com/getlantern/systray"
	"github.com/jacklipton/winapps_systray/pkg/container"
	"github.com/jacklipton/winapps_systray/assets"
)

type TrayManager struct {
	ctrl *container.Controller
	mStatus *systray.MenuItem
	mToggle *systray.MenuItem
	mKill   *systray.MenuItem
}

func NewTrayManager(ctrl *container.Controller) *TrayManager {
	return &TrayManager{ctrl: ctrl}
}

func (t *TrayManager) Setup() {
	t.mStatus = systray.AddMenuItem("Status: Unknown", "Current container status")
	t.mStatus.Disable()
	systray.AddSeparator()
	t.mToggle = systray.AddMenuItem("Start Windows", "Toggle container state")
	t.mKill = systray.AddMenuItem("Force Kill", "Forcefully stop the container")
	t.mKill.Hide()
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit")

	go t.statusLoop()
	go t.clickLoop(mQuit)
}

func (t *TrayManager) updateUI(state container.State) {
	t.mStatus.SetTitle(fmt.Sprintf("Status: %s", state))
	t.mKill.Hide()

	switch state {
	case container.StateRunning:
		systray.SetIcon(assets.IconRunning)
		t.mToggle.SetTitle("Stop Windows")
		t.mToggle.Enable()
	case container.StateStopped:
		systray.SetIcon(assets.IconStopped)
		t.mToggle.SetTitle("Start Windows")
		t.mToggle.Enable()
	case container.StateStarting, container.StateStopping:
		systray.SetIcon(assets.IconTransition)
		t.mToggle.Disable()
		if state == container.StateStopping {
			t.mKill.Show()
		}
	}
}

func (t *TrayManager) statusLoop() {
	for {
		status, err := t.ctrl.GetStatus()
		if err != nil {
			log.Printf("status poll error: %v", err)
		}
		t.updateUI(status)
		time.Sleep(5 * time.Second)
	}
}

func (t *TrayManager) clickLoop(mQuit *systray.MenuItem) {
	for {
		select {
		case <-t.mToggle.ClickedCh:
			// Re-check state inside the goroutine so we act on the most
			// current value, not a snapshot from before the goroutine runs.
			go func() {
				status, _ := t.ctrl.GetStatus()
				if status == container.StateRunning {
					t.ctrl.Stop()
					t.ctrl.WaitUntilState(container.StateStopped, 120*time.Second)
				} else if status == container.StateStopped {
					t.ctrl.Start()
					t.ctrl.WaitUntilState(container.StateRunning, 60*time.Second)
				}
			}()
		case <-t.mKill.ClickedCh:
			t.ctrl.Kill()
		case <-mQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}
