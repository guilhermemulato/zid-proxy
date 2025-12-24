//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/guilherme/zid-proxy/internal/agentui"
)

func runApplyUpdateMode(logMgr *agentui.LogManager, pid int, targetPath, newPath string) error {
	if pid <= 0 || targetPath == "" || newPath == "" {
		return errors.New("missing required flags (update-pid/update-target/update-new)")
	}

	logMgr.Addf("Updater: waiting for pid %d to exit...", pid)

	h, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(pid))
	if err == nil {
		_, _ = syscall.WaitForSingleObject(h, uint32((5*time.Minute)/time.Millisecond))
		_ = syscall.CloseHandle(h)
	} else {
		logMgr.Addf("Updater: could not open process handle: %v (continuing)", err)
		time.Sleep(3 * time.Second)
	}

	logMgr.Add("Updater: swapping binaries...")

	if err := swapFiles(targetPath, newPath); err != nil {
		return err
	}

	logMgr.Add("Updater: starting updated agent...")
	cmd := exec.Command(targetPath)
	cmd.Dir = filepath.Dir(targetPath)
	return cmd.Start()
}

func swapFiles(targetPath, newPath string) error {
	oldPath := targetPath + ".old"
	_ = os.Remove(oldPath)

	if fileExists(targetPath) {
		_ = os.Rename(targetPath, oldPath)
	}

	_ = os.Remove(targetPath)
	if err := os.Rename(newPath, targetPath); err != nil {
		return fmt.Errorf("rename new -> target failed: %w", err)
	}

	return nil
}
