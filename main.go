package main

import (
	"log"
	"github.com/getlantern/systray"
	"github.com/jacklipton/winapps_systray/pkg/discovery"
	"github.com/jacklipton/winapps_systray/pkg/container"
	"github.com/jacklipton/winapps_systray/pkg/tray"
)

func main() {
	systray.Run(onReady, func() {})
}

func onReady() {
	cfg, err := discovery.GetConfig()
	if err != nil {
		log.Fatalf("Discovery failed: %v", err)
	}

	ctrl := container.NewController(cfg)
	tm := tray.NewTrayManager(ctrl)
	tm.Setup()
}
