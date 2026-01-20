package transport

import "github.com/ssungk/ertmp/pkg/rtmp/buf"

// Message represents an RTMP message with reference-counted buffer
type Message struct {
	Header MessageHeader
	buffer *buf.Buffer
}

// NewMessage creates a message from existing buffer (zero-copy).
// Takes ownership of the buffer - caller must not use buffer after this.
// To share the buffer, call buffer.Retain() before passing it.
func NewMessage(header MessageHeader, buffer *buf.Buffer) Message {
	header.MessageLength = uint32(buffer.Len())

	return Message{
		Header: header,
		buffer: buffer,
	}
}

// Data returns the payload bytes
func (m *Message) Data() []byte {
	return m.buffer.Data()
}

// Type returns the message type ID
func (m *Message) Type() uint8 {
	return m.Header.MessageTypeID
}

// StreamID returns the message stream ID
func (m *Message) StreamID() uint32 {
	return m.Header.MessageStreamID
}

// Timestamp returns the message timestamp
func (m *Message) Timestamp() uint32 {
	return m.Header.Timestamp
}

// Buffer returns the underlying buffer.
// Use buffer.Retain() and buffer.Release() to manage reference counting.
func (m *Message) Buffer() *buf.Buffer {
	return m.buffer
}
