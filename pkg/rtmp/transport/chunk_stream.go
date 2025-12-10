package transport

// ChunkStream represents a chunk stream
type ChunkStream struct {
	MessageHeader MessageHeader
	PrevHeader    MessageHeader
	buffers       [][]byte
	BytesRead     uint32
}

// NewChunkStream creates a new chunk stream
func NewChunkStream() *ChunkStream {
	return &ChunkStream{
		buffers: GetBufferSlice(),
	}
}

// AppendBuffer appends a buffer to the chunk stream (zero-copy)
func (cs *ChunkStream) AppendBuffer(buf []byte) {
	cs.buffers = append(cs.buffers, buf)
	cs.BytesRead += uint32(len(buf))
}

// MoveBuffers moves buffer ownership to caller (zero-copy)
func (cs *ChunkStream) MoveBuffers() [][]byte {
	buffers := cs.buffers
	cs.buffers = GetBufferSlice()
	cs.BytesRead = 0
	return buffers
}

// IsComplete checks if the message is complete
func (cs *ChunkStream) IsComplete() bool {
	return cs.BytesRead >= cs.MessageHeader.MessageLength
}
