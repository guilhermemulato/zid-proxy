package agentui

import "sync"

// RingBuffer is a thread-safe circular buffer that stores a fixed number of items.
// When the buffer is full, new items overwrite the oldest items.
type RingBuffer[T any] struct {
	mu       sync.RWMutex
	buffer   []T
	capacity int
	head     int  // next write position
	size     int  // current number of items
	full     bool // whether buffer has wrapped around
}

// NewRingBuffer creates a new ring buffer with the specified capacity.
func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	if capacity <= 0 {
		capacity = 100
	}
	return &RingBuffer[T]{
		buffer:   make([]T, capacity),
		capacity: capacity,
	}
}

// Add adds an item to the ring buffer.
// If the buffer is full, it overwrites the oldest item.
func (rb *RingBuffer[T]) Add(item T) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.buffer[rb.head] = item
	rb.head = (rb.head + 1) % rb.capacity

	if rb.full {
		// Buffer is full, we're overwriting old items
	} else {
		rb.size++
		if rb.head == 0 {
			rb.full = true
		}
	}
}

// GetAll returns a copy of all items in the buffer, in chronological order
// (oldest to newest).
func (rb *RingBuffer[T]) GetAll() []T {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.size == 0 {
		return []T{}
	}

	result := make([]T, rb.size)

	if rb.full {
		// Buffer has wrapped around, so head points to the oldest item
		for i := 0; i < rb.capacity; i++ {
			src := (rb.head + i) % rb.capacity
			result[i] = rb.buffer[src]
		}
	} else {
		// Buffer hasn't wrapped, items are from 0 to head-1
		copy(result, rb.buffer[:rb.size])
	}

	return result
}

// Size returns the current number of items in the buffer.
func (rb *RingBuffer[T]) Size() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.size
}

// Clear removes all items from the buffer.
func (rb *RingBuffer[T]) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.head = 0
	rb.size = 0
	rb.full = false
	rb.buffer = make([]T, rb.capacity)
}
