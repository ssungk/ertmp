package buf

import "sync/atomic"

// Buffer represents a reference-counted buffer with custom release function
type Buffer struct {
	data     []byte
	refCount *atomic.Int32
	release  func([]byte)
}

// New creates a buffer without release function (GC managed)
func New(data []byte) *Buffer {
	refCount := &atomic.Int32{}
	refCount.Store(1)

	return &Buffer{
		data:     data,
		refCount: refCount,
		release:  nil,
	}
}

// NewPooled creates a buffer from pool
func NewPooled(size int) *Buffer {
	data := alloc(size)
	return NewWithRelease(data, free)
}

// NewWithRelease creates a buffer with custom release function
func NewWithRelease(data []byte, release func([]byte)) *Buffer {
	refCount := &atomic.Int32{}
	refCount.Store(1)

	return &Buffer{
		data:     data,
		refCount: refCount,
		release:  release,
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

// Release decrements the reference count and calls release function when it reaches zero
func (b *Buffer) Release() {
	if b.refCount == nil {
		return
	}

	count := b.refCount.Add(-1)
	if count == 0 && b.release != nil {
		b.release(b.data)
	}
}
