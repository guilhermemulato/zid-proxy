//go:build !windows

package main

import (
	"fmt"

	"github.com/guilherme/zid-proxy/internal/agentui"
)

func runApplyUpdateMode(_ *agentui.LogManager, _ int, _, _ string) error {
	return fmt.Errorf("apply-update is only supported on Windows")
}
