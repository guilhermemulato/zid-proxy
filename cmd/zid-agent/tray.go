package main

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/systray"
	"github.com/guilherme/zid-proxy/internal/agentui"
)

func setupSystemTray(deskApp desktop.App, fyneApp fyne.App, logMgr *agentui.LogManager, statusMgr *agentui.StatusManager, cfgMgr *agentui.ConfigManager, cancel context.CancelFunc, version, buildTime string) {
	titleItem := fyne.NewMenuItem(fmt.Sprintf("ZID Agent v%s", version), nil)
	titleItem.Disabled = true

	statusItem := fyne.NewMenuItem("Status: initializing...", nil)
	statusItem.Disabled = true

	showLogsItem := fyne.NewMenuItem("Logs", func() {
		logMgr.Add("Opening logs window...")
		showLogsWindow(fyneApp, logMgr)
	})

	settingsItem := fyne.NewMenuItem("Settings", func() {
		logMgr.Add("Opening settings window...")
		showSettingsWindow(fyneApp, logMgr, cfgMgr)
	})

	updateItem := fyne.NewMenuItem("Update", func() {
		logMgr.Add("Starting update flow...")
		startUpdateFlow(fyneApp, logMgr, version)
	})

	aboutItem := fyne.NewMenuItem("About", func() {
		logMgr.Add("Opening about window...")
		showAboutWindow(fyneApp, statusMgr, cfgMgr, version, buildTime)
	})

	exitItem := fyne.NewMenuItem("Sair", func() {
		logMgr.Add("Shutting down by user request...")
		cancel()
		fyneApp.Quit()
	})

	menu := fyne.NewMenu(
		"ZID Agent",
		titleItem,
		statusItem,
		fyne.NewMenuItemSeparator(),
		showLogsItem,
		settingsItem,
		updateItem,
		aboutItem,
		fyne.NewMenuItemSeparator(),
		exitItem,
	)

	deskApp.SetSystemTrayMenu(menu)
	deskApp.SetSystemTrayIcon(statusIconResource(statusMgr.Get()))
	setTrayTooltip(statusMgr.Get())

	logMgr.Add("System tray initialized")

	ch := statusMgr.Subscribe()
	go func() {
		for range ch {
			status := statusMgr.Get()
			fyneApp.Driver().DoFromGoroutine(func() {
				deskApp.SetSystemTrayIcon(statusIconResource(status))
				statusItem.Label = "Status: " + formatStatusLine(status)
				deskApp.SetSystemTrayMenu(menu)
			}, false)
			setTrayTooltip(status)
		}
	}()
}

func setTrayTooltip(status agentui.HeartbeatStatus) {
	tooltip := "ZID Agent"
	if line := formatStatusLine(status); line != "" {
		tooltip = "ZID Agent - " + line
	}
	systray.SetTooltip(tooltip)
}

func formatStatusLine(status agentui.HeartbeatStatus) string {
	ts := status.Timestamp.Format("2006-01-02 15:04:05")
	switch status.State {
	case agentui.HeartbeatOK:
		if status.Target != "" {
			return fmt.Sprintf("OK (%s) @ %s", status.Target, ts)
		}
		return fmt.Sprintf("OK @ %s", ts)
	case agentui.HeartbeatFail:
		if status.Target != "" && status.Message != "" {
			return fmt.Sprintf("FAIL (%s): %s @ %s", status.Target, status.Message, ts)
		}
		if status.Message != "" {
			return fmt.Sprintf("FAIL: %s @ %s", status.Message, ts)
		}
		return fmt.Sprintf("FAIL @ %s", ts)
	default:
		return fmt.Sprintf("UNKNOWN @ %s", ts)
	}
}
