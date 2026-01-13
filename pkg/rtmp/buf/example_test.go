package buf_test

import (
	"fmt"

	"github.com/ssungk/ertmp/pkg/rtmp/buf"
)

// Example of using pooled buffer
func ExampleNewFromPool() {
	// Get buffer from pool
	buf := buf.NewFromPool(1024)
	defer buf.Release()

	// Use buffer
	copy(buf.Data(), []byte("Hello, World!"))
	fmt.Printf("Data: %s\n", buf.Data()[:13])

	// Output: Data: Hello, World!
}

// Example of using buffer with custom finalizer
func ExampleNewWithFinalizer() {
	// Example with custom cleanup logic
	size := 1024
	data := make([]byte, size)

	// Create buffer with custom finalizer
	b := buf.NewWithFinalizer(data, func(data []byte) {
		// Custom cleanup (e.g., C.free, file close, etc.)
		fmt.Println("Custom finalizer called")
	})

	// Use buffer
	copy(b.Data(), []byte("Data with custom cleanup"))

	// Release will call custom finalizer
	b.Release()

	// Output: Custom finalizer called
}

// Example of reference counting
func ExampleBuffer_Retain() {
	buf := buf.NewFromPool(1024)

	// Share buffer
	buf.Retain() // refCount = 2

	// First release
	buf.Release() // refCount = 1

	// Buffer still valid
	copy(buf.Data(), []byte("Still valid"))
	fmt.Printf("Data: %s\n", buf.Data()[:11])

	// Final release
	buf.Release() // refCount = 0, returns to pool

	// Output: Data: Still valid
}

// Example of basic buffer (GC managed)
func ExampleNew() {
	data := make([]byte, 100)
	b := buf.New(data)

	// Use buffer
	copy(b.Data(), []byte("GC managed"))

	// Release is no-op, GC will handle it
	b.Release()

	fmt.Printf("Data: %s\n", b.Data()[:10])

	// Output: Data: GC managed
}

// Example showing different pool sizes
func Example_poolSizes() {
	sizes := []int{
		buf.Size32,   // 32B
		buf.Size512,  // 512B
		buf.Size4K,   // 4KB
		buf.Size16K,  // 16KB
		buf.Size64K,  // 64KB
		buf.Size256K, // 256KB
		buf.Size1M,   // 1MB
		buf.Size4M,   // 4MB
		buf.Size8M,   // 8MB
	}

	for _, size := range sizes {
		buf := buf.NewFromPool(size)
		fmt.Printf("Size: %7d bytes, Len: %7d, Cap: %7d\n", size, buf.Len(), buf.Cap())
		buf.Release()
	}

	// Output:
	// Size:      32 bytes, Len:      32, Cap:      32
	// Size:     512 bytes, Len:     512, Cap:     512
	// Size:    4096 bytes, Len:    4096, Cap:    4096
	// Size:   16384 bytes, Len:   16384, Cap:   16384
	// Size:   65536 bytes, Len:   65536, Cap:   65536
	// Size:  262144 bytes, Len:  262144, Cap:  262144
	// Size: 1048576 bytes, Len: 1048576, Cap: 1048576
	// Size: 4194304 bytes, Len: 4194304, Cap: 4194304
	// Size: 8388608 bytes, Len: 8388608, Cap: 8388608
}

// Example of oversized buffer (exceeds pool)
func Example_oversized() {
	// Request 10MB buffer (exceeds Size8M)
	size := 10 * 1024 * 1024
	buf := buf.NewFromPool(size)
	defer buf.Release()

	fmt.Printf("Oversized buffer: %d bytes\n", buf.Len())
	// This will be allocated directly with make() and GC collected after Release()

	// Output: Oversized buffer: 10485760 bytes
}
