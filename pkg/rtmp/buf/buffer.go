// Package buf provides reference-counted buffers with memory pooling.
//
// The package implements a zero-copy buffer management system optimized
// for RTMP message handling. Buffers are pooled in 9 size tiers from
// 32 bytes to 8MB to minimize heap allocations.
//
// Basic usage:
//
//	buf := buf.NewFromPool(1024)
//	defer buf.Release()
//	copy(buf.Data(), data)
//
// For shared buffers, use Retain/Release:
//
//	buf.Retain()  // Increment reference count
//	go func() {
//	    defer buf.Release()
//	    // use buf
//	}()
//	buf.Release()  // Original owner releases
//
// Buffers must be created through constructors (New, NewFromPool, NewWithFinalizer).
// Direct struct initialization will cause a panic.
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
	b.refCount.Add(1)
}

// Release decrements the reference count and calls finalizer when it reaches zero
func (b *Buffer) Release() {
	count := b.refCount.Add(-1)
	if count == 0 && b.finalizer != nil {
		b.finalizer(b.data)
	}
}
