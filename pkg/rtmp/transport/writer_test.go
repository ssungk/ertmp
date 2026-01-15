package transport

import (
	"bytes"
	"testing"
)

// TestWriterExtendedTimestamp_Basic tests Extended Timestamp support
func TestWriterExtendedTimestamp_Basic(t *testing.T) {
	// Create test connection
	conn := newTestConn()
	mc := newMeteredConn(conn)
	writer := NewWriter(mc)

	// Create message with Extended Timestamp
	// ExtendedTimestampThreshold = 0xFFFFFF
	extTimestamp := uint32(ExtendedTimestampThreshold + 1000)
	data := []byte("test data with extended timestamp")

	header := NewMessageHeader(1, extTimestamp, MsgTypeAMF0Command)
	msg := NewMessage(header, data)

	// Write message
	if err := writer.WriteMessage(msg); err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	// Flush writer
	if err := writer.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	msg.Release()

	// Copy writeBuf to readBuf for reading
	conn.readBuf.Write(conn.writeBuf.Bytes())
	conn.writeBuf.Reset()

	// Create reader
	reader := NewReader(newMeteredConn(conn))

	// Read message back
	receivedMsg, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	defer receivedMsg.Release()

	// Verify timestamp
	if receivedMsg.Timestamp() != extTimestamp {
		t.Errorf("Timestamp mismatch: expected %d (0x%X), got %d (0x%X)",
			extTimestamp, extTimestamp, receivedMsg.Timestamp(), receivedMsg.Timestamp())
	}

	// Verify data
	receivedData := receivedMsg.Data()
	if !bytes.Equal(receivedData, data) {
		t.Errorf("Data mismatch: expected %v, got %v", data, receivedData)
	}

	t.Logf("Extended Timestamp test passed: timestamp=0x%X (%d)", extTimestamp, extTimestamp)
}

// TestWriterExtendedTimestamp_Boundary tests timestamp at boundary
func TestWriterExtendedTimestamp_Boundary(t *testing.T) {
	testCases := []struct {
		name      string
		timestamp uint32
	}{
		{"Below threshold", ExtendedTimestampThreshold - 1},
		{"At threshold", ExtendedTimestampThreshold},
		{"Just above threshold", ExtendedTimestampThreshold + 1},
		{"Far above threshold", ExtendedTimestampThreshold + 100000},
		{"Max uint32", 0xFFFFFFFF},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test connection
			conn := newTestConn()
			mc := newMeteredConn(conn)
			writer := NewWriter(mc)

			data := []byte("boundary test")
			header := NewMessageHeader(1, tc.timestamp, MsgTypeAMF0Command)
			msg := NewMessage(header, data)

			// Write message
			if err := writer.WriteMessage(msg); err != nil {
				t.Fatalf("WriteMessage failed: %v", err)
			}

			if err := writer.Flush(); err != nil {
				t.Fatalf("Flush failed: %v", err)
			}

			msg.Release()

			// Copy writeBuf to readBuf for reading
			conn.readBuf.Write(conn.writeBuf.Bytes())
			conn.writeBuf.Reset()

			// Create reader
			reader := NewReader(newMeteredConn(conn))

			// Read message back
			receivedMsg, err := reader.ReadMessage()
			if err != nil {
				t.Fatalf("ReadMessage failed: %v", err)
			}
			defer receivedMsg.Release()

			// Verify timestamp
			if receivedMsg.Timestamp() != tc.timestamp {
				t.Errorf("Timestamp mismatch: expected %d (0x%X), got %d (0x%X)",
					tc.timestamp, tc.timestamp, receivedMsg.Timestamp(), receivedMsg.Timestamp())
			}

			// Verify data
			if !bytes.Equal(receivedMsg.Data(), data) {
				t.Errorf("Data mismatch")
			}

			t.Logf("Boundary test passed: %s - timestamp=0x%X", tc.name, tc.timestamp)
		})
	}
}

// TestWriterExtendedTimestamp_MultiChunk tests Extended Timestamp with large messages
func TestWriterExtendedTimestamp_MultiChunk(t *testing.T) {
	// Create test connection
	conn := newTestConn()
	mc := newMeteredConn(conn)
	writer := NewWriter(mc)

	// Create large message (10 chunks)
	extTimestamp := uint32(ExtendedTimestampThreshold + 5000)
	chunkSize := DefaultChunkSize
	dataSize := int(chunkSize * 10)
	data := make([]byte, dataSize)
	for i := range data {
		data[i] = byte(i % 256)
	}

	header := NewMessageHeader(1, extTimestamp, MsgTypeAMF0Data)
	msg := NewMessage(header, data)

	// Write message
	if err := writer.WriteMessage(msg); err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	if err := writer.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	msg.Release()

	// Copy writeBuf to readBuf for reading
	conn.readBuf.Write(conn.writeBuf.Bytes())
	conn.writeBuf.Reset()

	// Create reader
	reader := NewReader(newMeteredConn(conn))

	// Read message back
	receivedMsg, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	defer receivedMsg.Release()

	// Verify timestamp
	if receivedMsg.Timestamp() != extTimestamp {
		t.Errorf("Timestamp mismatch: expected %d (0x%X), got %d (0x%X)",
			extTimestamp, extTimestamp, receivedMsg.Timestamp(), receivedMsg.Timestamp())
	}

	// Verify data
	receivedData := receivedMsg.Data()
	if len(receivedData) != dataSize {
		t.Errorf("Data size mismatch: expected %d, got %d", dataSize, len(receivedData))
	}

	if !bytes.Equal(receivedData, data) {
		t.Errorf("Data content mismatch")
	}

	t.Logf("Multi-chunk Extended Timestamp test passed: %d chunks, timestamp=0x%X",
		dataSize/int(chunkSize), extTimestamp)
}

// TestWriterExtendedTimestamp_RoundTrip tests various timestamp values
func TestWriterExtendedTimestamp_RoundTrip(t *testing.T) {
	testTimestamps := []uint32{
		0,
		1000,
		0xFFFFFE,              // Just below threshold
		0xFFFFFF,              // At threshold (ExtendedTimestampThreshold)
		0xFFFFFF + 1,          // Just above threshold
		0xFFFFFF + 1000000,    // Far above threshold
		ChunkSizeMsgMask,      // Max int32 (0x7FFFFFFF)
		0xFFFFFFFF,            // Max uint32
	}

	for _, ts := range testTimestamps {
		t.Run("", func(t *testing.T) {
			// Create test connection
			conn := newTestConn()
			mc := newMeteredConn(conn)
			writer := NewWriter(mc)

			data := []byte("round-trip test")
			header := NewMessageHeader(1, ts, MsgTypeAMF0Command)
			msg := NewMessage(header, data)

			// Write
			if err := writer.WriteMessage(msg); err != nil {
				t.Fatalf("WriteMessage failed for ts=0x%X: %v", ts, err)
			}

			if err := writer.Flush(); err != nil {
				t.Fatalf("Flush failed for ts=0x%X: %v", ts, err)
			}

			msg.Release()

			// Copy writeBuf to readBuf for reading
			conn.readBuf.Write(conn.writeBuf.Bytes())
			conn.writeBuf.Reset()

			// Create reader
			reader := NewReader(newMeteredConn(conn))

			// Read
			receivedMsg, err := reader.ReadMessage()
			if err != nil {
				t.Fatalf("ReadMessage failed for ts=0x%X: %v", ts, err)
			}
			defer receivedMsg.Release()

			// Verify
			if receivedMsg.Timestamp() != ts {
				t.Errorf("Round-trip failed for ts=0x%X: got 0x%X", ts, receivedMsg.Timestamp())
			}

			if !bytes.Equal(receivedMsg.Data(), data) {
				t.Errorf("Data mismatch for ts=0x%X", ts)
			}
		})
	}

	t.Logf("Round-trip test passed for %d timestamp values", len(testTimestamps))
}

// TestWriterTimestampDelta_Basic tests Timestamp Delta for FmtType1/2
func TestWriterTimestampDelta_Basic(t *testing.T) {
	// Create test connection
	conn := newTestConn()
	mc := newMeteredConn(conn)
	writer := NewWriter(mc)

	// Send two messages on the same stream with different timestamps
	ts1 := uint32(1000)
	ts2 := uint32(2000) // delta = 1000

	data1 := []byte("first message")
	header1 := NewMessageHeader(1, ts1, MsgTypeAMF0Command)
	msg1 := NewMessage(header1, data1)

	data2 := []byte("second message")
	header2 := NewMessageHeader(1, ts2, MsgTypeAMF0Command)
	msg2 := NewMessage(header2, data2)

	// Write first message
	if err := writer.WriteMessage(msg1); err != nil {
		t.Fatalf("WriteMessage msg1 failed: %v", err)
	}

	// Write second message (should use FmtType2 with delta)
	if err := writer.WriteMessage(msg2); err != nil {
		t.Fatalf("WriteMessage msg2 failed: %v", err)
	}

	if err := writer.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	msg1.Release()
	msg2.Release()

	// Copy writeBuf to readBuf for reading
	conn.readBuf.Write(conn.writeBuf.Bytes())
	conn.writeBuf.Reset()

	// Create reader
	reader := NewReader(newMeteredConn(conn))

	// Read first message
	receivedMsg1, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage msg1 failed: %v", err)
	}
	defer receivedMsg1.Release()

	if receivedMsg1.Timestamp() != ts1 {
		t.Errorf("msg1 timestamp mismatch: expected %d, got %d", ts1, receivedMsg1.Timestamp())
	}

	if !bytes.Equal(receivedMsg1.Data(), data1) {
		t.Errorf("msg1 data mismatch")
	}

	// Read second message
	receivedMsg2, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage msg2 failed: %v", err)
	}
	defer receivedMsg2.Release()

	if receivedMsg2.Timestamp() != ts2 {
		t.Errorf("msg2 timestamp mismatch: expected %d, got %d", ts2, receivedMsg2.Timestamp())
	}

	if !bytes.Equal(receivedMsg2.Data(), data2) {
		t.Errorf("msg2 data mismatch")
	}

	t.Logf("Timestamp Delta test passed: ts1=%d, ts2=%d, delta=%d", ts1, ts2, ts2-ts1)
}

// TestWriterTimestampDelta_Extended tests Timestamp Delta with Extended Timestamp
func TestWriterTimestampDelta_Extended(t *testing.T) {
	// Create test connection
	conn := newTestConn()
	mc := newMeteredConn(conn)
	writer := NewWriter(mc)

	// Send two messages with Extended Timestamp deltas
	ts1 := uint32(ExtendedTimestampThreshold + 1000)
	ts2 := uint32(ExtendedTimestampThreshold + 2000) // delta = 1000

	data1 := []byte("first message with extended timestamp")
	header1 := NewMessageHeader(1, ts1, MsgTypeAMF0Command)
	msg1 := NewMessage(header1, data1)

	data2 := []byte("second message with extended timestamp")
	header2 := NewMessageHeader(1, ts2, MsgTypeAMF0Command)
	msg2 := NewMessage(header2, data2)

	// Write first message
	if err := writer.WriteMessage(msg1); err != nil {
		t.Fatalf("WriteMessage msg1 failed: %v", err)
	}

	// Write second message
	if err := writer.WriteMessage(msg2); err != nil {
		t.Fatalf("WriteMessage msg2 failed: %v", err)
	}

	if err := writer.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	msg1.Release()
	msg2.Release()

	// Copy writeBuf to readBuf for reading
	conn.readBuf.Write(conn.writeBuf.Bytes())
	conn.writeBuf.Reset()

	// Create reader
	reader := NewReader(newMeteredConn(conn))

	// Read first message
	receivedMsg1, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage msg1 failed: %v", err)
	}
	defer receivedMsg1.Release()

	if receivedMsg1.Timestamp() != ts1 {
		t.Errorf("msg1 timestamp mismatch: expected %d (0x%X), got %d (0x%X)",
			ts1, ts1, receivedMsg1.Timestamp(), receivedMsg1.Timestamp())
	}

	if !bytes.Equal(receivedMsg1.Data(), data1) {
		t.Errorf("msg1 data mismatch")
	}

	// Read second message
	receivedMsg2, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage msg2 failed: %v", err)
	}
	defer receivedMsg2.Release()

	if receivedMsg2.Timestamp() != ts2 {
		t.Errorf("msg2 timestamp mismatch: expected %d (0x%X), got %d (0x%X)",
			ts2, ts2, receivedMsg2.Timestamp(), receivedMsg2.Timestamp())
	}

	if !bytes.Equal(receivedMsg2.Data(), data2) {
		t.Errorf("msg2 data mismatch")
	}

	t.Logf("Extended Timestamp Delta test passed: ts1=0x%X, ts2=0x%X, delta=%d",
		ts1, ts2, ts2-ts1)
}

// TestWriterTimestampDelta_LargeDelta tests large delta (> 0xFFFFFF)
func TestWriterTimestampDelta_LargeDelta(t *testing.T) {
	// Create test connection
	conn := newTestConn()
	mc := newMeteredConn(conn)
	writer := NewWriter(mc)

	// Send two messages with large delta
	ts1 := uint32(1000)
	ts2 := uint32(ExtendedTimestampThreshold + 5000) // delta > 0xFFFFFF

	data1 := []byte("first")
	header1 := NewMessageHeader(1, ts1, MsgTypeAMF0Command)
	msg1 := NewMessage(header1, data1)

	data2 := []byte("second")
	header2 := NewMessageHeader(1, ts2, MsgTypeAMF0Command)
	msg2 := NewMessage(header2, data2)

	// Write messages
	if err := writer.WriteMessage(msg1); err != nil {
		t.Fatalf("WriteMessage msg1 failed: %v", err)
	}

	if err := writer.WriteMessage(msg2); err != nil {
		t.Fatalf("WriteMessage msg2 failed: %v", err)
	}

	if err := writer.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	msg1.Release()
	msg2.Release()

	// Copy writeBuf to readBuf for reading
	conn.readBuf.Write(conn.writeBuf.Bytes())
	conn.writeBuf.Reset()

	// Create reader
	reader := NewReader(newMeteredConn(conn))

	// Read messages
	receivedMsg1, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage msg1 failed: %v", err)
	}
	defer receivedMsg1.Release()

	receivedMsg2, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage msg2 failed: %v", err)
	}
	defer receivedMsg2.Release()

	// Verify timestamps
	if receivedMsg1.Timestamp() != ts1 {
		t.Errorf("msg1 timestamp mismatch: expected %d, got %d", ts1, receivedMsg1.Timestamp())
	}

	if receivedMsg2.Timestamp() != ts2 {
		t.Errorf("msg2 timestamp mismatch: expected %d (0x%X), got %d (0x%X)",
			ts2, ts2, receivedMsg2.Timestamp(), receivedMsg2.Timestamp())
	}

	t.Logf("Large Delta test passed: ts1=%d, ts2=0x%X, delta=0x%X", ts1, ts2, ts2-ts1)
}

// TestWriterFmtType3_ExtendedTimestamp tests FmtType3 with Extended Timestamp in continuation chunks
func TestWriterFmtType3_ExtendedTimestamp(t *testing.T) {
	// Create test connection
	conn := newTestConn()
	mc := newMeteredConn(conn)
	writer := NewWriter(mc)

	// Create message with Extended Timestamp that spans 2 chunks
	extTimestamp := uint32(ExtendedTimestampThreshold + 1000)
	chunkSize := DefaultChunkSize // 128
	dataSize := int(chunkSize + 50) // 178 bytes, spans 2 chunks
	data := make([]byte, dataSize)
	for i := range data {
		data[i] = byte(i % 256)
	}

	header := NewMessageHeader(1, extTimestamp, MsgTypeAMF0Data)
	msg := NewMessage(header, data)

	t.Logf("Sending message: timestamp=0x%X, size=%d bytes, chunks=2", extTimestamp, dataSize)

	// Write message
	if err := writer.WriteMessage(msg); err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	if err := writer.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	msg.Release()

	t.Logf("Written %d bytes to writeBuf", conn.writeBuf.Len())

	// Copy writeBuf to readBuf for reading
	conn.readBuf.Write(conn.writeBuf.Bytes())
	conn.writeBuf.Reset()

	// Create reader
	reader := NewReader(newMeteredConn(conn))

	// Read message back
	receivedMsg, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	defer receivedMsg.Release()

	// Verify timestamp
	if receivedMsg.Timestamp() != extTimestamp {
		t.Errorf("Timestamp mismatch: expected %d (0x%X), got %d (0x%X)",
			extTimestamp, extTimestamp, receivedMsg.Timestamp(), receivedMsg.Timestamp())
	}

	// Verify data
	receivedData := receivedMsg.Data()
	if len(receivedData) != dataSize {
		t.Errorf("Data size mismatch: expected %d, got %d", dataSize, len(receivedData))
	} else {
		t.Logf("Data size OK: %d bytes", len(receivedData))
	}

	if !bytes.Equal(receivedData, data) {
		t.Errorf("Data content mismatch")
		// Find first mismatched byte
		for i := 0; i < len(data) && i < len(receivedData); i++ {
			if data[i] != receivedData[i] {
				t.Logf("First mismatch at byte %d: expected %d, got %d", i, data[i], receivedData[i])
				if i > 0 {
					t.Logf("Previous bytes: expected %v, got %v", data[i-1:i], receivedData[i-1:i])
				}
				break
			}
		}
	} else {
		t.Logf("Data content OK")
	}

	t.Logf("FmtType3 Extended Timestamp test passed: 2 chunks, timestamp=0x%X", extTimestamp)
}
