package main

import (
	"github.com/getlantern/systray"
	"log"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTitle("WinApps")
	systray.SetTooltip("WinApps Container Controller")
	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	go func() {
		for {
			select {
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	// Clean up here if needed
}
