//go:build !linux

package main

import (
	"fmt"

	"github.com/guilherme/zid-proxy/internal/agentui"
)

func applyUpdateLinux(_ *agentui.LogManager, _, _ string) error {
	return fmt.Errorf("update not supported on this platform")
}
