//go:build !windows

package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"github.com/guilherme/zid-proxy/internal/agentui"
)

func applyUpdateWindows(_ fyne.App, _ *agentui.LogManager, _, _ string) error {
	return fmt.Errorf("update not supported on this platform")
}
