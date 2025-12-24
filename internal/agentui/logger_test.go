package agentui

import (
	"strings"
	"testing"
	"time"
)

func TestLogManager_AddAndGet(t *testing.T) {
	lm := NewLogManager(10)

	lm.Add("first message")
	lm.Add("second message")

	entries := lm.GetAll()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].Message != "first message" {
		t.Errorf("expected 'first message', got '%s'", entries[0].Message)
	}

	if entries[1].Message != "second message" {
		t.Errorf("expected 'second message', got '%s'", entries[1].Message)
	}
}

func TestLogManager_Addf(t *testing.T) {
	lm := NewLogManager(10)

	lm.Addf("Hello %s, count: %d", "world", 42)

	entries := lm.GetAll()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	expected := "Hello world, count: 42"
	if entries[0].Message != expected {
		t.Errorf("expected '%s', got '%s'", expected, entries[0].Message)
	}
}

func TestLogManager_Format(t *testing.T) {
	entry := LogEntry{
		Timestamp: time.Date(2025, 12, 23, 10, 30, 45, 0, time.UTC),
		Message:   "test message",
	}

	formatted := entry.Format()
	expected := "2025-12-23 10:30:45 | test message"

	if formatted != expected {
		t.Errorf("expected '%s', got '%s'", expected, formatted)
	}
}

func TestLogManager_Subscribe(t *testing.T) {
	lm := NewLogManager(10)

	ch := lm.Subscribe()
	defer lm.Unsubscribe(ch)

	done := make(chan bool)
	notified := false

	go func() {
		select {
		case <-ch:
			notified = true
			done <- true
		case <-time.After(100 * time.Millisecond):
			done <- false
		}
	}()

	time.Sleep(10 * time.Millisecond) // give goroutine time to start
	lm.Add("test message")

	if success := <-done; !success {
		t.Error("subscriber was not notified")
	}

	if !notified {
		t.Error("notification was not received")
	}
}

func TestLogManager_Clear(t *testing.T) {
	lm := NewLogManager(10)

	lm.Add("message 1")
	lm.Add("message 2")

	if lm.Size() != 2 {
		t.Errorf("expected size 2, got %d", lm.Size())
	}

	lm.Clear()

	if lm.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", lm.Size())
	}

	entries := lm.GetAll()
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after clear, got %d", len(entries))
	}
}

func TestLogManager_Wraparound(t *testing.T) {
	lm := NewLogManager(3)

	lm.Add("msg1")
	lm.Add("msg2")
	lm.Add("msg3")
	lm.Add("msg4") // overwrites msg1
	lm.Add("msg5") // overwrites msg2

	entries := lm.GetAll()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	expected := []string{"msg3", "msg4", "msg5"}
	for i, exp := range expected {
		if entries[i].Message != exp {
			t.Errorf("entry %d: expected '%s', got '%s'", i, exp, entries[i].Message)
		}
	}
}

func TestLogEntry_Format_ContainsPipe(t *testing.T) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Message:   "test",
	}

	formatted := entry.Format()
	if !strings.Contains(formatted, "|") {
		t.Error("formatted entry should contain pipe separator")
	}
}
