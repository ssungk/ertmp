package rtmp

import (
	"fmt"
	"net"

	"github.com/ssungk/ertmp/pkg/rtmp/transport"
)

// Conn represents an RTMP connection
type Conn struct {
	transport *transport.Transport
	config    Config

	// 스트림 관리
	streams      map[uint32]*Stream
	nextStreamID uint32
}

// AcceptConn accepts a server-side RTMP connection with handshake
func AcceptConn(netConn net.Conn) (*Conn, error) {
	// 서버 핸드셰이크 수행
	if err := transport.ServerHandshake(netConn); err != nil {
		return nil, err
	}
	return newConn(netConn), nil
}

// DialConn creates a client-side RTMP connection with handshake
func DialConn(netConn net.Conn) (*Conn, error) {
	// 클라이언트 핸드셰이크 수행
	if err := transport.ClientHandshake(netConn); err != nil {
		return nil, err
	}
	return newConn(netConn), nil
}

// newConn creates a new RTMP connection without handshake (internal use)
func newConn(netConn net.Conn) *Conn {
	return &Conn{
		transport:    transport.NewTransport(netConn),
		config:       DefaultConfig(),
		streams:      make(map[uint32]*Stream),
		nextStreamID: 1,
	}
}

// Close closes the connection
func (c *Conn) Close() error {
	return c.transport.Close()
}

// createStream creates a new stream and returns it (internal use)
func (c *Conn) createStream() *Stream {
	streamID := c.nextStreamID
	c.nextStreamID++

	stream := NewStream(streamID)
	c.streams[streamID] = stream

	return stream
}

// GetStream returns a stream by ID
func (c *Conn) GetStream(streamID uint32) *Stream {
	return c.streams[streamID]
}

// DeleteStream deletes a stream by ID
func (c *Conn) DeleteStream(streamID uint32) {
	delete(c.streams, streamID)
}

// Streams returns all streams
func (c *Conn) Streams() map[uint32]*Stream {
	return c.streams
}

// ReadMessage reads a message from the connection
// 프로토콜 제어 메시지 (SetChunkSize 등)는 자동으로 내부 처리됨
func (c *Conn) ReadMessage() (*transport.Message, error) {
	return c.transport.ReadMessage()
}

// WriteMessage writes a message to the connection
// Protocol control messages that require state synchronization cannot be sent directly
func (c *Conn) WriteMessage(msg *transport.Message) error {
	// Prevent direct sending of protocol control messages that have dedicated methods
	switch msg.Type() {
	case transport.MsgTypeSetChunkSize:
		return fmt.Errorf("cannot send SetChunkSize directly: use conn.SetChunkSize() instead")
	case transport.MsgTypeWindowAckSize:
		return fmt.Errorf("cannot send WindowAckSize directly: use conn.SetWindowAckSize() instead")
	case transport.MsgTypeSetPeerBW:
		return fmt.Errorf("cannot send SetPeerBandwidth directly: use conn.SetPeerBandwidth() instead")
	case transport.MsgTypeAcknowledgement:
		return fmt.Errorf("Acknowledgement messages are sent automatically by the Transport layer")
	}

	return c.transport.WriteMessage(msg)
}

// SetChunkSize sets the outgoing chunk size
// Sends SetChunkSize message and updates local writer state
func (c *Conn) SetChunkSize(size uint32) error {
	return c.transport.SetOutChunkSize(size)
}

// SetWindowAckSize sets the window acknowledgement size
// Sends WindowAckSize message
func (c *Conn) SetWindowAckSize(size uint32) error {
	return c.transport.SetWindowAckSize(size)
}

// SetPeerBandwidth sets the peer bandwidth
// Sends SetPeerBandwidth message
func (c *Conn) SetPeerBandwidth(size uint32, limitType uint8) error {
	return c.transport.SetPeerBandwidth(size, limitType)
}
