package transport

import (
	"fmt"
)

// Writer writes RTMP messages to a stream
type Writer struct {
	conn         *meteredConn
	chunkStreams map[uint32]*ChunkStream
	chunkSize    uint32
}

// NewWriter creates a new RTMP writer
func NewWriter(mc *meteredConn) *Writer {
	return &Writer{
		conn:         mc,
		chunkStreams: make(map[uint32]*ChunkStream),
		chunkSize:    DefaultChunkSize,
	}
}

// SetChunkSize sets the chunk size for writing
func (w *Writer) SetChunkSize(size uint32) error {
	if size > MaxChunkSize {
		return fmt.Errorf("chunk size %d exceeds maximum %d", size, MaxChunkSize)
	}
	if size < 1 {
		return fmt.Errorf("chunk size must be at least 1")
	}
	w.chunkSize = size
	return nil
}

// WriteMessage writes a complete RTMP message
func (w *Writer) WriteMessage(msg *Message) error {
	// 청크 스트림 획득 또는 생성
	csid := w.getChunkStreamID(msg.Header.MessageTypeID)
	cs := w.getChunkStream(csid)

	// 포맷 타입 결정
	fmtType := w.determineFormatType(cs.PrevHeader, msg.Header)

	// 메시지 데이터 획득
	data := msg.Data()
	if data == nil {
		data = []byte{}
	}

	// 청크 단위로 메시지 작성
	totalBytes := uint32(len(data))
	bytesWritten := uint32(0)
	isFirstChunk := true

	for bytesWritten < totalBytes {
		// 청크 크기 계산
		remainingBytes := totalBytes - bytesWritten
		chunkDataSize := w.chunkSize
		if remainingBytes < chunkDataSize {
			chunkDataSize = remainingBytes
		}

		// 청크 헤더 작성
		if isFirstChunk {
			// 기본 헤더 작성
			basicHeader := newBasicHeader(fmtType, csid)
			if _, err := basicHeader.WriteTo(w.conn); err != nil {
				return fmt.Errorf("chunk basic header: %w: %w", ErrRtmpWrite, err)
			}

			// 메시지 헤더 작성
			if _, err := msg.Header.WriteTo(w.conn, fmtType); err != nil {
				return fmt.Errorf("chunk message header: %w: %w", ErrRtmpWrite, err)
			}

			isFirstChunk = false
		} else {
			// 연속 헤더 작성 (fmt 3)
			basicHeader := newBasicHeader(FmtType3, csid)
			if _, err := basicHeader.WriteTo(w.conn); err != nil {
				return fmt.Errorf("chunk continuation header: %w: %w", ErrRtmpWrite, err)
			}
		}

		// 청크 데이터 작성
		chunkData := data[bytesWritten : bytesWritten+chunkDataSize]
		if _, err := w.conn.Write(chunkData); err != nil {
			return fmt.Errorf("chunk data: %w: %w", ErrRtmpWrite, err)
		}

		bytesWritten += chunkDataSize
	}

	// 이전 헤더 업데이트
	cs.PrevHeader = msg.Header

	return nil
}

// Flush flushes the writer
func (w *Writer) Flush() error {
	return w.conn.Flush()
}

// BytesWritten returns the total number of bytes written to the socket
func (w *Writer) BytesWritten() uint64 {
	return w.conn.BytesWritten()
}

// determineFormatType determines the optimal format type
func (w *Writer) determineFormatType(prevHeader, currHeader MessageHeader) uint8 {
	if prevHeader.MessageStreamID != currHeader.MessageStreamID {
		return FmtType0 // 전체 헤더 필요
	}
	if prevHeader.MessageLength != currHeader.MessageLength ||
		prevHeader.MessageTypeID != currHeader.MessageTypeID {
		return FmtType1 // 타입 또는 길이 변경됨
	}
	if prevHeader.Timestamp != currHeader.Timestamp {
		return FmtType2 // 타임스탬프만 변경됨
	}
	return FmtType3 // 변경사항 없음 (첫 청크에서는 발생하지 않아야 함)
}

// getChunkStreamID returns the appropriate chunk stream ID for a message type
func (w *Writer) getChunkStreamID(msgType uint8) uint32 {
	switch msgType {
	case MsgTypeSetChunkSize, MsgTypeAbort, MsgTypeAcknowledgement,
		MsgTypeWindowAckSize, MsgTypeSetPeerBW, MsgTypeUserControl:
		return ChunkStreamProtocol
	case MsgTypeAMF0Command, MsgTypeAMF3Command:
		return ChunkStreamCommand
	case MsgTypeAudio:
		return ChunkStreamAudio
	case MsgTypeVideo:
		return ChunkStreamVideo
	case MsgTypeAMF0Data, MsgTypeAMF3Data:
		return ChunkStreamData
	default:
		return ChunkStreamCommand
	}
}

// getChunkStream gets or creates a chunk stream
func (w *Writer) getChunkStream(id uint32) *ChunkStream {
	cs, ok := w.chunkStreams[id]
	if !ok {
		cs = NewChunkStream()
		w.chunkStreams[id] = cs
	}
	return cs
}

// WriteByte writes a single byte
func (w *Writer) WriteByte(b byte) error {
	return w.conn.WriteByte(b)
}

// Write writes data
func (w *Writer) Write(p []byte) (int, error) {
	return w.conn.Write(p)
}
