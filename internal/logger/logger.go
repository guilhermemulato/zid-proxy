package logger

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Action represents the action taken on a connection
type Action string

const (
	ActionAllow Action = "ALLOW"
	ActionBlock Action = "BLOCK"
)

// Entry represents a single log entry
type Entry struct {
	Timestamp time.Time
	SourceIP  string
	Hostname  string
	Group     string
	Action    Action
	Machine   string
	Username  string
	App       string // Detected application (from AppID)
}

// Logger handles structured logging to a file
type Logger struct {
	mu       sync.Mutex
	file     *os.File
	writer   *bufio.Writer
	filePath string
}

// New creates a new Logger that writes to the specified file
func New(filePath string) (*Logger, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &Logger{
		file:     file,
		writer:   bufio.NewWriter(file),
		filePath: filePath,
	}, nil
}

// Log writes a log entry
func (l *Logger) Log(entry Entry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Format: TIMESTAMP | SOURCE_IP | HOSTNAME | GROUP | ACTION | MACHINE | USER | APP
	line := fmt.Sprintf("%s | %s | %s | %s | %s | %s | %s | %s\n",
		entry.Timestamp.Format(time.RFC3339),
		entry.SourceIP,
		entry.Hostname,
		entry.Group,
		entry.Action,
		entry.Machine,
		entry.Username,
		entry.App,
	)

	l.writer.WriteString(line)
}

// LogConnection is a convenience method to log a connection
func (l *Logger) LogConnection(srcIP, hostname, group, machine, username, app string, action Action) {
	l.Log(Entry{
		Timestamp: time.Now(),
		SourceIP:  srcIP,
		Hostname:  hostname,
		Group:     group,
		Action:    action,
		Machine:   machine,
		Username:  username,
		App:       app,
	})
}

// Flush flushes any buffered data to the file
func (l *Logger) Flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush log buffer: %w", err)
	}
	return nil
}

// Close flushes and closes the log file
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush log buffer: %w", err)
	}

	if err := l.file.Close(); err != nil {
		return fmt.Errorf("failed to close log file: %w", err)
	}

	return nil
}

// Reopen closes and reopens the log file (useful for log rotation)
func (l *Logger) Reopen() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Flush current buffer
	if err := l.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush log buffer: %w", err)
	}

	// Close current file
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("failed to close log file: %w", err)
	}

	// Reopen file
	file, err := os.OpenFile(l.filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to reopen log file: %w", err)
	}

	l.file = file
	l.writer = bufio.NewWriter(file)

	return nil
}

// NullLogger is a logger that discards all output
type NullLogger struct{}

// NewNullLogger creates a logger that discards all output
func NewNullLogger() *NullLogger {
	return &NullLogger{}
}

func (l *NullLogger) Log(entry Entry) {}
func (l *NullLogger) LogConnection(srcIP, hostname, group, machine, username, app string, action Action) {
}
func (l *NullLogger) Flush() error { return nil }
func (l *NullLogger) Close() error { return nil }

// Interface defines the logger interface for dependency injection
type Interface interface {
	Log(entry Entry)
	LogConnection(srcIP, hostname, group, machine, username, app string, action Action)
	Flush() error
	Close() error
}

// Ensure Logger and NullLogger implement Interface
var _ Interface = (*Logger)(nil)
var _ Interface = (*NullLogger)(nil)

// WriterLogger wraps an io.Writer for testing
type WriterLogger struct {
	mu     sync.Mutex
	writer io.Writer
}

// NewWriterLogger creates a logger that writes to the given io.Writer
func NewWriterLogger(w io.Writer) *WriterLogger {
	return &WriterLogger{writer: w}
}

func (l *WriterLogger) Log(entry Entry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	line := fmt.Sprintf("%s | %s | %s | %s | %s | %s | %s | %s\n",
		entry.Timestamp.Format(time.RFC3339),
		entry.SourceIP,
		entry.Hostname,
		entry.Group,
		entry.Action,
		entry.Machine,
		entry.Username,
		entry.App,
	)
	l.writer.Write([]byte(line))
}

func (l *WriterLogger) LogConnection(srcIP, hostname, group, machine, username, app string, action Action) {
	l.Log(Entry{
		Timestamp: time.Now(),
		SourceIP:  srcIP,
		Hostname:  hostname,
		Group:     group,
		Action:    action,
		Machine:   machine,
		Username:  username,
		App:       app,
	})
}

func (l *WriterLogger) Flush() error { return nil }
func (l *WriterLogger) Close() error { return nil }

var _ Interface = (*WriterLogger)(nil)
