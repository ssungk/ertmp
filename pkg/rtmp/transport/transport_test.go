package transport

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

func TestTransportAck_Disabled(t *testing.T) {
	conn := newTestConn()
	transport := NewTransport(conn)

	// windowAckSize is 0 by default
	if transport.windowAckSize != 0 {
		t.Errorf("expected windowAckSize=0, got %d", transport.windowAckSize)
	}

	// Write a small message to readBuf
	data := make([]byte, 1000)
	writeTestMessage(conn.readBuf, data)

	// Read message
	msg, err := transport.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	msg.Release()

	// Verify bytesRead is updated
	if transport.conn.BytesRead() == 0 {
		t.Error("bytesRead should be > 0")
	}

	// Verify lastAckSent is still 0 (no ACK sent)
	if transport.lastAckSent != 0 {
		t.Errorf("expected lastAckSent=0, got %d", transport.lastAckSent)
	}

	// Verify no ACK was sent to writeBuf
	if conn.writeBuf.Len() > 0 {
		t.Errorf("expected no ACK, but writeBuf has %d bytes", conn.writeBuf.Len())
	}

	t.Logf("BytesRead: %d, lastAckSent: %d", transport.conn.BytesRead(), transport.lastAckSent)
}

func TestTransportAck_Basic(t *testing.T) {
	conn := newTestConn()
	transport := NewTransport(conn)

	// Set windowAckSize to 500 bytes (small for testing)
	windowSize := uint32(500)
	transport.windowAckSize = windowSize

	// Write 600 byte message to readBuf
	data := make([]byte, 600)
	writeTestMessage(conn.readBuf, data)

	// Read message (this should trigger ACK)
	msg, err := transport.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	msg.Release()

	// Check bytesRead
	bytesRead := transport.conn.BytesRead()
	t.Logf("bytesRead: %d, lastAckSent: %d, windowSize: %d",
		bytesRead, transport.lastAckSent, windowSize)

	// Check lastAckSent is aligned to windowAckSize
	if transport.lastAckSent != uint64(windowSize) {
		t.Errorf("expected lastAckSent=%d, got %d", windowSize, transport.lastAckSent)
	}

	// Read ACK from writeBuf
	ackBytes, err := readAckMessage(conn.writeBuf)
	if err != nil {
		t.Fatalf("failed to read ACK: %v", err)
	}

	// ACK should contain the lastAckSent value (window boundary)
	if ackBytes != uint32(windowSize) {
		t.Errorf("expected ACK(%d), got ACK(%d)", windowSize, ackBytes)
	}

	t.Logf("bytesRead: %d, ACK sent: %d", bytesRead, ackBytes)
}

func TestTransportAck_MultipleAcks(t *testing.T) {
	conn := newTestConn()
	transport := NewTransport(conn)

	// Set windowAckSize to 2.5MB
	windowSize := uint32(2500000)
	transport.windowAckSize = windowSize

	// Write 10MB message to readBuf
	data := make([]byte, 10*1024*1024)
	writeTestMessage(conn.readBuf, data)

	// Read message
	msg, err := transport.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	msg.Release()

	// Should receive 4 ACKs: 2.5MB, 5MB, 7.5MB, 10MB
	expectedAcks := []uint32{
		windowSize,
		windowSize * 2,
		windowSize * 3,
		windowSize * 4,
	}

	for i, expectedAck := range expectedAcks {
		ackBytes, err := readAckMessage(conn.writeBuf)
		if err != nil {
			t.Fatalf("failed to read ACK %d: %v", i+1, err)
		}

		// Each ACK should report the window boundary value
		if ackBytes != expectedAck {
			t.Errorf("ACK %d: expected ACK(%d), got ACK(%d)", i+1, expectedAck, ackBytes)
		}

		t.Logf("ACK %d: sent ACK(%d)", i+1, ackBytes)
	}

	// Check final lastAckSent
	expectedFinalAckSent := uint64(windowSize) * 4
	if transport.lastAckSent != expectedFinalAckSent {
		t.Errorf("expected lastAckSent=%d, got %d", expectedFinalAckSent, transport.lastAckSent)
	}
}

func TestTransportAck_MidStreamConfig(t *testing.T) {
	conn := newTestConn()
	transport := NewTransport(conn)

	// Initially windowAckSize = 0, read 2.8MB
	data1 := make([]byte, 2800000)
	writeTestMessage(conn.readBuf, data1)

	msg1, err := transport.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage 1 failed: %v", err)
	}
	msg1.Release()

	// No ACK should be sent yet
	bytesReadBefore := transport.conn.BytesRead()
	if conn.writeBuf.Len() > 0 {
		t.Errorf("expected no ACK before config, but writeBuf has %d bytes", conn.writeBuf.Len())
	}

	// Now set windowAckSize to 2.5MB
	windowSize := uint32(2500000)
	transport.windowAckSize = windowSize

	// Read another small message
	data2 := make([]byte, 100)
	writeTestMessage(conn.readBuf, data2)

	msg2, err := transport.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage 2 failed: %v", err)
	}
	msg2.Release()

	// ACK should be sent now
	ackBytes, err := readAckMessage(conn.writeBuf)
	if err != nil {
		t.Fatalf("failed to read ACK: %v", err)
	}

	// lastAckSent should be aligned to 2.5MB
	if transport.lastAckSent != uint64(windowSize) {
		t.Errorf("expected lastAckSent=%d, got %d", windowSize, transport.lastAckSent)
	}

	// ACK should contain the window boundary value (2.5MB)
	bytesRead := transport.conn.BytesRead()
	if ackBytes != windowSize {
		t.Errorf("expected ACK(%d), got ACK(%d)", windowSize, ackBytes)
	}

	t.Logf("BytesRead before config: %d, after: %d, ACK sent: %d", bytesReadBefore, bytesRead, ackBytes)
}

// Helper functions and types

// testConn implements io.ReadWriteCloser with separate read/write buffers
type testConn struct {
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
}

func newTestConn() *testConn {
	return &testConn{
		readBuf:  new(bytes.Buffer),
		writeBuf: new(bytes.Buffer),
	}
}

func (tc *testConn) Read(p []byte) (int, error) {
	return tc.readBuf.Read(p)
}

func (tc *testConn) Write(p []byte) (int, error) {
	return tc.writeBuf.Write(p)
}

func (tc *testConn) Close() error {
	return nil
}

// writeTestMessage writes a test message to a buffer in RTMP chunk format
func writeTestMessage(buf *bytes.Buffer, data []byte) {
	chunkSize := uint32(DefaultChunkSize) // 128 bytes
	msgLen := uint32(len(data))

	// Write first chunk with full header (fmt=0, csid=3)
	buf.WriteByte(0x03) // basic header

	// Write message header (fmt 0: 11 bytes)
	header := make([]byte, 11)
	// timestamp (3 bytes): 0
	header[0] = 0
	header[1] = 0
	header[2] = 0
	// message length (3 bytes)
	header[3] = byte(msgLen >> 16)
	header[4] = byte(msgLen >> 8)
	header[5] = byte(msgLen)
	// message type (1 byte): AMF0 Command
	header[6] = MsgTypeAMF0Command
	// message stream id (4 bytes, little endian): 0
	header[7] = 0
	header[8] = 0
	header[9] = 0
	header[10] = 0
	buf.Write(header)

	// Write data in chunks
	bytesWritten := uint32(0)
	isFirstChunk := true

	for bytesWritten < msgLen {
		// Calculate chunk data size
		remaining := msgLen - bytesWritten
		chunkDataSize := chunkSize
		if remaining < chunkDataSize {
			chunkDataSize = remaining
		}

		// Write continuation header for subsequent chunks (fmt=3)
		if !isFirstChunk {
			buf.WriteByte(0xC3) // fmt=3, csid=3
		}

		// Write chunk data
		buf.Write(data[bytesWritten : bytesWritten+chunkDataSize])

		bytesWritten += chunkDataSize
		isFirstChunk = false
	}
}

// readAckMessage reads and verifies ACK message from buffer
func readAckMessage(buf *bytes.Buffer) (uint32, error) {
	// Read basic header
	basicHeader, err := buf.ReadByte()
	if err != nil {
		return 0, err
	}

	// Parse fmt from basic header
	fmt := (basicHeader >> 6) & 0x03

	// Determine header size based on fmt
	var headerSize int
	switch fmt {
	case 0:
		headerSize = 11 // Type 0: full header
	case 1:
		headerSize = 7 // Type 1: no message stream ID
	case 2:
		headerSize = 3 // Type 2: timestamp delta only
	case 3:
		headerSize = 0 // Type 3: no header
	}

	// Read message header
	if headerSize > 0 {
		msgHeader := make([]byte, headerSize)
		if _, err := io.ReadFull(buf, msgHeader); err != nil {
			return 0, err
		}
	}

	// Read ACK payload (4 bytes)
	payload := make([]byte, 4)
	if _, err := io.ReadFull(buf, payload); err != nil {
		return 0, err
	}

	ackBytes := binary.BigEndian.Uint32(payload)
	return ackBytes, nil
}
