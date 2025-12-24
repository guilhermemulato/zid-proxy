package main

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/guilherme/zid-proxy/internal/agentui"
)

var (
	logsWindow      fyne.Window
	logsWindowMutex sync.Mutex
)

// showLogsWindow creates or shows the logs window.
// Only one instance of the logs window is allowed at a time.
func showLogsWindow(fyneApp fyne.App, logMgr *agentui.LogManager) {
	logsWindowMutex.Lock()
	defer logsWindowMutex.Unlock()

	// If window already exists, just show it
	if logsWindow != nil {
		logsWindow.Show()
		logsWindow.RequestFocus()
		return
	}

	// Create new window
	logsWindow = fyneApp.NewWindow("ZID Agent - Logs")

	// Create log list widget
	logList := widget.NewList(
		func() int {
			return logMgr.Size()
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			entries := logMgr.GetAll()
			if id < len(entries) {
				label := obj.(*widget.Label)
				label.SetText(entries[id].Format())
			}
		},
	)

	// Create buttons
	btnClear := widget.NewButton("Clear", func() {
		logMgr.Clear()
		logList.Refresh()
	})

	btnClose := widget.NewButton("Close", func() {
		logsWindow.Hide()
	})

	buttonBox := container.NewHBox(btnClear, btnClose)

	// Layout: list on top, buttons at bottom
	content := container.NewBorder(nil, buttonBox, nil, nil, logList)

	logsWindow.SetContent(content)
	logsWindow.Resize(fyne.NewSize(800, 500))

	logsWindow.SetCloseIntercept(func() {
		logsWindow.Hide()
	})

	// Auto-refresh when new log entries arrive
	ch := logMgr.Subscribe()
	go func() {
		for range ch {
			fyneApp.Driver().DoFromGoroutine(func() {
				if logsWindow == nil {
					return
				}
				logList.Refresh()
				if logMgr.Size() > 0 {
					logList.ScrollToBottom()
				}
			}, false)
		}
	}()

	// Handle window close
	logsWindow.SetOnClosed(func() {
		logsWindowMutex.Lock()
		defer logsWindowMutex.Unlock()
		logMgr.Unsubscribe(ch)
		logsWindow = nil
	})

	logsWindow.Show()
}
