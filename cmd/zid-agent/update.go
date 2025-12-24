package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"github.com/guilherme/zid-proxy/internal/agentui"
	"github.com/guilherme/zid-proxy/internal/agentupdate"
)

const (
	updateURLLinuxGUI   = "https://s3.soulsolucoes.com.br/soul/portal/zid-agent-linux-gui-latest.tar.gz"
	updateURLWindowsGUI = "https://s3.soulsolucoes.com.br/soul/portal/zid-agent-windows-gui-latest.tar.gz"
)

func startUpdateFlow(fyneApp fyne.App, logMgr *agentui.LogManager, currentVersion string) {
	parent := ensureDialogParent(fyneApp)

	dialog.NewConfirm("ZID Agent - Update", "Baixar e instalar a versão mais recente agora?\nO agente será reiniciado.", func(ok bool) {
		if !ok {
			return
		}

		progress := dialog.NewProgressInfinite("ZID Agent - Update", "Baixando bundle...", parent)
		progress.Show()

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			err := runUpdate(ctx, fyneApp, logMgr, currentVersion)
			fyneApp.Driver().DoFromGoroutine(func() {
				progress.Hide()
				if err != nil {
					logMgr.Addf("Update failed: %v", err)
					dialog.ShowError(err, parent)
				}
			}, false)
		}()
	}, parent).Show()
}

func runUpdate(ctx context.Context, fyneApp fyne.App, logMgr *agentui.LogManager, currentVersion string) error {
	url, expectedBinaryName, targetPath, err := updatePlatformConfig()
	if err != nil {
		return err
	}

	logMgr.Addf("Update: downloading %s", url)

	pu, err := agentupdate.PrepareFromURL(ctx, agentupdate.Downloader{}, url, expectedBinaryName)
	if err != nil {
		return err
	}
	defer pu.Cleanup()

	logMgr.Addf("Update: bundle version %s", pu.Version)

	if pu.Version == currentVersion {
		fyneApp.Driver().DoFromGoroutine(func() {
			parent := ensureDialogParent(fyneApp)
			dialog.ShowInformation("ZID Agent - Update", fmt.Sprintf("Você já está na versão mais recente (%s).", currentVersion), parent)
		}, false)
		return nil
	}

	switch runtime.GOOS {
	case "windows":
		return applyUpdateWindows(fyneApp, logMgr, targetPath, pu.BinaryPath)
	default:
		return applyUpdateLinux(logMgr, targetPath, pu.BinaryPath)
	}
}

func updatePlatformConfig() (url, expectedBinaryName, targetPath string, err error) {
	switch runtime.GOOS {
	case "windows":
		url = updateURLWindowsGUI
		expectedBinaryName = "zid-agent-windows-gui.exe"
		targetPath = defaultWindowsInstallPath()
		if targetPath == "" {
			return "", "", "", fmt.Errorf("could not resolve install path")
		}
		return url, expectedBinaryName, targetPath, nil
	case "linux":
		url = updateURLLinuxGUI
		expectedBinaryName = "zid-agent-linux-gui"
		targetPath = defaultLinuxInstallPath()
		if targetPath == "" {
			return "", "", "", fmt.Errorf("could not resolve install path")
		}
		return url, expectedBinaryName, targetPath, nil
	default:
		return "", "", "", fmt.Errorf("update not supported on %s", runtime.GOOS)
	}
}

func defaultLinuxInstallPath() string {
	const p = "/usr/local/bin/zid-agent"
	if fileExists(p) {
		return p
	}
	if exe, err := os.Executable(); err == nil && exe != "" {
		if real, err := filepath.EvalSymlinks(exe); err == nil && real != "" {
			return real
		}
		return exe
	}
	return ""
}

func defaultWindowsInstallPath() string {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData != "" {
		p := filepath.Join(localAppData, "ZIDAgent", "zid-agent.exe")
		if fileExists(p) {
			return p
		}
		// If not installed yet, still use the standard path (update will fail with clear error if not writable).
		return p
	}

	if exe, err := os.Executable(); err == nil && exe != "" {
		return exe
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
