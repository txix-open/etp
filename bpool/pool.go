// Package bpool provides a buffer pool for reusing *bytes.Buffer instances.
// It reduces memory allocations by recycling buffers.
package bpool

import (
	"bytes"
	"sync"
)

var (
	bpool = sync.Pool{New: func() any {
		return bytes.NewBuffer(make([]byte, 1024))
	}}
)

// Get returns a buffer from the pool. The buffer is reset before being returned.
// Callers must call Put when done using the buffer.
func Get() *bytes.Buffer {
	b := bpool.Get().(*bytes.Buffer)
	b.Reset()
	return b
}

// Put returns a buffer to the pool for reuse.
func Put(b *bytes.Buffer) {
	bpool.Put(b)
}
