package transport

import (
	"testing"
)

func TestMessageAssembler_SingleChunkMessage(t *testing.T) {
	ma := newMessageAssembler()

	header := NewMessageHeader(1, 1000, MsgTypeAudio)
	header.MessageLength = 50

	if !ma.isNewMessage() {
		t.Fatal("expected new message state")
	}

	ma.startMessage(header)

	if ma.buffer == nil {
		t.Fatal("buffer not allocated")
	}
	if len(ma.buffer.Data()) != 50 {
		t.Fatalf("buffer size = %d, want 50", len(ma.buffer.Data()))
	}

	// Write data
	buf := ma.nextBuffer(50)
	for i := range buf {
		buf[i] = byte(i)
	}
	ma.addBytes(50)

	if !ma.isComplete() {
		t.Fatal("expected complete")
	}

	// Move buffer
	moved := ma.moveBuffer()
	if moved == nil {
		t.Fatal("moveBuffer returned nil")
	}

	if !ma.isNewMessage() {
		t.Error("expected new message state after move")
	}

	// Verify data sample
	data := moved.Data()
	if data[0] != 0 || data[25] != 25 || data[49] != 49 {
		t.Error("data verification failed")
	}

	moved.Release()
}

func TestMessageAssembler_MultiChunkMessage(t *testing.T) {
	ma := newMessageAssembler()

	header := NewMessageHeader(2, 5000, MsgTypeVideo)
	header.MessageLength = 300

	ma.startMessage(header)

	// Chunk 1: 128 bytes
	buf1 := ma.nextBuffer(128)
	for i := range buf1 {
		buf1[i] = 0xAA
	}
	ma.addBytes(128)

	if ma.isComplete() {
		t.Error("should not be complete after chunk 1")
	}
	if ma.isNewMessage() {
		t.Error("should not be new message after chunk 1")
	}

	// Chunk 2: 128 bytes
	buf2 := ma.nextBuffer(128)
	for i := range buf2 {
		buf2[i] = 0xBB
	}
	ma.addBytes(128)

	if ma.isComplete() {
		t.Error("should not be complete after chunk 2")
	}

	// Chunk 3: 44 bytes (final)
	buf3 := ma.nextBuffer(44)
	for i := range buf3 {
		buf3[i] = 0xCC
	}
	ma.addBytes(44)

	if !ma.isComplete() {
		t.Fatal("should be complete after chunk 3")
	}

	// Verify assembled data sample
	data := ma.buffer.Data()
	if data[0] != 0xAA || data[127] != 0xAA {
		t.Error("chunk 1 data verification failed")
	}
	if data[128] != 0xBB || data[255] != 0xBB {
		t.Error("chunk 2 data verification failed")
	}
	if data[256] != 0xCC || data[299] != 0xCC {
		t.Error("chunk 3 data verification failed")
	}

	ma.clear()
}

func TestMessageAssembler_PartialMessageClear(t *testing.T) {
	ma := newMessageAssembler()

	header := NewMessageHeader(0, 0, 0)
	header.MessageLength = 200
	ma.startMessage(header)

	buf := ma.nextBuffer(80)
	for i := range buf {
		buf[i] = byte(i)
	}
	ma.addBytes(80)

	if ma.isComplete() {
		t.Error("should not be complete")
	}

	ma.clear()

	if ma.buffer != nil {
		t.Error("buffer should be nil after clear")
	}
	if !ma.isNewMessage() {
		t.Error("expected new message state after clear")
	}

	// Safe to call clear again
	ma.clear()
}

func TestMessageAssembler_HeaderPersistence(t *testing.T) {
	ma := newMessageAssembler()

	header1 := NewMessageHeader(5, 2000, MsgTypeAMF0Command)
	header1.MessageLength = 100
	ma.startMessage(header1)

	buf := ma.nextBuffer(100)
	for i := range buf {
		buf[i] = byte(i)
	}
	ma.addBytes(100)

	moved := ma.moveBuffer()
	if moved == nil {
		t.Fatal("moveBuffer returned nil")
	}

	// Verify header persists after moveBuffer (acts as prevHeader)
	if ma.header().Timestamp != 2000 {
		t.Errorf("header not persisted: got timestamp %d, want 2000", ma.header().Timestamp)
	}

	// Verify header is replaced when starting new message
	header2 := NewMessageHeader(5, 3000, MsgTypeAMF0Command)
	header2.MessageLength = 50
	ma.startMessage(header2)

	if ma.header().Timestamp != 3000 {
		t.Errorf("header not updated: got timestamp %d, want 3000", ma.header().Timestamp)
	}

	moved.Release()
	ma.clear()
}
