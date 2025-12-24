package agentui

import (
	"testing"
)

func TestRingBuffer_Basic(t *testing.T) {
	rb := NewRingBuffer[int](3)

	if rb.Size() != 0 {
		t.Errorf("expected size 0, got %d", rb.Size())
	}

	rb.Add(1)
	rb.Add(2)
	rb.Add(3)

	if rb.Size() != 3 {
		t.Errorf("expected size 3, got %d", rb.Size())
	}

	items := rb.GetAll()
	expected := []int{1, 2, 3}
	if len(items) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(items))
	}

	for i, v := range expected {
		if items[i] != v {
			t.Errorf("item %d: expected %d, got %d", i, v, items[i])
		}
	}
}

func TestRingBuffer_Wraparound(t *testing.T) {
	rb := NewRingBuffer[int](3)

	rb.Add(1)
	rb.Add(2)
	rb.Add(3)
	rb.Add(4) // overwrites 1
	rb.Add(5) // overwrites 2

	if rb.Size() != 3 {
		t.Errorf("expected size 3, got %d", rb.Size())
	}

	items := rb.GetAll()
	expected := []int{3, 4, 5}
	if len(items) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(items))
	}

	for i, v := range expected {
		if items[i] != v {
			t.Errorf("item %d: expected %d, got %d", i, v, items[i])
		}
	}
}

func TestRingBuffer_Clear(t *testing.T) {
	rb := NewRingBuffer[int](3)

	rb.Add(1)
	rb.Add(2)
	rb.Clear()

	if rb.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", rb.Size())
	}

	items := rb.GetAll()
	if len(items) != 0 {
		t.Errorf("expected 0 items after clear, got %d", len(items))
	}
}

func TestRingBuffer_DefaultCapacity(t *testing.T) {
	rb := NewRingBuffer[int](0)
	if rb.capacity != 100 {
		t.Errorf("expected default capacity 100, got %d", rb.capacity)
	}
}
