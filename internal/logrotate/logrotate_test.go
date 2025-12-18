package logrotate

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRun_CreatesMissingFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "zid-proxy.log")

	rotated, err := Run(Options{
		LogPath:  logPath,
		KeepDays: 7,
		Now:      time.Date(2025, 12, 18, 10, 0, 0, 0, time.Local),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if rotated {
		t.Fatalf("Run() rotated = true, want false")
	}

	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("expected log file to exist: %v", err)
	}
}

func TestRun_NoRotateSameDay(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "zid-proxy.log")
	if err := os.WriteFile(logPath, []byte("x\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	now := time.Date(2025, 12, 18, 10, 0, 0, 0, time.Local)
	if err := os.Chtimes(logPath, now, now); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	rotated, err := Run(Options{LogPath: logPath, KeepDays: 7, Now: now})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if rotated {
		t.Fatalf("Run() rotated = true, want false")
	}

	if _, err := os.Stat(logPath + ".0"); !os.IsNotExist(err) {
		t.Fatalf("expected no rotated file, stat err=%v", err)
	}
}

func TestRun_RotatePreviousDay(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "zid-proxy.log")
	if err := os.WriteFile(logPath, []byte("day1\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	day1 := time.Date(2025, 12, 17, 23, 59, 0, 0, time.Local)
	if err := os.Chtimes(logPath, day1, day1); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	now := time.Date(2025, 12, 18, 0, 1, 0, 0, time.Local)
	rotated, err := Run(Options{LogPath: logPath, KeepDays: 3, Now: now})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !rotated {
		t.Fatalf("Run() rotated = false, want true")
	}

	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("expected new current log to exist: %v", err)
	}
	got, err := os.ReadFile(logPath + ".0")
	if err != nil {
		t.Fatalf("ReadFile .0: %v", err)
	}
	if string(got) != "day1\n" {
		t.Fatalf("rotated content = %q, want %q", string(got), "day1\n")
	}
}

func TestRun_KeepDays_ShiftsAndRemoves(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "zid-proxy.log")

	// Simulate existing rotated logs.
	if err := os.WriteFile(logPath+".0", []byte("d0\n"), 0644); err != nil {
		t.Fatalf("WriteFile .0: %v", err)
	}
	if err := os.WriteFile(logPath+".1", []byte("d1\n"), 0644); err != nil {
		t.Fatalf("WriteFile .1: %v", err)
	}
	if err := os.WriteFile(logPath+".2", []byte("d2\n"), 0644); err != nil {
		t.Fatalf("WriteFile .2: %v", err)
	}
	if err := os.WriteFile(logPath, []byte("current\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	day1 := time.Date(2025, 12, 17, 10, 0, 0, 0, time.Local)
	if err := os.Chtimes(logPath, day1, day1); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	now := time.Date(2025, 12, 18, 10, 0, 0, 0, time.Local)
	rotated, err := Run(Options{LogPath: logPath, KeepDays: 3, Now: now})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !rotated {
		t.Fatalf("Run() rotated = false, want true")
	}

	// keepDays=3 means keep .0,.1,.2. Oldest is .2; it should have been removed then shifted.
	if _, err := os.Stat(logPath + ".3"); !os.IsNotExist(err) {
		t.Fatalf("expected .3 to not exist, stat err=%v", err)
	}

	got2, err := os.ReadFile(logPath + ".2")
	if err != nil {
		t.Fatalf("ReadFile .2: %v", err)
	}
	if string(got2) != "d1\n" {
		t.Fatalf(".2 content = %q, want %q", string(got2), "d1\n")
	}

	got1, err := os.ReadFile(logPath + ".1")
	if err != nil {
		t.Fatalf("ReadFile .1: %v", err)
	}
	if string(got1) != "d0\n" {
		t.Fatalf(".1 content = %q, want %q", string(got1), "d0\n")
	}

	got0, err := os.ReadFile(logPath + ".0")
	if err != nil {
		t.Fatalf("ReadFile .0: %v", err)
	}
	if string(got0) != "current\n" {
		t.Fatalf(".0 content = %q, want %q", string(got0), "current\n")
	}
}

