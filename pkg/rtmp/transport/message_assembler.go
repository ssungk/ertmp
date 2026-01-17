package transport

import "github.com/ssungk/ertmp/pkg/rtmp/buf"

// MessageAssembler represents a message assembler that reconstructs messages from chunks
type MessageAssembler struct {
	messageHeader MessageHeader
	buffer        *buf.Buffer
	bytesRead     uint32
}

// newMessageAssembler creates a new message assembler
func newMessageAssembler() *MessageAssembler {
	return &MessageAssembler{}
}

// startMessage initializes a new message with header and allocates buffer
func (ma *MessageAssembler) startMessage(header MessageHeader) {
	ma.messageHeader = header
	ma.buffer = buf.NewFromPool(int(header.MessageLength))
}

// isNewMessage returns true if this is the start of a new message
func (ma *MessageAssembler) isNewMessage() bool {
	return ma.bytesRead == 0
}

// addBytes increments the bytes read counter
func (ma *MessageAssembler) addBytes(size uint32) {
	ma.bytesRead += size
}

// nextBuffer returns a buffer slice for the next chunk data
func (ma *MessageAssembler) nextBuffer(size uint32) []byte {
	return ma.buffer.Data()[ma.bytesRead : ma.bytesRead+size]
}

// remainingBytes returns the number of bytes left to read
func (ma *MessageAssembler) remainingBytes() uint32 {
	return ma.messageHeader.MessageLength - ma.bytesRead
}

// isComplete checks if the message is complete
func (ma *MessageAssembler) isComplete() bool {
	return ma.bytesRead >= ma.messageHeader.MessageLength
}

// moveBuffer moves buffer ownership to caller and resets message assembler
func (ma *MessageAssembler) moveBuffer() *buf.Buffer {
	buffer := ma.buffer
	ma.buffer = nil
	ma.bytesRead = 0
	return buffer
}

// clear releases buffer and resets state
func (ma *MessageAssembler) clear() {
	if ma.buffer != nil {
		ma.buffer.Release()
		ma.buffer = nil
	}
	ma.bytesRead = 0
}

// header returns a pointer to the message header for external reference
func (ma *MessageAssembler) header() *MessageHeader {
	return &ma.messageHeader
}
