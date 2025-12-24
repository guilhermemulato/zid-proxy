package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/guilherme/zid-proxy/internal/agentui"
)

var (
	settingsWindow      fyne.Window
	settingsWindowMutex sync.Mutex
)

func showSettingsWindow(fyneApp fyne.App, logMgr *agentui.LogManager, cfgMgr *agentui.ConfigManager) {
	settingsWindowMutex.Lock()
	defer settingsWindowMutex.Unlock()

	if settingsWindow != nil {
		settingsWindow.Show()
		settingsWindow.RequestFocus()
		return
	}

	settingsWindow = fyneApp.NewWindow("ZID Agent - Settings")
	settingsWindow.Resize(fyne.NewSize(520, 280))

	cfg := cfgMgr.Get()

	entryPort := widget.NewEntry()
	entryPort.SetText(strconv.Itoa(cfg.Port))

	entryDNS := widget.NewEntry()
	entryDNS.SetText(cfg.DNSFallback)

	entryInterval := widget.NewEntry()
	entryInterval.SetText(strconv.Itoa(cfg.IntervalSeconds))

	errorLabel := widget.NewLabel("")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Port", Widget: entryPort},
			{Text: "DNS fallback", Widget: entryDNS},
			{Text: "Interval (seconds)", Widget: entryInterval},
		},
		OnSubmit: func() {
			port, err := strconv.Atoi(strings.TrimSpace(entryPort.Text))
			if err != nil {
				errorLabel.SetText("Invalid port")
				return
			}
			intervalSeconds, err := strconv.Atoi(strings.TrimSpace(entryInterval.Text))
			if err != nil {
				errorLabel.SetText("Invalid interval")
				return
			}

			newCfg := agentui.Config{
				Port:            port,
				DNSFallback:     strings.TrimSpace(entryDNS.Text),
				IntervalSeconds: intervalSeconds,
			}

			if err := cfgMgr.Set(newCfg); err != nil {
				errorLabel.SetText(fmt.Sprintf("Invalid settings: %v", err))
				return
			}
			if err := cfgMgr.SaveToDisk(); err != nil {
				errorLabel.SetText(fmt.Sprintf("Could not save: %v", err))
				return
			}

			errorLabel.SetText("Saved")
			logMgr.Addf("Settings saved: port=%d dns=%s interval=%ds", newCfg.Port, newCfg.DNSFallback, newCfg.IntervalSeconds)
		},
		SubmitText: "Save",
		OnCancel: func() {
			settingsWindow.Hide()
		},
		CancelText: "Close",
	}

	pathLabel := widget.NewLabel("")
	if p := cfgMgr.Path(); p != "" {
		pathLabel.SetText("Config: " + p)
	} else {
		pathLabel.SetText("Config: (unavailable)")
	}

	content := container.NewVBox(pathLabel, form, errorLabel)
	settingsWindow.SetContent(content)

	settingsWindow.SetCloseIntercept(func() { settingsWindow.Hide() })
	settingsWindow.SetOnClosed(func() {
		settingsWindowMutex.Lock()
		defer settingsWindowMutex.Unlock()
		settingsWindow = nil
	})

	settingsWindow.Show()
}
