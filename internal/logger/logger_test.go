package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	logger, err := New(logFile)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log some entries
	testTime := time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC)
	logger.Log(Entry{
		Timestamp: testTime,
		SourceIP:  "192.168.1.100",
		Hostname:  "www.example.com",
		Group:     "acesso_liberado",
		Action:    ActionAllow,
	})

	logger.Log(Entry{
		Timestamp: testTime.Add(time.Second),
		SourceIP:  "192.168.1.50",
		Hostname:  "blocked.example.com",
		Group:     "acesso_restrito",
		Action:    ActionBlock,
	})

	// Flush to ensure data is written
	if err := logger.Flush(); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}

	// Read and verify log content
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 log lines, got %d", len(lines))
	}

	// Verify first line format
	if !strings.Contains(lines[0], "192.168.1.100") {
		t.Error("first line missing source IP")
	}
	if !strings.Contains(lines[0], "www.example.com") {
		t.Error("first line missing hostname")
	}
	if !strings.Contains(lines[0], "ALLOW") {
		t.Error("first line missing action")
	}

	// Verify second line
	if !strings.Contains(lines[1], "BLOCK") {
		t.Error("second line missing BLOCK action")
	}
}

func TestLoggerReopen(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	logger, err := New(logFile)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Log entry before reopen
	logger.LogConnection("10.0.0.1", "before.example.com", "g1", "pc-01", "alice", ActionAllow)
	logger.Flush()

	// Reopen (simulating log rotation)
	if err := logger.Reopen(); err != nil {
		t.Fatalf("failed to reopen: %v", err)
	}

	// Log entry after reopen
	logger.LogConnection("10.0.0.2", "after.example.com", "g2", "", "", ActionBlock)
	logger.Close()

	// Verify both entries exist
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "before.example.com") {
		t.Error("missing entry from before reopen")
	}
	if !strings.Contains(string(content), "after.example.com") {
		t.Error("missing entry from after reopen")
	}
}

func TestWriterLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWriterLogger(&buf)

	testTime := time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC)
	logger.Log(Entry{
		Timestamp: testTime,
		SourceIP:  "192.168.1.1",
		Hostname:  "test.example.com",
		Group:     "acesso_controlado",
		Action:    ActionBlock,
	})

	output := buf.String()
	if !strings.Contains(output, "2025-01-15T10:30:45Z") {
		t.Error("missing timestamp in output")
	}
	if !strings.Contains(output, "192.168.1.1") {
		t.Error("missing source IP in output")
	}
	if !strings.Contains(output, "test.example.com") {
		t.Error("missing hostname in output")
	}
	if !strings.Contains(output, "BLOCK") {
		t.Error("missing action in output")
	}
}

func TestNullLogger(t *testing.T) {
	logger := NewNullLogger()

	// These should not panic
	logger.LogConnection("192.168.1.1", "example.com", "", "", "", ActionAllow)
	logger.Log(Entry{})

	if err := logger.Flush(); err != nil {
		t.Errorf("unexpected error from Flush: %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Errorf("unexpected error from Close: %v", err)
	}
}

func BenchmarkLogger(b *testing.B) {
	tmpDir := b.TempDir()
	logFile := filepath.Join(tmpDir, "bench.log")

	logger, err := New(logFile)
	if err != nil {
		b.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	entry := Entry{
		Timestamp: time.Now(),
		SourceIP:  "192.168.1.100",
		Hostname:  "www.example.com",
		Group:     "g",
		Action:    ActionAllow,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Log(entry)
	}
}
