package buf

import (
	"testing"
)

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
	size := Size8M + 1024
	buf := alloc(size)

	if len(buf) != size {
		t.Errorf("expected size %d, got %d", size, len(buf))
	}

	// Put should not panic
	free(buf)
}

func TestAllPoolSizes(t *testing.T) {
	// Test all pool sizes to ensure full coverage
	sizes := []int{
		Size32,   // 32B
		Size512,  // 512B
		Size4K,   // 4KB
		Size16K,  // 16KB
		Size64K,  // 64KB
		Size256K, // 256KB
		Size1M,   // 1MB
		Size4M,   // 4MB
		Size8M,   // 8MB
	}

	for _, size := range sizes {
		t.Run(string(rune(size)), func(t *testing.T) {
			// Allocate exact pool size
			buf := alloc(size)
			if len(buf) != size {
				t.Errorf("expected size %d, got %d", size, len(buf))
			}
			if cap(buf) != size {
				t.Errorf("expected capacity %d, got %d", size, cap(buf))
			}

			// Write some data
			for i := 0; i < min(len(buf), 100); i++ {
				buf[i] = byte(i)
			}

			// Return to pool
			free(buf)

			// Allocate slightly smaller size (should use same pool)
			if size > 1 {
				smallerSize := size - 1
				buf2 := alloc(smallerSize)
				if len(buf2) != smallerSize {
					t.Errorf("expected size %d, got %d", smallerSize, len(buf2))
				}
				if cap(buf2) != size {
					t.Errorf("expected capacity %d (pool size), got %d", size, cap(buf2))
				}
				free(buf2)
			}
		})
	}
}

func TestPoolNilBuffer(t *testing.T) {
	// free(nil) should not panic
	free(nil)
}

func TestPoolIntermediateSizes(t *testing.T) {
	// Test sizes between pool boundaries
	testCases := []struct {
		size         int
		expectedPool int
	}{
		{1, Size32},
		{16, Size32},
		{32, Size32},
		{33, Size512},
		{256, Size512},
		{512, Size512},
		{513, Size4K},
		{2048, Size4K},
		{4096, Size4K},
		{4097, Size16K},
		{8192, Size16K},
		{16384, Size16K},
		{16385, Size64K},
		{32768, Size64K},
		{65536, Size64K},
		{65537, Size256K},
		{131072, Size256K},
		{262144, Size256K},
		{262145, Size1M},
		{524288, Size1M},
		{1048576, Size1M},
		{1048577, Size4M},
		{2097152, Size4M},
		{4194304, Size4M},
		{4194305, Size8M},
		{8388608, Size8M},
	}

	for _, tc := range testCases {
		t.Run(string(rune(tc.size)), func(t *testing.T) {
			buf := alloc(tc.size)
			if len(buf) != tc.size {
				t.Errorf("size=%d: expected len %d, got %d", tc.size, tc.size, len(buf))
			}
			if cap(buf) != tc.expectedPool {
				t.Errorf("size=%d: expected cap %d, got %d", tc.size, tc.expectedPool, cap(buf))
			}
			free(buf)
		})
	}
}

func TestPoolNonStandardCapacity(t *testing.T) {
	// Allocate a buffer with non-standard capacity
	customBuf := make([]byte, 1000, 1500) // cap=1500 doesn't match any pool size

	// free should handle it gracefully (GC will collect it)
	free(customBuf)
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

func BenchmarkPoolGet4M(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := alloc(4194304)
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

func BenchmarkDirectAlloc4M(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := make([]byte, 4194304)
		_ = buf
	}
}
