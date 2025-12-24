package main

import (
	"sync"

	"fyne.io/fyne/v2"
)

var (
	dialogParentWindow      fyne.Window
	dialogParentWindowMutex sync.Mutex
)

func ensureDialogParent(app fyne.App) fyne.Window {
	dialogParentWindowMutex.Lock()
	defer dialogParentWindowMutex.Unlock()

	if logsWindow != nil {
		return logsWindow
	}
	if dialogParentWindow != nil {
		return dialogParentWindow
	}

	w := app.NewWindow("ZID Agent")
	w.Resize(fyne.NewSize(1, 1))
	w.Hide()
	dialogParentWindow = w
	return w
}
