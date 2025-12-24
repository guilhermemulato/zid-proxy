package agentui

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const DefaultMaxLogBytes = 1 * 1024 * 1024

type FileLogSink struct {
	mu      sync.Mutex
	path    string
	maxSize int64
}

func NewFileLogSink(path string, maxSizeBytes int64) *FileLogSink {
	if maxSizeBytes <= 0 {
		maxSizeBytes = DefaultMaxLogBytes
	}
	return &FileLogSink{
		path:    path,
		maxSize: maxSizeBytes,
	}
}

func (s *FileLogSink) Write(entry LogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	if err := s.rotateIfNeeded(); err != nil {
		return err
	}

	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintln(f, entry.Format())
	return err
}

func (s *FileLogSink) rotateIfNeeded() error {
	info, err := os.Stat(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Size() < s.maxSize {
		return nil
	}

	rotated := s.path + ".1"
	_ = os.Remove(rotated)
	if err := os.Rename(s.path, rotated); err != nil {
		return err
	}
	return nil
}
