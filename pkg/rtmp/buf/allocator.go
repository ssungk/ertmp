package buf

import "sync"

// Predefined buffer pool sizes.
// The maximum size (8MB) is chosen to handle 4K video I-frames,
// which typically range from 2-8MB depending on quality settings.
const (
	Size32   = 1 << 5  // 32 bytes
	Size512  = 1 << 9  // 512 bytes
	Size4K   = 1 << 12 // 4 KB
	Size16K  = 1 << 14 // 16 KB
	Size64K  = 1 << 16 // 64 KB
	Size256K = 1 << 18 // 256 KB
	Size1M   = 1 << 20 // 1 MB
	Size4M   = 1 << 22 // 4 MB
	Size8M   = 1 << 23 // 8 MB
)

// Buffer pools for different size tiers.
// Each pool manages buffers of a fixed capacity to reduce heap allocations
// and improve performance for frequently-allocated sizes.
var (
	pool32   = sync.Pool{New: func() any { return make([]byte, Size32) }}
	pool512  = sync.Pool{New: func() any { return make([]byte, Size512) }}
	pool4K   = sync.Pool{New: func() any { return make([]byte, Size4K) }}
	pool16K  = sync.Pool{New: func() any { return make([]byte, Size16K) }}
	pool64K  = sync.Pool{New: func() any { return make([]byte, Size64K) }}
	pool256K = sync.Pool{New: func() any { return make([]byte, Size256K) }}
	pool1M   = sync.Pool{New: func() any { return make([]byte, Size1M) }}
	pool4M   = sync.Pool{New: func() any { return make([]byte, Size4M) }}
	pool8M   = sync.Pool{New: func() any { return make([]byte, Size8M) }}
)

// alloc returns a buffer from pool based on size
// If size exceeds largest pool, allocates directly
func alloc(size int) []byte {
	switch {
	case size <= Size32:
		return pool32.Get().([]byte)[:size]
	case size <= Size512:
		return pool512.Get().([]byte)[:size]
	case size <= Size4K:
		return pool4K.Get().([]byte)[:size]
	case size <= Size16K:
		return pool16K.Get().([]byte)[:size]
	case size <= Size64K:
		return pool64K.Get().([]byte)[:size]
	case size <= Size256K:
		return pool256K.Get().([]byte)[:size]
	case size <= Size1M:
		return pool1M.Get().([]byte)[:size]
	case size <= Size4M:
		return pool4M.Get().([]byte)[:size]
	case size <= Size8M:
		return pool8M.Get().([]byte)[:size]
	default:
		// Size exceeds pool range, allocate directly
		return make([]byte, size)
	}
}

// free returns a buffer to the appropriate pool based on capacity
func free(buf []byte) {
	if buf == nil {
		return
	}

	capacity := cap(buf)

	switch capacity {
	case Size32:
		pool32.Put(buf[:cap(buf)])
	case Size512:
		pool512.Put(buf[:cap(buf)])
	case Size4K:
		pool4K.Put(buf[:cap(buf)])
	case Size16K:
		pool16K.Put(buf[:cap(buf)])
	case Size64K:
		pool64K.Put(buf[:cap(buf)])
	case Size256K:
		pool256K.Put(buf[:cap(buf)])
	case Size1M:
		pool1M.Put(buf[:cap(buf)])
	case Size4M:
		pool4M.Put(buf[:cap(buf)])
	case Size8M:
		pool8M.Put(buf[:cap(buf)])
	default:
		// Not from pool or oversized, let GC handle it
	}
}
