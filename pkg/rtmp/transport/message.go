package transport

import "github.com/ssungk/ertmp/pkg/rtmp/buf"

// Message represents an RTMP message with reference-counted buffer
type Message struct {
	Header MessageHeader
	buffer *buf.Buffer
}

// NewMessage creates a new message with data (copies data into pooled buffer)
func NewMessage(header MessageHeader, data []byte) *Message {
	header.MessageLength = uint32(len(data))

	// Allocate buffer from pool
	buffer := buf.NewPooled(len(data))
	copy(buffer.Data(), data)

	return &Message{
		Header: header,
		buffer: buffer,
	}
}

// NewMessageFromBuffer creates a message from existing buffer (zero-copy)
func NewMessageFromBuffer(header MessageHeader, buffer *buf.Buffer) *Message {
	header.MessageLength = uint32(buffer.Len())

	return &Message{
		Header: header,
		buffer: buffer,
	}
}

// Data returns the payload bytes
func (m *Message) Data() []byte {
	if m.buffer == nil {
		return nil
	}
	data := m.buffer.Data()
	if uint32(len(data)) > m.Header.MessageLength {
		return data[:m.Header.MessageLength]
	}
	return data
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

// Retain increments the reference count
func (m *Message) Retain() {
	if m.buffer != nil {
		m.buffer.Retain()
	}
}

// Share creates a new message sharing the same buffer with different streamID (zero-copy)
func (m *Message) Share(streamID uint32) *Message {
	// Increment reference count on buffer
	if m.buffer != nil {
		m.buffer.Retain()
	}

	header := NewMessageHeader(streamID, m.Header.Timestamp, m.Header.MessageTypeID)
	header.MessageLength = m.Header.MessageLength

	return &Message{
		Header: header,
		buffer: m.buffer, // Share same buffer pointer
	}
}

// Release releases buffer back to pool
func (m *Message) Release() {
	if m.buffer != nil {
		m.buffer.Release()
	}
}
