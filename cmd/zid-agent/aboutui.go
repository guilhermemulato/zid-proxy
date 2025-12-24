package main

import (
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/guilherme/zid-proxy/internal/agentui"
)

var (
	aboutWindow      fyne.Window
	aboutWindowMutex sync.Mutex
)

func showAboutWindow(fyneApp fyne.App, statusMgr *agentui.StatusManager, cfgMgr *agentui.ConfigManager, version, buildTime string) {
	aboutWindowMutex.Lock()
	defer aboutWindowMutex.Unlock()

	if aboutWindow != nil {
		aboutWindow.Show()
		aboutWindow.RequestFocus()
		return
	}

	aboutWindow = fyneApp.NewWindow("ZID Agent - About")
	aboutWindow.Resize(fyne.NewSize(560, 240))

	labelVersion := widget.NewLabel(fmt.Sprintf("Version: %s", version))
	labelBuild := widget.NewLabel(fmt.Sprintf("Build time: %s", buildTime))
	labelConfig := widget.NewLabel(fmt.Sprintf("Config: %s", cfgMgr.Path()))

	logPath := "(unavailable)"
	if p, err := agentui.DefaultLogPath(); err == nil {
		logPath = p
	}
	labelLog := widget.NewLabel(fmt.Sprintf("Logs: %s", logPath))

	labelStatus := widget.NewLabel("")

	refresh := func() {
		status := statusMgr.Get()
		labelStatus.SetText("Last heartbeat: " + formatStatusLine(status))
	}
	refresh()

	btnClose := widget.NewButton("Close", func() { aboutWindow.Hide() })

	content := container.NewVBox(
		labelVersion,
		labelBuild,
		labelConfig,
		labelLog,
		widget.NewSeparator(),
		labelStatus,
		widget.NewSeparator(),
		btnClose,
	)
	aboutWindow.SetContent(content)

	ch := statusMgr.Subscribe()
	go func() {
		for range ch {
			fyneApp.Driver().DoFromGoroutine(func() {
				if aboutWindow == nil {
					return
				}
				refresh()
			}, false)
		}
	}()

	aboutWindow.SetCloseIntercept(func() { aboutWindow.Hide() })
	aboutWindow.SetOnClosed(func() {
		aboutWindowMutex.Lock()
		defer aboutWindowMutex.Unlock()
		statusMgr.Unsubscribe(ch)
		aboutWindow = nil
	})

	aboutWindow.Show()
}
