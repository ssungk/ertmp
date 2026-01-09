package transport

import (
	"encoding/binary"
	"fmt"
	"net"
)

// Transport represents a bidirectional RTMP protocol handler
type Transport struct {
	netConn       net.Conn
	conn          *meteredConn
	reader        *Reader
	writer        *Writer

	// 프로토콜 제어
	windowAckSize uint32
	peerBandwidth uint32

	// TODO: Acknowledgement 자동 전송 구현
	// - windowAckSize 기준으로 자동 전송
}

// NewTransport creates a new Transport
func NewTransport(conn net.Conn) *Transport {
	mc := newMeteredConn(conn)
	return &Transport{
		netConn:       conn,
		conn:          mc,
		reader:        NewReader(mc),
		writer:        NewWriter(mc),
		windowAckSize: 2500000, // 기본 2.5MB
	}
}

// ReadMessage reads a message and handles protocol control automatically
func (t *Transport) ReadMessage() (*Message, error) {
	msg, err := t.reader.ReadMessage()
	if err != nil {
		return nil, err
	}

	// 프로토콜 제어 메시지 자동 처리
	if err := t.handleProtocolControl(msg); err != nil {
		return nil, err
	}

	// TODO: Acknowledgement 자동 전송
	// bytesRead := t.conn.BytesRead()
	// if t.windowAckSize > 0 && bytesRead-t.lastAckSent >= t.windowAckSize {
	//     sendAcknowledgement(bytesRead)
	// }

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

// TODO: sendAcknowledgement 구현
// - windowAckSize 기준으로 자동 전송
// - t.conn.BytesRead() 사용

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
