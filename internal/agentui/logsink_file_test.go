package agentui

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileLogSink_Rotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "logs.txt")

	sink := NewFileLogSink(path, 120)
	entry := LogEntry{Timestamp: time.Date(2025, 12, 24, 10, 0, 0, 0, time.UTC), Message: "0123456789012345678901234567890123456789"}

	for i := 0; i < 20; i++ {
		if err := sink.Write(entry); err != nil {
			t.Fatalf("Write() error: %v", err)
		}
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}

	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("expected rotated file to exist: %v", err)
	}
}
