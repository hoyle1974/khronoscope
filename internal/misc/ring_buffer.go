package misc

import (
	"bytes"
	"strings"
	"sync"
)

// RingBuffer is a fixed-size, thread-safe ring buffer for log messages.
type RingBuffer struct {
	mu      sync.Mutex
	data    []string
	size    int
	current int
	full    bool
}

// NewRingBuffer initializes a ring buffer with the given size.
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		data: make([]string, size),
		size: size,
	}
}

// Write implements io.Writer, allowing it to be used as a log destination.
func (rb *RingBuffer) Write(p []byte) (n int, err error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	s := string(p)
	rb.data[rb.current] = strings.TrimSpace(s) // Store log entry
	rb.current = (rb.current + 1) % rb.size    // Move to next slot

	// If we've wrapped around, mark the buffer as full
	if rb.current == 0 {
		rb.full = true
	}

	return len(p), nil
}

// GetLogs returns a snapshot of all stored log messages in order.
func (rb *RingBuffer) GetLogs() []string {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if !rb.full {
		return rb.data[:rb.current]
	}

	// Return logs in order: oldest first
	return append(rb.data[rb.current:], rb.data[:rb.current]...)
}

// String returns a single string representation of the entire buffer.
func (rb *RingBuffer) String() string {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	var buffer bytes.Buffer

	if rb.full {
		for _, logEntry := range rb.data[rb.current:] {
			buffer.WriteString(logEntry + "\n")
		}
		for _, logEntry := range rb.data[:rb.current] {
			buffer.WriteString(logEntry + "\n")
		}
	} else {
		for _, logEntry := range rb.data[:rb.current] {
			buffer.WriteString(logEntry + "\n")
		}
	}

	return buffer.String()
}
