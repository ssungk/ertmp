package transport

import (
	"bufio"
	"io"
)

// meteredConn wraps a connection and meters all bytes read and written
// for RTMP acknowledgement and flow control.
// Counts all bytes including RTMP protocol overhead (chunk headers, etc).
// Not thread-safe: designed for single-goroutine usage.
type meteredConn struct {
	*bufio.ReadWriter
	bytesRead    uint64
	bytesWritten uint64
}

// newMeteredConn creates a new metered connection
func newMeteredConn(rw io.ReadWriter) *meteredConn {
	return &meteredConn{
		ReadWriter: bufio.NewReadWriter(
			bufio.NewReaderSize(rw, IOBufferSize),
			bufio.NewWriterSize(rw, IOBufferSize),
		),
	}
}

// Read reads data into p and meters the bytes read.
// Only increments bytesRead when n > 0, ensuring errors don't affect count.
func (mc *meteredConn) Read(p []byte) (int, error) {
	n, err := mc.Reader.Read(p)
	if n > 0 {
		mc.bytesRead += uint64(n)
	}
	return n, err
}

// ReadByte reads a single byte and meters it
func (mc *meteredConn) ReadByte() (byte, error) {
	b, err := mc.Reader.ReadByte()
	if err == nil {
		mc.bytesRead++
	}
	return b, err
}

// Write writes data from p and meters the bytes written.
// Only increments bytesWritten when n > 0, ensuring errors don't affect count.
func (mc *meteredConn) Write(p []byte) (int, error) {
	n, err := mc.Writer.Write(p)
	if n > 0 {
		mc.bytesWritten += uint64(n)
	}
	return n, err
}

// WriteByte writes a single byte and meters it
func (mc *meteredConn) WriteByte(c byte) error {
	err := mc.Writer.WriteByte(c)
	if err == nil {
		mc.bytesWritten++
	}
	return err
}

// Flush flushes the write buffer
func (mc *meteredConn) Flush() error {
	return mc.Writer.Flush()
}

// BytesRead returns the total number of bytes read
func (mc *meteredConn) BytesRead() uint64 {
	return mc.bytesRead
}

// BytesWritten returns the total number of bytes written
func (mc *meteredConn) BytesWritten() uint64 {
	return mc.bytesWritten
}
