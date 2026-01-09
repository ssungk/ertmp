package transport

import (
	"encoding/binary"
	"fmt"
	"net"
)

// Transport represents a bidirectional RTMP protocol handler
type Transport struct {
	netConn net.Conn
	conn    *meteredConn
	reader  *Reader
	writer  *Writer

	// 프로토콜 제어
	windowAckSize uint32
	peerBandwidth uint32
	lastAckSent   uint64
}

// NewTransport creates a new Transport
func NewTransport(conn net.Conn) *Transport {
	mc := newMeteredConn(conn)
	return &Transport{
		netConn:       conn,
		conn:          mc,
		reader:        NewReader(mc),
		writer:        NewWriter(mc),
		windowAckSize: 0,
	}
}

// ReadMessage reads a message and handles protocol control automatically
func (t *Transport) ReadMessage() (*Message, error) {
	msg, err := t.reader.ReadMessage()
	if err != nil {
		return nil, err
	}

	if err := t.handleProtocolControl(msg); err != nil {
		return nil, err
	}

	if err := t.handleAckWindow(); err != nil {
		return nil, err
	}

	return msg, nil
}

// WriteMessage writes a message with automatic flush
func (t *Transport) WriteMessage(msg *Message) error {
	if err := t.writer.WriteMessage(msg); err != nil {
		return err
	}

	// 자동 Flush
	return t.writer.Flush()
}

// handleProtocolControl handles protocol control messages
func (t *Transport) handleProtocolControl(msg *Message) error {
	switch msg.Type() {
	case MsgTypeSetChunkSize:
		if len(msg.Data()) != 4 {
			return fmt.Errorf("invalid SetChunkSize message length")
		}
		size := binary.BigEndian.Uint32(msg.Data()) & 0x7FFFFFFF
		if size > MaxChunkSize {
			return fmt.Errorf("chunk size %d exceeds maximum", size)
		}
		_ = t.reader.SetChunkSize(size)

	case MsgTypeWindowAckSize:
		if len(msg.Data()) != 4 {
			return fmt.Errorf("invalid WindowAckSize message length")
		}
		t.windowAckSize = binary.BigEndian.Uint32(msg.Data())

	case MsgTypeSetPeerBW:
		if len(msg.Data()) != 5 {
			return fmt.Errorf("invalid SetPeerBandwidth message length")
		}
		t.peerBandwidth = binary.BigEndian.Uint32(msg.Data())
	}

	return nil
}

// handleAckWindow sends acknowledgement if needed based on windowAckSize
func (t *Transport) handleAckWindow() error {
	if t.windowAckSize == 0 {
		return nil
	}

	bytesRead := t.conn.BytesRead()

	// Send multiple ACKs if we've crossed multiple window boundaries
	// This handles cases where a large message spans multiple windows
	for bytesRead-t.lastAckSent >= uint64(t.windowAckSize) {
		t.lastAckSent += uint64(t.windowAckSize)
		if err := t.sendAcknowledgement(t.lastAckSent); err != nil {
			return fmt.Errorf("send acknowledgement: %w", err)
		}
	}

	return nil
}

// sendAcknowledgement sends an acknowledgement message
func (t *Transport) sendAcknowledgement(bytesRead uint64) error {
	// RTMP ACK message uses uint32 (4 bytes)
	ackBytes := uint32(bytesRead) // uint64 to uint32, wrap-around is expected

	// Create 4-byte payload
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, ackBytes)

	// Create ACK message (StreamID=0, Timestamp=0, TypeID=Acknowledgement)
	msg := NewMessage(0, 0, MsgTypeAcknowledgement, payload)

	// Send message
	return t.WriteMessage(msg)
}

// SetInChunkSize sets the incoming chunk size
func (t *Transport) SetInChunkSize(size uint32) error {
	return t.reader.SetChunkSize(size)
}

// SetOutChunkSize sets the outgoing chunk size
func (t *Transport) SetOutChunkSize(size uint32) error {
	return t.writer.SetChunkSize(size)
}

// Close closes the transport
func (t *Transport) Close() error {
	return t.netConn.Close()
}
