package transport

import "sync/atomic"

// Message represents an RTMP message with reference counting
type Message struct {
	Header   MessageHeader
	buffers  [][]byte
	refCount *atomic.Int32
}

// NewMessage creates a new message with data
func NewMessage(streamID uint32, timestamp uint32, typeID uint8, data []byte) *Message {
	refCount := &atomic.Int32{}
	refCount.Store(1)

	buffers := GetBufferSlice()
	if len(data) > 0 {
		buf := GetBuffer(len(data))
		copy(buf, data)
		buffers = append(buffers, buf[:len(data)])
	}

	header := NewMessageHeader(streamID, timestamp, typeID)
	header.MessageLength = uint32(len(data))

	return &Message{
		Header:   header,
		buffers:  buffers,
		refCount: refCount,
	}
}

// Data returns the payload bytes
func (m *Message) Data() []byte {
	if len(m.buffers) == 0 {
		return nil
	}
	if len(m.buffers) == 1 {
		return m.buffers[0][:m.Header.MessageLength] // 실제 데이터 크기만 반환
	}
	// multiple buffers, merge only when needed
	result := make([]byte, 0, m.Header.MessageLength)
	remaining := m.Header.MessageLength
	for _, buf := range m.buffers {
		if remaining == 0 {
			break
		}
		if uint32(len(buf)) <= remaining {
			result = append(result, buf...)
			remaining -= uint32(len(buf))
		} else {
			result = append(result, buf[:remaining]...)
			break
		}
	}
	return result
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
	if m.refCount != nil {
		m.refCount.Add(1)
	}
}

// Share creates a new message sharing the same buffers with different streamID
func (m *Message) Share(streamID uint32) *Message {
	if m.refCount != nil {
		m.refCount.Add(1)
	}
	header := NewMessageHeader(streamID, m.Header.Timestamp, m.Header.MessageTypeID)
	header.MessageLength = m.Header.MessageLength
	return &Message{
		Header:   header,
		buffers:  m.buffers,
		refCount: m.refCount,
	}
}

// Release releases message resources back to pool
func (m *Message) Release() {
	if m.refCount == nil || m.buffers == nil {
		return
	}

	// refCount 감소
	count := m.refCount.Add(-1)

	// 마지막 참조가 해제되면 버퍼를 풀에 반납
	if count == 0 {
		PutBufferSlice(m.buffers)
		m.buffers = nil
	}
}
