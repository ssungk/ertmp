package transport

import (
	"bytes"
	"io"
	"testing"
)

func TestMeteredConn_Read(t *testing.T) {
	data := []byte("Hello, World!")
	mc := newMeteredConn(newBytesReadWriter(data))

	// Read 5 bytes
	buf := make([]byte, 5)
	n, err := mc.Read(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Fatalf("expected to read 5 bytes, got %d", n)
	}
	if mc.BytesRead() != 5 {
		t.Errorf("expected bytesRead=5, got %d", mc.BytesRead())
	}

	// Read remaining bytes
	buf = make([]byte, 100)
	n, err = mc.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 8 {
		t.Fatalf("expected to read 8 bytes, got %d", n)
	}
	if mc.BytesRead() != 13 {
		t.Errorf("expected bytesRead=13, got %d", mc.BytesRead())
	}
}

func TestMeteredConn_ReadByte(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	mc := newMeteredConn(newBytesReadWriter(data))

	for i := 0; i < 3; i++ {
		b, err := mc.ReadByte()
		if err != nil {
			t.Fatalf("unexpected error at byte %d: %v", i, err)
		}
		if b != data[i] {
			t.Errorf("expected byte %d to be 0x%02x, got 0x%02x", i, data[i], b)
		}
		if mc.BytesRead() != uint64(i+1) {
			t.Errorf("expected bytesRead=%d, got %d", i+1, mc.BytesRead())
		}
	}

	// Read past EOF
	_, err := mc.ReadByte()
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
	// bytesRead should not increase on error
	if mc.BytesRead() != 3 {
		t.Errorf("expected bytesRead=3 after EOF, got %d", mc.BytesRead())
	}
}

func TestMeteredConn_Write(t *testing.T) {
	buf := &bytes.Buffer{}
	mc := newMeteredConn(newBytesReadWriter(nil))
	mc.Writer.Reset(buf)

	// Write 5 bytes
	data := []byte("Hello")
	n, err := mc.Write(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Fatalf("expected to write 5 bytes, got %d", n)
	}
	if mc.BytesWritten() != 5 {
		t.Errorf("expected bytesWritten=5, got %d", mc.BytesWritten())
	}

	// Write more bytes
	data = []byte(", World!")
	n, err = mc.Write(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 8 {
		t.Fatalf("expected to write 8 bytes, got %d", n)
	}
	if mc.BytesWritten() != 13 {
		t.Errorf("expected bytesWritten=13, got %d", mc.BytesWritten())
	}

	// Flush and verify
	if err := mc.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if buf.String() != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %q", buf.String())
	}
}

func TestMeteredConn_WriteByte(t *testing.T) {
	buf := &bytes.Buffer{}
	mc := newMeteredConn(newBytesReadWriter(nil))
	mc.Writer.Reset(buf)

	data := []byte{0x01, 0x02, 0x03}
	for i, b := range data {
		err := mc.WriteByte(b)
		if err != nil {
			t.Fatalf("unexpected error at byte %d: %v", i, err)
		}
		if mc.BytesWritten() != uint64(i+1) {
			t.Errorf("expected bytesWritten=%d, got %d", i+1, mc.BytesWritten())
		}
	}

	if err := mc.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), data) {
		t.Errorf("expected %v, got %v", data, buf.Bytes())
	}
}

func TestMeteredConn_ReadWrite(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	buf := &bytes.Buffer{}
	mc := newMeteredConn(newBytesReadWriter(data))
	mc.Writer.Reset(buf)

	// Read 2 bytes
	readBuf := make([]byte, 2)
	n, err := mc.Read(readBuf)
	if err != nil || n != 2 {
		t.Fatalf("Read failed: %v, n=%d", err, n)
	}
	if mc.BytesRead() != 2 {
		t.Errorf("expected bytesRead=2, got %d", mc.BytesRead())
	}

	// Write 3 bytes
	writeData := []byte{0x0A, 0x0B, 0x0C}
	n, err = mc.Write(writeData)
	if err != nil || n != 3 {
		t.Fatalf("Write failed: %v, n=%d", err, n)
	}
	if mc.BytesWritten() != 3 {
		t.Errorf("expected bytesWritten=3, got %d", mc.BytesWritten())
	}

	// Read 1 byte
	b, err := mc.ReadByte()
	if err != nil || b != 0x03 {
		t.Fatalf("ReadByte failed: %v, byte=%02x", err, b)
	}
	if mc.BytesRead() != 3 {
		t.Errorf("expected bytesRead=3, got %d", mc.BytesRead())
	}

	// Write 1 byte
	err = mc.WriteByte(0x0D)
	if err != nil {
		t.Fatalf("WriteByte failed: %v", err)
	}
	if mc.BytesWritten() != 4 {
		t.Errorf("expected bytesWritten=4, got %d", mc.BytesWritten())
	}

	// Flush
	if err := mc.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	expected := []byte{0x0A, 0x0B, 0x0C, 0x0D}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("expected %v, got %v", expected, buf.Bytes())
	}
}

// bytesReadWriter implements io.ReadWriter for testing
type bytesReadWriter struct {
	*bytes.Reader
	*bytes.Buffer
}

func newBytesReadWriter(data []byte) *bytesReadWriter {
	return &bytesReadWriter{
		Reader: bytes.NewReader(data),
		Buffer: &bytes.Buffer{},
	}
}

func (brw *bytesReadWriter) Read(p []byte) (int, error) {
	return brw.Reader.Read(p)
}

func (brw *bytesReadWriter) Write(p []byte) (int, error) {
	return brw.Buffer.Write(p)
}
