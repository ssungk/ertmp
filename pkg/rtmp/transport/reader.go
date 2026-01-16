package transport

import (
	"fmt"
	"io"
)

// Reader reads RTMP messages from a stream
type Reader struct {
	conn       *meteredConn
	assemblers map[uint32]*MessageAssembler
	chunkSize  uint32
}

// NewReader creates a new RTMP reader
func NewReader(mc *meteredConn) *Reader {
	return &Reader{
		conn:       mc,
		assemblers: make(map[uint32]*MessageAssembler),
		chunkSize:  DefaultChunkSize,
	}
}

// ReadMessage reads a complete RTMP message
func (r *Reader) ReadMessage() (*Message, error) {
	for {
		csid, err := r.readChunk()
		if err != nil {
			return nil, err
		}

		if msg := r.getReadyMessage(csid); msg != nil {
			return msg, nil
		}
	}
}

// readChunk reads a single chunk and accumulates data in message assemblers
func (r *Reader) readChunk() (uint32, error) {
	// Read basic header
	basicHeader, err := readBasicHeader(r.conn)
	if err != nil {
		return 0, fmt.Errorf("chunk basic header: %w: %w", ErrRtmpRead, err)
	}

	// 메시지 어셈블러 획득 또는 생성
	ma, ok := r.assemblers[basicHeader.chunkStreamID]
	if !ok {
		ma = newMessageAssembler()
		r.assemblers[basicHeader.chunkStreamID] = ma
	}

	// fmt에 따라 메시지 헤더 읽기
	msgHeader, err := readMessageHeader(r.conn, basicHeader.fmt, &ma.prevHeader)
	if err != nil {
		return 0, fmt.Errorf("chunk message header: %w: %w", ErrRtmpRead, err)
	}

	// 새 메시지 시작: 헤더 갱신 및 버퍼 할당
	if ma.bytesRead == 0 {
		ma.startNewMessage(msgHeader)
	}

	// 청크 데이터 크기 계산
	remainingBytes := ma.messageHeader.MessageLength - ma.bytesRead
	chunkDataSize := min(r.chunkSize, remainingBytes)

	// 청크 데이터를 메시지 버퍼에 직접 읽기
	if _, err := io.ReadFull(r.conn, ma.nextBuffer(chunkDataSize)); err != nil {
		return 0, fmt.Errorf("chunk data: %w: %w", ErrRtmpRead, err)
	}

	// 읽은 바이트 수 업데이트
	ma.bytesRead += chunkDataSize

	// 이전 헤더 업데이트 (다음 청크를 위해, FmtType3이 참조함)
	ma.prevHeader = msgHeader

	// 청크 스트림 ID 반환
	return basicHeader.chunkStreamID, nil
}

// getReadyMessage returns a completed message if available
func (r *Reader) getReadyMessage(csid uint32) *Message {
	// 완성된 메시지가 있는지 확인
	ma := r.assemblers[csid]
	if ma == nil || !ma.isComplete() {
		return nil
	}

	// 완성된 메시지 생성 (버퍼 소유권 이전)
	buffer := ma.moveBuffer()
	msg := NewMessageFromBuffer(ma.messageHeader, buffer)

	// 이전 헤더 업데이트 (다음 메시지를 위해)
	ma.prevHeader = ma.messageHeader

	return msg
}

// SetChunkSize sets the chunk size for reading
func (r *Reader) SetChunkSize(size uint32) error {
	if size > MaxChunkSize {
		return fmt.Errorf("chunk size %d exceeds maximum %d", size, MaxChunkSize)
	}
	if size < 1 {
		return fmt.Errorf("chunk size must be at least 1")
	}
	r.chunkSize = size
	return nil
}

// ClearChunkStream clears partially received message for a chunk stream
func (r *Reader) ClearChunkStream(csid uint32) {
	ma := r.assemblers[csid]
	if ma == nil {
		return
	}

	ma.clear()
}
