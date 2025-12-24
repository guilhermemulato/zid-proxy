package agentui

import (
	"fmt"
	"sync"
	"time"
)

// LogEntry represents a single log message with timestamp.
type LogEntry struct {
	Timestamp time.Time
	Message   string
}

// Format returns a formatted string representation of the log entry.
func (le LogEntry) Format() string {
	return fmt.Sprintf("%s | %s",
		le.Timestamp.Format("2006-01-02 15:04:05"),
		le.Message)
}

// LogManager manages a ring buffer of log entries and notifies subscribers
// when new entries are added.
type LogManager struct {
	buffer      *RingBuffer[LogEntry]
	mu          sync.RWMutex
	subscribers []chan struct{}
	sinks       []LogSink
}

// NewLogManager creates a new log manager with the specified capacity.
func NewLogManager(capacity int) *LogManager {
	return &LogManager{
		buffer:      NewRingBuffer[LogEntry](capacity),
		subscribers: make([]chan struct{}, 0),
		sinks:       make([]LogSink, 0),
	}
}

// LogSink receives log entries for persistence or forwarding.
// Implementations must be safe for concurrent use.
type LogSink interface {
	Write(LogEntry) error
}

// AddSink registers a sink to receive future log entries.
func (lm *LogManager) AddSink(sink LogSink) {
	if sink == nil {
		return
	}

	lm.mu.Lock()
	lm.sinks = append(lm.sinks, sink)
	lm.mu.Unlock()
}

// Add adds a new log message with the current timestamp.
func (lm *LogManager) Add(message string) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Message:   message,
	}

	lm.buffer.Add(entry)
	lm.writeSinks(entry)
	lm.notifySubscribers()
}

// Addf adds a formatted log message with the current timestamp.
func (lm *LogManager) Addf(format string, args ...interface{}) {
	lm.Add(fmt.Sprintf(format, args...))
}

// GetAll returns all log entries in chronological order (oldest to newest).
func (lm *LogManager) GetAll() []LogEntry {
	return lm.buffer.GetAll()
}

// Size returns the current number of log entries.
func (lm *LogManager) Size() int {
	return lm.buffer.Size()
}

// Clear removes all log entries.
func (lm *LogManager) Clear() {
	lm.buffer.Clear()
	lm.notifySubscribers()
}

// Subscribe creates a new subscription channel that receives notifications
// when new log entries are added. The caller should receive from this channel
// in a goroutine and call Unsubscribe when done.
func (lm *LogManager) Subscribe() <-chan struct{} {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	ch := make(chan struct{}, 10) // buffered to avoid blocking
	lm.subscribers = append(lm.subscribers, ch)
	return ch
}

// Unsubscribe removes a subscription channel.
func (lm *LogManager) Unsubscribe(ch <-chan struct{}) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	for i, sub := range lm.subscribers {
		if sub == ch {
			// Close the channel and remove from list
			close(sub)
			lm.subscribers = append(lm.subscribers[:i], lm.subscribers[i+1:]...)
			return
		}
	}
}

// notifySubscribers sends a notification to all subscribers.
func (lm *LogManager) notifySubscribers() {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	for _, ch := range lm.subscribers {
		select {
		case ch <- struct{}{}:
		default:
			// Channel buffer full, skip this notification
		}
	}
}

func (lm *LogManager) writeSinks(entry LogEntry) {
	lm.mu.RLock()
	sinks := append([]LogSink(nil), lm.sinks...)
	lm.mu.RUnlock()

	for _, sink := range sinks {
		_ = sink.Write(entry)
	}
}
