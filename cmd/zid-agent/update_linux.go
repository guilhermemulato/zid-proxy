//go:build linux

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/guilherme/zid-proxy/internal/agentui"
)

func applyUpdateLinux(logMgr *agentui.LogManager, targetPath, newBinaryPath string) error {
	logMgr.Addf("Update: applying to %s", targetPath)

	if err := copyFileAtomic(targetPath, newBinaryPath, 0o755); err != nil {
		return fmt.Errorf("failed to write %s: %w", targetPath, err)
	}

	logMgr.Add("Update: applied successfully, restarting...")
	return syscall.Exec(targetPath, os.Args, os.Environ())
}

func copyFileAtomic(destPath, srcPath string, mode os.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}

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
	return os.Rename(tmp, destPath)
}
