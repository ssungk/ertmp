package buf

import (
	"sync"
	"testing"
	"unsafe"
)

func TestBufferPooled(t *testing.T) {
	// Create pooled buffer
	buf := NewPooled(1024)
	if buf.Len() != 1024 {
		t.Errorf("expected length 1024, got %d", buf.Len())
	}

	// Write some data
	copy(buf.Data(), []byte("test data"))

	// Release should return to pool
	buf.Release()
}

func TestBufferNoRelease(t *testing.T) {
	data := make([]byte, 100)
	buf := New(data)

	if buf.Len() != 100 {
		t.Errorf("expected length 100, got %d", buf.Len())
	}

	// Release should not panic
	buf.Release()
}

func TestBufferCustomRelease(t *testing.T) {
	released := false
	data := make([]byte, 100)

	buf := NewWithRelease(data, func(b []byte) {
		released = true
	})

	buf.Release()

	if !released {
		t.Error("custom release function not called")
	}
}

func TestBufferRefCount(t *testing.T) {
	released := false
	data := make([]byte, 100)

	buf := NewWithRelease(data, func(b []byte) {
		released = true
	})

	// Retain twice
	buf.Retain()
	buf.Retain()

	// Release twice - should not call release function
	buf.Release()
	buf.Release()

	if released {
		t.Error("release function called before refcount reached zero")
	}

	// Final release
	buf.Release()

	if !released {
		t.Error("release function not called after refcount reached zero")
	}
}

func TestBufferConcurrentRetainRelease(t *testing.T) {
	const goroutines = 100
	const iterations = 1000

	buf := NewPooled(1024)

	// Retain many times
	for i := 0; i < goroutines*iterations; i++ {
		buf.Retain()
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Concurrent releases
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				buf.Release()
			}
		}()
	}

	wg.Wait()

	// Final release
	buf.Release()
}

func TestPoolGetPut(t *testing.T) {
	sizes := []int{32, 512, 4096, 16384, 65536, 262144, 1048576, 4194304}

	for _, size := range sizes {
		buf := alloc(size)
		if len(buf) != size {
			t.Errorf("expected size %d, got %d", size, len(buf))
		}

		// Write pattern
		for i := 0; i < len(buf); i++ {
			buf[i] = byte(i % 256)
		}

		free(buf)

		// Get again and verify it's cleared or reused
		buf2 := alloc(size)
		if len(buf2) != size {
			t.Errorf("expected size %d, got %d", size, len(buf2))
		}
		free(buf2)
	}
}

func TestPoolOversized(t *testing.T) {
	// Request size larger than largest pool
	size := Size4M + 1024
	buf := alloc(size)

	if len(buf) != size {
		t.Errorf("expected size %d, got %d", size, len(buf))
	}

	// Put should not panic
	free(buf)
}

func BenchmarkBufferPooled(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := NewPooled(1024)
		_ = buf.Data()
		buf.Release()
	}
}

func BenchmarkBufferNoRelease(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		data := make([]byte, 1024)
		buf := New(data)
		_ = buf.Data()
		buf.Release()
	}
}

func BenchmarkBufferRetainRelease(b *testing.B) {
	buf := NewPooled(1024)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Retain()
		buf.Release()
	}
}

func BenchmarkPoolGet1K(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := alloc(1024)
		free(buf)
	}
}

func BenchmarkPoolGet64K(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := alloc(65536)
		free(buf)
	}
}

func BenchmarkPoolGet1M(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := alloc(1048576)
		free(buf)
	}
}

func BenchmarkDirectAlloc1M(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := make([]byte, 1048576)
		_ = buf
	}
}

// Test memory overhead
func TestBufferSize(t *testing.T) {
	buf := NewPooled(100)
	size := unsafe.Sizeof(*buf)
	t.Logf("Buffer struct size: %d bytes", size)
	t.Logf("  data: %d bytes", unsafe.Sizeof(buf.data))
	t.Logf("  refCount: %d bytes", unsafe.Sizeof(buf.refCount))
	t.Logf("  release: %d bytes", unsafe.Sizeof(buf.release))
	buf.Release()
}
