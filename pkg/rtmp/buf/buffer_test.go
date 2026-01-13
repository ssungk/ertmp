package buf

import (
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"
)

func TestBufferPooled(t *testing.T) {
	// Create pooled buffer
	buf := NewFromPool(1024)
	if buf.Len() != 1024 {
		t.Errorf("expected length 1024, got %d", buf.Len())
	}

	// Write some data
	copy(buf.Data(), []byte("test data"))

	// Release should return to pool
	buf.Release()
}

func TestBufferGCManaged(t *testing.T) {
	data := make([]byte, 100)
	buf := New(data)

	if buf.Len() != 100 {
		t.Errorf("expected length 100, got %d", buf.Len())
	}

	// Release should not panic (no finalizer to call)
	buf.Release()
}

func TestBufferCustomRelease(t *testing.T) {
	released := false
	data := make([]byte, 100)

	buf := NewWithFinalizer(data, func(b []byte) {
		released = true
	})

	buf.Release()

	if !released {
		t.Error("custom finalizer not called")
	}
}

func TestBufferRefCount(t *testing.T) {
	released := false
	data := make([]byte, 100)

	buf := NewWithFinalizer(data, func(b []byte) {
		released = true
	})

	// Retain twice
	buf.Retain()
	buf.Retain()

	// Release twice - should not call finalizer
	buf.Release()
	buf.Release()

	if released {
		t.Error("finalizer called before refcount reached zero")
	}

	// Final release
	buf.Release()

	if !released {
		t.Error("finalizer not called after refcount reached zero")
	}
}

func TestBufferConcurrentRetainRelease(t *testing.T) {
	const goroutines = 100
	const iterations = 1000

	buf := NewFromPool(1024)

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

func TestBufferDataAccessors(t *testing.T) {
	size := 1024
	buf := NewFromPool(size)
	defer buf.Release()

	// Test Len
	if buf.Len() != size {
		t.Errorf("expected Len=%d, got %d", size, buf.Len())
	}

	// Test Cap
	if buf.Cap() < size {
		t.Errorf("expected Cap>=%d, got %d", size, buf.Cap())
	}

	// Test Data
	data := buf.Data()
	if len(data) != size {
		t.Errorf("expected data length=%d, got %d", size, len(data))
	}

	// Verify we can write to data
	copy(data, []byte("test"))
	if string(data[:4]) != "test" {
		t.Error("data write failed")
	}
}

func TestBufferReleaseWithoutRetain(t *testing.T) {
	releaseCalled := false
	data := make([]byte, 100)

	buf := NewWithFinalizer(data, func(b []byte) {
		releaseCalled = true
	})

	// Single release should call finalizer
	buf.Release()

	if !releaseCalled {
		t.Error("finalizer should be called when refcount reaches zero")
	}
}

// Test memory overhead
func TestBufferSize(t *testing.T) {
	buf := NewFromPool(100)
	size := unsafe.Sizeof(*buf)
	t.Logf("Buffer struct size: %d bytes", size)
	t.Logf("  data: %d bytes", unsafe.Sizeof(buf.data))
	t.Logf("  refCount: %d bytes", unsafe.Sizeof(buf.refCount))
	t.Logf("  finalizer: %d bytes", unsafe.Sizeof(buf.finalizer))
	buf.Release()
}

func BenchmarkBufferPooled(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := NewFromPool(1024)
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
	buf := NewFromPool(1024)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Retain()
		buf.Release()
	}
}

func BenchmarkBufferConcurrentRetain(b *testing.B) {
	buf := NewFromPool(1024)
	defer buf.Release()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf.Retain()
			buf.Release()
		}
	})
}

func BenchmarkAtomicInt32(b *testing.B) {
	var counter atomic.Int32
	counter.Store(1)

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			counter.Add(1)
			counter.Add(-1)
		}
	})
}
