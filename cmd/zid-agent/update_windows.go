//go:build windows

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"github.com/guilherme/zid-proxy/internal/agentui"
)

func applyUpdateWindows(fyneApp fyne.App, logMgr *agentui.LogManager, targetPath, newBinaryPath string) error {
	logMgr.Addf("Update: staging new binary for %s", targetPath)

	installDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return err
	}

	stagedPath := targetPath + ".new"
	if err := copyFile(stagedPath, newBinaryPath, 0o755); err != nil {
		return fmt.Errorf("failed to stage new binary: %w", err)
	}

	currentExe, err := os.Executable()
	if err != nil {
		return err
	}

	helperPath := filepath.Join(filepath.Dir(stagedPath), "zid-agent-updater.exe")
	if err := copyFile(helperPath, currentExe, 0o755); err != nil {
		return fmt.Errorf("failed to create updater helper: %w", err)
	}

	logMgr.Add("Update: starting updater helper...")
	cmd := exec.Command(helperPath,
		"--apply-update",
		"--update-pid", strconv.Itoa(os.Getpid()),
		"--update-target", targetPath,
		"--update-new", stagedPath,
	)
	cmd.Dir = installDir

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start helper: %w", err)
	}

	fyneApp.Driver().DoFromGoroutine(func() {
		parent := ensureDialogParent(fyneApp)
		dialog.ShowInformation("ZID Agent - Update", "Atualização iniciada. O agente será reiniciado.", parent)
	}, false)

	fyneApp.Quit()
	return nil
}

func copyFile(destPath, srcPath string, mode os.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	tmp := destPath + ".tmp"
	dst, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}

	_ = os.Remove(destPath)
	return os.Rename(tmp, destPath)
}
