package transport

import (
	"encoding/binary"
	"fmt"
)

// Writer writes RTMP messages to a stream
type Writer struct {
	conn        *meteredConn
	prevHeaders map[uint32]MessageHeader
	chunkSize   uint32
}

// NewWriter creates a new RTMP writer
func NewWriter(mc *meteredConn) *Writer {
	return &Writer{
		conn:        mc,
		prevHeaders: make(map[uint32]MessageHeader),
		chunkSize:   DefaultChunkSize,
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
	// 청크 스트림 ID 결정
	csid := w.getChunkStreamID(msg.Header.MessageTypeID)

	// 포맷 타입 결정 및 헤더 준비
	prevHeader, exists := w.prevHeaders[csid]
	var fmtType uint8
	var headerToWrite MessageHeader

	if !exists {
		fmtType = FmtType0 // 첫 메시지는 전체 헤더
		headerToWrite = msg.Header
		// FmtType0: TimestampDelta는 Timestamp와 동일 (연속 청크용)
		headerToWrite.TimestampDelta = msg.Header.Timestamp
	} else {
		fmtType = w.determineFormatType(prevHeader, msg.Header)
		headerToWrite = msg.Header

		// Delta 계산 (FmtType1/2에서 사용)
		if fmtType == FmtType1 || fmtType == FmtType2 {
			headerToWrite.TimestampDelta = msg.Header.Timestamp - prevHeader.Timestamp
		} else if fmtType == FmtType0 {
			// FmtType0: TimestampDelta는 Timestamp와 동일 (연속 청크용)
			headerToWrite.TimestampDelta = msg.Header.Timestamp
		}
	}

	// Extended Timestamp 플래그 설정
	if headerToWrite.Timestamp >= ExtendedTimestampThreshold ||
		headerToWrite.TimestampDelta >= ExtendedTimestampThreshold {
		headerToWrite.hasExtendedTimestamp = true
	}

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
			if _, err := headerToWrite.WriteTo(w.conn, fmtType); err != nil {
				return fmt.Errorf("chunk message header: %w: %w", ErrRtmpWrite, err)
			}

			isFirstChunk = false
		} else {
			// 연속 헤더 작성 (fmt 3)
			basicHeader := newBasicHeader(FmtType3, csid)
			if _, err := basicHeader.WriteTo(w.conn); err != nil {
				return fmt.Errorf("chunk continuation header: %w: %w", ErrRtmpWrite, err)
			}

			// Extended Timestamp 처리 (첫 청크가 사용했다면 매 청크마다)
			if headerToWrite.hasExtendedTimestamp {
				extTs := make([]byte, 4)
				binary.BigEndian.PutUint32(extTs, headerToWrite.TimestampDelta)
				if _, err := w.conn.Write(extTs); err != nil {
					return fmt.Errorf("chunk continuation extended timestamp: %w: %w", ErrRtmpWrite, err)
				}
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
	w.prevHeaders[csid] = headerToWrite

	return nil
}

// Flush flushes the writer
func (w *Writer) Flush() error {
	return w.conn.Flush()
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

// WriteByte writes a single byte
func (w *Writer) WriteByte(b byte) error {
	return w.conn.WriteByte(b)
}

// Write writes data
func (w *Writer) Write(p []byte) (int, error) {
	return w.conn.Write(p)
}
