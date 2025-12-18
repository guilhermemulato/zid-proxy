package logrotate

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Options struct {
	LogPath  string
	KeepDays int
	Now      time.Time
}

func Run(opts Options) (bool, error) {
	if opts.LogPath == "" {
		return false, fmt.Errorf("log path is required")
	}
	if opts.KeepDays < 1 {
		return false, fmt.Errorf("keep days must be >= 1")
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}

	if err := ensureFile(opts.LogPath); err != nil {
		return false, err
	}

	info, err := os.Stat(opts.LogPath)
	if err != nil {
		return false, fmt.Errorf("stat log file: %w", err)
	}

	logDay := dayStart(info.ModTime())
	nowDay := dayStart(opts.Now)
	if logDay.Equal(nowDay) {
		return false, nil
	}

	if err := rotateNumeric(opts.LogPath, opts.KeepDays); err != nil {
		return false, err
	}

	if err := ensureFile(opts.LogPath); err != nil {
		return true, err
	}

	return true, nil
}

func dayStart(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func ensureFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("mkdir log dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	return f.Close()
}

func rotateNumeric(logPath string, keepDays int) error {
	// KeepDays=N means we keep N rotated files: .0 .. .(N-1)
	// Current file remains as logPath.
	oldest := fmt.Sprintf("%s.%d", logPath, keepDays-1)
	if err := os.Remove(oldest); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove oldest rotated log: %w", err)
	}

	for i := keepDays - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", logPath, i-1)
		dst := fmt.Sprintf("%s.%d", logPath, i)
		if err := os.Rename(src, dst); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("rename %s -> %s: %w", src, dst, err)
		}
	}

	dst := fmt.Sprintf("%s.0", logPath)
	if err := os.Rename(logPath, dst); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", logPath, dst, err)
	}

	return nil
}

