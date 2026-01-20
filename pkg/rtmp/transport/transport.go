package transport

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ssungk/ertmp/pkg/rtmp/buf"
)

// Transport represents a bidirectional RTMP protocol handler
type Transport struct {
	conn   *meteredConn
	reader *Reader
	writer *Writer

	// 프로토콜 제어
	windowAckSize uint32
	peerBandwidth uint32
	lastAckSent   uint64
}

// NewTransport creates a new Transport
func NewTransport(rwc io.ReadWriteCloser) *Transport {
	mc := newMeteredConn(rwc)
	return &Transport{
		conn:          mc,
		reader:        NewReader(mc),
		writer:        NewWriter(mc),
		windowAckSize: 0,
	}
}

// Close closes the transport
func (t *Transport) Close() error {
	return t.conn.Close()
}

// ReadMessage reads a message and handles protocol control automatically
func (t *Transport) ReadMessage() (Message, error) {
	msg, err := t.reader.ReadMessage()
	if err != nil {
		return Message{}, err
	}

	if err := t.handleProtocolControl(msg); err != nil {
		msg.Buffer().Release()
		return Message{}, err
	}

	if err := t.handleAckWindow(); err != nil {
		msg.Buffer().Release()
		return Message{}, err
	}

	return msg, nil
}

// WriteMessage writes a message with automatic flush
func (t *Transport) WriteMessage(msg Message) error {
	if err := t.writer.WriteMessage(msg); err != nil {
		return err
	}

	// 자동 Flush
	return t.writer.Flush()
}

// handleProtocolControl handles protocol control messages
func (t *Transport) handleProtocolControl(msg Message) error {
	switch msg.Type() {
	case MsgTypeSetChunkSize:
		if len(msg.Data()) != 4 {
			return fmt.Errorf("invalid SetChunkSize message length: expected 4, got %d", len(msg.Data()))
		}
		size := binary.BigEndian.Uint32(msg.Data()) & ChunkSizeMsgMask

		return t.reader.SetChunkSize(size)
	case MsgTypeAbort:
		if len(msg.Data()) != 4 {
			return fmt.Errorf("invalid Abort message length: expected 4, got %d", len(msg.Data()))
		}
		csid := binary.BigEndian.Uint32(msg.Data())
		t.reader.ClearChunkStream(csid)
	case MsgTypeAcknowledgement:
		if len(msg.Data()) != 4 {
			return fmt.Errorf("invalid Acknowledgement message length: expected 4, got %d", len(msg.Data()))
		}
		// TODO: 상대방 ACK 추적하여 송신 flow control 구현 필요 (선택적)
		// Acknowledgement는 상대방이 받은 바이트 수를 알려줌
	case MsgTypeUserControl:
		if len(msg.Data()) < 2 {
			return fmt.Errorf("invalid UserControl message length: expected >= 2, got %d", len(msg.Data()))
		}

		eventType, eventData, err := parseUserControl(msg.Data())
		if err != nil {
			return err
		}

		// PingRequest 자동 응답
		if eventType == UserControlPingRequest {
			if len(eventData) != 4 {
				return fmt.Errorf("invalid PingRequest data length: expected 4, got %d", len(eventData))
			}

			// PingResponse 전송 (동일한 timestamp)
			pongMsg := createUserControl(UserControlPingResponse, eventData)
			defer pongMsg.Buffer().Release()

			if err := t.WriteMessage(pongMsg); err != nil {
				return fmt.Errorf("send PingResponse: %w", err)
			}
		}
		// 다른 UserControl 이벤트는 무시 (StreamBegin, StreamEOF 등)
	case MsgTypeWindowAckSize:
		if len(msg.Data()) != 4 {
			return fmt.Errorf("invalid WindowAckSize message length: expected 4, got %d", len(msg.Data()))
		}
		newWindowSize := binary.BigEndian.Uint32(msg.Data())
		if t.windowAckSize != newWindowSize {
			t.lastAckSent = t.conn.BytesRead()
		}
		t.windowAckSize = newWindowSize
	case MsgTypeSetPeerBW:
		if len(msg.Data()) != 5 {
			return fmt.Errorf("invalid SetPeerBandwidth message length: expected 5, got %d", len(msg.Data()))
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
	buffer := buf.NewFromPool(4)
	binary.BigEndian.PutUint32(buffer.Data(), ackBytes)

	// Create ACK message (StreamID=0, Timestamp=0, TypeID=Acknowledgement)
	header := NewMessageHeader(0, 0, MsgTypeAcknowledgement)
	msg := NewMessage(header, buffer)
	defer msg.Buffer().Release()

	// Send message
	return t.WriteMessage(msg)
}

// SetOutChunkSize sets the outgoing chunk size and sends SetChunkSize message
func (t *Transport) SetOutChunkSize(size uint32) error {
	// Validate chunk size
	if size < 1 || size > MaxChunkSize {
		return fmt.Errorf("invalid chunk size: %d (must be 1-%d)", size, MaxChunkSize)
	}

	// Create 4-byte payload (MSB must be 0, so mask with 0x7FFFFFFF)
	buffer := buf.NewFromPool(4)
	binary.BigEndian.PutUint32(buffer.Data(), size&ChunkSizeMsgMask)

	header := NewMessageHeader(0, 0, MsgTypeSetChunkSize)
	msg := NewMessage(header, buffer)
	defer msg.Buffer().Release()

	// Send message
	if err := t.WriteMessage(msg); err != nil {
		return fmt.Errorf("send SetChunkSize: %w", err)
	}

	// Update local outgoing chunk size
	return t.writer.SetChunkSize(size)
}

// SetWindowAckSize sets the window acknowledgement size and sends WindowAckSize message
func (t *Transport) SetWindowAckSize(size uint32) error {
	// Create 4-byte payload
	buffer := buf.NewFromPool(4)
	binary.BigEndian.PutUint32(buffer.Data(), size)

	header := NewMessageHeader(0, 0, MsgTypeWindowAckSize)
	msg := NewMessage(header, buffer)
	defer msg.Buffer().Release()

	// Send message
	if err := t.WriteMessage(msg); err != nil {
		return fmt.Errorf("send WindowAckSize: %w", err)
	}

	// Update local window ack size (no state needed on send side)
	return nil
}

// SetPeerBandwidth sets the peer bandwidth and sends SetPeerBandwidth message
func (t *Transport) SetPeerBandwidth(size uint32, limitType uint8) error {
	// Create 5-byte payload (4 bytes size + 1 byte limit type)
	buffer := buf.NewFromPool(5)
	binary.BigEndian.PutUint32(buffer.Data(), size)
	buffer.Data()[4] = limitType

	header := NewMessageHeader(0, 0, MsgTypeSetPeerBW)
	msg := NewMessage(header, buffer)
	defer msg.Buffer().Release()

	// Send message
	if err := t.WriteMessage(msg); err != nil {
		return fmt.Errorf("send SetPeerBandwidth: %w", err)
	}

	// Update local peer bandwidth (no state needed on send side)
	return nil
}

// parseUserControl parses UserControl message data
func parseUserControl(data []byte) (eventType uint16, eventData []byte, err error) {
	if len(data) < 2 {
		return 0, nil, fmt.Errorf("UserControl message too short: expected >= 2, got %d", len(data))
	}
	eventType = binary.BigEndian.Uint16(data[0:2])
	eventData = data[2:]
	return
}

// createUserControl creates a UserControl message
func createUserControl(eventType uint16, eventData []byte) Message {
	data := make([]byte, 2+len(eventData))
	binary.BigEndian.PutUint16(data[0:2], eventType)
	copy(data[2:], eventData)

	buffer := buf.New(data)
	header := NewMessageHeader(0, 0, MsgTypeUserControl)
	return NewMessage(header, buffer)
}
