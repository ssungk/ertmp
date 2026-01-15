package transport

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/ssungk/ertmp/pkg/rtmp/buf"
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

func TestTransportAck_NoRetroactiveAcks(t *testing.T) {
	conn := newTestConn()
	transport := NewTransport(conn)

	// windowAckSize=0으로 10MB 읽기 (ACK 없음)
	data1 := make([]byte, 10*1024*1024)
	writeTestMessage(conn.readBuf, data1)

	msg1, err := transport.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage 1 failed: %v", err)
	}
	msg1.Release()

	bytesReadBefore := transport.conn.BytesRead()
	if conn.writeBuf.Len() > 0 {
		t.Errorf("expected no ACK before windowAckSize set, but writeBuf has %d bytes", conn.writeBuf.Len())
	}

	// windowAckSize를 2.5MB로 설정
	windowSize := uint32(2500000)
	transport.windowAckSize = windowSize
	transport.lastAckSent = bytesReadBefore // 수동 설정으로 버그 수정 시뮬레이션

	// 작은 데이터(100 bytes) 추가로 읽기
	data2 := make([]byte, 100)
	writeTestMessage(conn.readBuf, data2)

	msg2, err := transport.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage 2 failed: %v", err)
	}
	msg2.Release()

	// ACK가 0개여야 함 (아직 다음 window boundary를 넘지 않음)
	if conn.writeBuf.Len() > 0 {
		// 만약 버그가 있다면 여기서 많은 ACK가 나올 것
		ackCount := conn.writeBuf.Len() / 16 // 대략적인 ACK 메시지 크기
		t.Errorf("expected no ACK (no window boundary crossed), but got approximately %d ACKs in writeBuf", ackCount)
	}

	t.Logf("BytesRead before: %d, after: %d, no retroactive ACKs sent", bytesReadBefore, transport.conn.BytesRead())
}

func TestTransportAbort_ClearChunkStream(t *testing.T) {
	conn := newTestConn()
	transport := NewTransport(conn)

	// Directly create a message assembler with partial data (simulating incomplete message)
	csid := uint32(3)
	ma, ok := transport.reader.assemblers[csid]
	if !ok {
		ma = newMessageAssembler()
		transport.reader.assemblers[csid] = ma
	}

	// Simulate partial message reception
	ma.messageHeader.MessageLength = 300 // Set expected total length first
	if ma.buffer == nil {
		ma.buffer = buf.NewFromPool(int(ma.messageHeader.MessageLength))
	}

	// Write partial data
	partialData := ma.nextBuffer(128)
	for i := 0; i < 128; i++ {
		partialData[i] = byte(i)
	}
	ma.bytesRead += 128

	// Verify assembler has partial data
	if ma.bytesRead != 128 {
		t.Fatalf("expected bytesRead=128, got %d", ma.bytesRead)
	}
	if ma.isComplete() {
		t.Fatal("message should not be complete")
	}

	// Send Abort message for chunk stream 3
	abortPayload := make([]byte, 4)
	binary.BigEndian.PutUint32(abortPayload, csid)
	header := NewMessageHeader(0, 0, MsgTypeAbort)
	abortMsg := NewMessage(header, abortPayload)

	// Process abort
	if err := transport.handleProtocolControl(abortMsg); err != nil {
		t.Fatalf("handleProtocolControl failed: %v", err)
	}
	abortMsg.Release()

	// Verify assembler is cleared
	if ma.bytesRead != 0 {
		t.Errorf("expected bytesRead=0 after abort, got %d", ma.bytesRead)
	}
	if ma.buffer != nil {
		t.Errorf("expected nil buffer after abort, got non-nil buffer")
	}

	// Verify we can send a new complete message on the same chunk stream
	newData := make([]byte, 100)
	for i := range newData {
		newData[i] = byte(i + 100)
	}
	writeTestMessage(conn.readBuf, newData)

	msg, err := transport.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage after abort failed: %v", err)
	}

	receivedData := msg.Data()
	if len(receivedData) != 100 {
		t.Errorf("expected 100 bytes, got %d", len(receivedData))
	}
	if receivedData[0] != 100 {
		t.Errorf("expected first byte 100, got %d", receivedData[0])
	}

	msg.Release()
	t.Logf("Abort message successfully cleared message assembler")
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

// TestTransportPingPong_AutoResponse tests automatic PingResponse to PingRequest
func TestTransportPingPong_AutoResponse(t *testing.T) {
	conn := newTestConn()
	transport := NewTransport(conn)

	// Create PingRequest message with timestamp
	timestamp := uint32(12345)
	pingData := make([]byte, 6) // 2 bytes event type + 4 bytes timestamp
	binary.BigEndian.PutUint16(pingData[0:2], UserControlPingRequest)
	binary.BigEndian.PutUint32(pingData[2:6], timestamp)

	header := NewMessageHeader(0, 0, MsgTypeUserControl)
	pingMsg := NewMessage(header, pingData)

	// Write PingRequest to readBuf
	writePingMessage(conn.readBuf, pingData)

	// Read message (this should trigger PingResponse)
	msg, err := transport.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	msg.Release()
	pingMsg.Release()

	// Verify PingResponse was sent to writeBuf
	if conn.writeBuf.Len() == 0 {
		t.Fatal("Expected PingResponse, but writeBuf is empty")
	}

	// Read PingResponse from writeBuf
	eventType, responseTimestamp, err := readPingMessage(conn.writeBuf)
	if err != nil {
		t.Fatalf("Failed to read PingResponse: %v", err)
	}

	// Verify it's PingResponse
	if eventType != UserControlPingResponse {
		t.Errorf("Expected PingResponse (0x%X), got 0x%X", UserControlPingResponse, eventType)
	}

	// Verify timestamp is the same
	if responseTimestamp != timestamp {
		t.Errorf("Timestamp mismatch: expected %d, got %d", timestamp, responseTimestamp)
	}

	t.Logf("PingRequest (timestamp=%d) -> PingResponse (timestamp=%d)", timestamp, responseTimestamp)
}

// TestTransportUserControl_IgnoreOtherEvents tests that other UserControl events are ignored
func TestTransportUserControl_IgnoreOtherEvents(t *testing.T) {
	testCases := []struct {
		name      string
		eventType uint16
		dataLen   int
	}{
		{"StreamBegin", UserControlStreamBegin, 4},
		{"StreamEOF", UserControlStreamEOF, 4},
		{"SetBufferLen", UserControlSetBufferLen, 8},
		{"PingResponse", UserControlPingResponse, 4},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conn := newTestConn()
			transport := NewTransport(conn)

			// Create UserControl message
			msgData := make([]byte, 2+tc.dataLen)
			binary.BigEndian.PutUint16(msgData[0:2], tc.eventType)
			// Fill with dummy data
			for i := 2; i < len(msgData); i++ {
				msgData[i] = byte(i)
			}

			// Write message to readBuf
			writeTestMessage(conn.readBuf, msgData)

			// Read message (should not trigger any response)
			msg, err := transport.ReadMessage()
			if err != nil {
				t.Fatalf("ReadMessage failed: %v", err)
			}
			msg.Release()

			// Verify no response was sent
			if conn.writeBuf.Len() > 0 {
				t.Errorf("Expected no response for %s, but writeBuf has %d bytes",
					tc.name, conn.writeBuf.Len())
			}

			t.Logf("%s event ignored successfully", tc.name)
		})
	}
}

// Helper functions for Ping/Pong tests

// writePingMessage writes a UserControl message to buffer
func writePingMessage(buf *bytes.Buffer, data []byte) {
	// Write basic header (fmt=0, csid=2 for protocol control)
	buf.WriteByte(0x02)

	// Write message header (fmt 0: 11 bytes)
	header := make([]byte, 11)
	// timestamp (3 bytes): 0
	header[0] = 0
	header[1] = 0
	header[2] = 0
	// message length (3 bytes)
	msgLen := uint32(len(data))
	header[3] = byte(msgLen >> 16)
	header[4] = byte(msgLen >> 8)
	header[5] = byte(msgLen)
	// message type (1 byte): UserControl
	header[6] = MsgTypeUserControl
	// message stream id (4 bytes, little endian): 0
	header[7] = 0
	header[8] = 0
	header[9] = 0
	header[10] = 0
	buf.Write(header)

	// Write data
	buf.Write(data)
}

// readPingMessage reads a Ping/Pong message from buffer
func readPingMessage(buf *bytes.Buffer) (eventType uint16, timestamp uint32, err error) {
	// Read basic header
	_, err = buf.ReadByte()
	if err != nil {
		return 0, 0, err
	}

	// Read message header (11 bytes for fmt 0)
	msgHeader := make([]byte, 11)
	if _, err := io.ReadFull(buf, msgHeader); err != nil {
		return 0, 0, err
	}

	// Read event type (2 bytes)
	eventTypeBytes := make([]byte, 2)
	if _, err := io.ReadFull(buf, eventTypeBytes); err != nil {
		return 0, 0, err
	}
	eventType = binary.BigEndian.Uint16(eventTypeBytes)

	// Read timestamp (4 bytes)
	timestampBytes := make([]byte, 4)
	if _, err := io.ReadFull(buf, timestampBytes); err != nil {
		return 0, 0, err
	}
	timestamp = binary.BigEndian.Uint32(timestampBytes)

	return eventType, timestamp, nil
}
