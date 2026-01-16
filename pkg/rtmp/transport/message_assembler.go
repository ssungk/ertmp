package transport

import "github.com/ssungk/ertmp/pkg/rtmp/buf"

// MessageAssembler represents a message assembler that reconstructs messages from chunks
type MessageAssembler struct {
	messageHeader MessageHeader
	prevHeader    MessageHeader
	buffer        *buf.Buffer
	bytesRead     uint32
}

// newMessageAssembler creates a new message assembler
func newMessageAssembler() *MessageAssembler {
	return &MessageAssembler{}
}

// startNewMessage initializes a new message with header and allocates buffer
func (ma *MessageAssembler) startNewMessage(header MessageHeader) {
	ma.messageHeader = header
	ma.buffer = buf.NewFromPool(int(header.MessageLength))
}

// nextBuffer returns a buffer slice for the next chunk data
func (ma *MessageAssembler) nextBuffer(size uint32) []byte {
	return ma.buffer.Data()[ma.bytesRead : ma.bytesRead+size]
}

// moveBuffer moves buffer ownership to caller and resets message assembler
func (ma *MessageAssembler) moveBuffer() *buf.Buffer {
	buffer := ma.buffer
	ma.buffer = nil
	ma.bytesRead = 0
	return buffer
}

// isComplete checks if the message is complete
func (ma *MessageAssembler) isComplete() bool {
	return ma.bytesRead >= ma.messageHeader.MessageLength
}

// clear releases buffer and resets state
func (ma *MessageAssembler) clear() {
	if ma.buffer != nil {
		ma.buffer.Release()
		ma.buffer = nil
	}
	ma.bytesRead = 0
}
