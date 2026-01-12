package buf

import "sync/atomic"

// Buffer represents a reference-counted buffer with custom finalizer
type Buffer struct {
	data      []byte
	refCount  *atomic.Int32
	finalizer func([]byte)
}

// New creates a buffer without finalizer (GC managed)
func New(data []byte) *Buffer {
	return NewWithFinalizer(data, nil)
}

// NewFromPool creates a buffer from pool
func NewFromPool(size int) *Buffer {
	data := alloc(size)
	return NewWithFinalizer(data, free)
}

// NewWithFinalizer creates a buffer with custom finalizer
func NewWithFinalizer(data []byte, finalizer func([]byte)) *Buffer {
	refCount := &atomic.Int32{}
	refCount.Store(1)

	return &Buffer{
		data:      data,
		refCount:  refCount,
		finalizer: finalizer,
	}
}

// Data returns the underlying byte slice
func (b *Buffer) Data() []byte {
	return b.data
}

// Len returns the length of the buffer
func (b *Buffer) Len() int {
	return len(b.data)
}

// Cap returns the capacity of the buffer
func (b *Buffer) Cap() int {
	return cap(b.data)
}

// Retain increments the reference count
func (b *Buffer) Retain() {
	if b.refCount != nil {
		b.refCount.Add(1)
	}
}

// Release decrements the reference count and calls finalizer when it reaches zero
func (b *Buffer) Release() {
	if b.refCount == nil {
		return
	}

	count := b.refCount.Add(-1)
	if count == 0 && b.finalizer != nil {
		b.finalizer(b.data)
	}
}
