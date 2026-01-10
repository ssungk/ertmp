package transport

import (
	"errors"
	"fmt"
	"io"
	"sync/atomic"
)

// Reader reads RTMP messages from a stream
type Reader struct {
	conn         *meteredConn
	chunkStreams map[uint32]*ChunkStream
	chunkSize    uint32
}

// NewReader creates a new RTMP reader
func NewReader(mc *meteredConn) *Reader {
	return &Reader{
		conn:         mc,
		chunkStreams: make(map[uint32]*ChunkStream),
		chunkSize:    DefaultChunkSize,
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

// readChunk reads a single chunk and accumulates data in chunk streams
func (r *Reader) readChunk() (uint32, error) {
	// Read basic header
	basicHeader, err := readBasicHeader(r.conn)
	if err != nil {
		return 0, fmt.Errorf("chunk basic header: %w: %w", ErrRtmpRead, err)
	}

	// 청크 스트림 획득 또는 생성
	cs := r.getChunkStream(basicHeader.chunkStreamID)

	// fmt에 따라 메시지 헤더 읽기
	msgHeader, err := readMessageHeader(r.conn, basicHeader.fmt, &cs.PrevHeader)
	if err != nil {
		return 0, fmt.Errorf("chunk message header: %w: %w", ErrRtmpRead, err)
	}

	// 새 메시지 시작: 헤더 갱신
	if cs.BytesRead == 0 {
		cs.MessageHeader = msgHeader
	}

	// 청크 데이터 크기 계산
	remainingBytes := cs.MessageHeader.MessageLength - cs.BytesRead
	chunkDataSize := r.chunkSize
	if remainingBytes < chunkDataSize {
		chunkDataSize = remainingBytes
	}

	// 청크 데이터 읽기 (버퍼 풀 사용, 제로 카피)
	buf, err := ReadChunkData(r.conn, int(chunkDataSize))
	if err != nil {
		return 0, fmt.Errorf("chunk data: %w: %w", ErrRtmpRead, err)
	}

	// 메시지 버퍼에 추가 (복사 없이 버퍼 참조만 저장)
	cs.AppendBuffer(buf)

	// 청크 스트림 ID 반환
	return basicHeader.chunkStreamID, nil
}

// getReadyMessage returns a completed message if available
func (r *Reader) getReadyMessage(csid uint32) *Message {
	// 청크 스트림에 완성된 메시지가 있는지 확인
	cs := r.chunkStreams[csid]
	if cs == nil || !cs.IsComplete() {
		return nil
	}

	// 완성된 청크 스트림에서 메시지 생성 (zero-copy)
	refCount := &atomic.Int32{}
	refCount.Store(1)
	msg := &Message{
		Header:   cs.MessageHeader,
		buffers:  cs.MoveBuffers(),
		refCount: refCount,
	}

	// 이전 헤더 업데이트 (다음 메시지를 위해)
	cs.PrevHeader = cs.MessageHeader

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

// getChunkStream gets or creates a chunk stream
func (r *Reader) getChunkStream(id uint32) *ChunkStream {
	cs, ok := r.chunkStreams[id]
	if !ok {
		cs = NewChunkStream()
		r.chunkStreams[id] = cs
	}
	return cs
}

// BytesRead returns the total number of bytes read from the socket
func (r *Reader) BytesRead() uint64 {
	return r.conn.BytesRead()
}

// ReadByte reads a single byte
func (r *Reader) ReadByte() (byte, error) {
	return r.conn.ReadByte()
}

// Read reads data into a buffer
func (r *Reader) Read(p []byte) (int, error) {
	return r.conn.Read(p)
}

// ReadFull reads exactly len(p) bytes
func (r *Reader) ReadFull(p []byte) error {
	_, err := io.ReadFull(r.conn, p)
	return err
}

// ReadChunkData reads chunk data using buffer pool ([]byte returned)
func ReadChunkData(reader io.Reader, size int) ([]byte, error) {
	if reader == nil {
		return nil, errors.New("reader is nil")
	}

	if size <= 0 {
		return nil, fmt.Errorf("invalid size: %d (must be positive)", size)
	}

	buf := GetBuffer(size)
	_, err := io.ReadFull(reader, buf)
	if err != nil {
		PutBuffer(buf)
		return nil, fmt.Errorf("read %d bytes: %w: %w", size, ErrRtmpRead, err)
	}

	return buf, nil
}
