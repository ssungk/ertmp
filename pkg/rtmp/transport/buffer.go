package transport

import (
	"sync"
)

// 버퍼 크기 상수
const (
	BufferSize32  = 1 << 5  // 32B - Parsing: headers, protocol control
	BufferSize512 = 1 << 9  // 512B - Small messages: commands, audio
	BufferSize4K  = 1 << 12 // 4KB - Medium chunks, metadata
	BufferSize16K = 1 << 14 // 16KB - Video/Audio frames
	BufferSize64K = 1 << 16 // 64KB - Large chunks
)

// 크기별 버퍼 풀 ([]byte)
var (
	bufferPool32    = sync.Pool{New: func() any { return make([]byte, BufferSize32) }}
	bufferPool512   = sync.Pool{New: func() any { return make([]byte, BufferSize512) }}
	bufferPool4K    = sync.Pool{New: func() any { return make([]byte, BufferSize4K) }}
	bufferPool16K   = sync.Pool{New: func() any { return make([]byte, BufferSize16K) }}
	bufferPool64K   = sync.Pool{New: func() any { return make([]byte, BufferSize64K) }}
	bufferSlicePool = sync.Pool{New: func() any { return make([][]byte, 0, 16) }}
)

// GetBuffer returns a buffer from pool based on size
func GetBuffer(size int) []byte {
	switch {
	case size <= BufferSize32:
		return bufferPool32.Get().([]byte)[:size]
	case size <= BufferSize512:
		return bufferPool512.Get().([]byte)[:size]
	case size <= BufferSize4K:
		return bufferPool4K.Get().([]byte)[:size]
	case size <= BufferSize16K:
		return bufferPool16K.Get().([]byte)[:size]
	case size <= BufferSize64K:
		return bufferPool64K.Get().([]byte)[:size]
	default:
		return make([]byte, size)
	}
}

// PutBuffer 버퍼 반납 (capacity에 맞는 풀로 자동 반납)
func PutBuffer(buf []byte) {
	if buf == nil {
		return
	}

	capacity := cap(buf)

	switch {
	case capacity == BufferSize32:
		bufferPool32.Put(buf[:cap(buf)])
	case capacity == BufferSize512:
		bufferPool512.Put(buf[:cap(buf)])
	case capacity == BufferSize4K:
		bufferPool4K.Put(buf[:cap(buf)])
	case capacity == BufferSize16K:
		bufferPool16K.Put(buf[:cap(buf)])
	case capacity == BufferSize64K:
		bufferPool64K.Put(buf[:cap(buf)])
	default:
		// else: GC에 맡김 (표준 크기 아님)
	}
}

// GetBufferSlice gets a buffer slice from pool
func GetBufferSlice() [][]byte {
	return bufferSlicePool.Get().([][]byte)
}

// PutBufferSlice returns buffers and slice to pool
func PutBufferSlice(slice [][]byte) {
	// Return all buffers to pool
	for _, buf := range slice {
		PutBuffer(buf)
	}
	// Clear and return slice to pool
	slice = slice[:0]
	bufferSlicePool.Put(slice)
}
