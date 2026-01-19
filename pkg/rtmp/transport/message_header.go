package transport

import (
	"encoding/binary"
	"io"
)

// MessageHeader represents the message header
type MessageHeader struct {
	Timestamp       uint32
	TimestampDelta  uint32
	MessageLength   uint32
	MessageStreamID uint32
	MessageTypeID   uint8
	hasExtTimestamp bool
}

// NewMessageHeader creates a new message header
func NewMessageHeader(streamID, timestamp uint32, typeID uint8) MessageHeader {
	return MessageHeader{
		MessageStreamID: streamID,
		Timestamp:       timestamp,
		MessageTypeID:   typeID,
	}
}

// WriteTo writes the message header to writer based on format type
func (h MessageHeader) WriteTo(w io.Writer, fmtType uint8) (int64, error) {
	switch fmtType {
	case FmtType0:
		// 전체 헤더 (11바이트 + Extended Timestamp 4바이트)
		ts := h.Timestamp
		hasExtTimestamp := ts >= ExtTimestampThreshold
		if hasExtTimestamp {
			ts = ExtTimestampThreshold
		}
		
		data := make([]byte, 11)
		WriteUint24BE(data[0:3], ts)
		WriteUint24BE(data[3:6], h.MessageLength)
		data[6] = h.MessageTypeID
		binary.LittleEndian.PutUint32(data[7:11], h.MessageStreamID)
		n, err := w.Write(data)
		if err != nil {
			return int64(n), err
		}
		
		// Extended Timestamp (4바이트, 필요 시)
		if hasExtTimestamp {
			extTs := make([]byte, 4)
			binary.BigEndian.PutUint32(extTs, h.Timestamp)
			n2, err := w.Write(extTs)
			return int64(n) + int64(n2), err
		}
		
		return int64(n), nil
	
	case FmtType1:
		// 동일한 스트림 ID (7바이트 + Extended Timestamp 4바이트)
		// FmtType1은 Timestamp Delta를 사용
		delta := h.TimestampDelta
		hasExtTimestamp := delta >= ExtTimestampThreshold
		if hasExtTimestamp {
			delta = ExtTimestampThreshold
		}
		
		data := make([]byte, 7)
		WriteUint24BE(data[0:3], delta)
		WriteUint24BE(data[3:6], h.MessageLength)
		data[6] = h.MessageTypeID
		n, err := w.Write(data)
		if err != nil {
			return int64(n), err
		}
		
		// Extended Timestamp (delta, 4바이트, 필요 시)
		if hasExtTimestamp {
			extTs := make([]byte, 4)
			binary.BigEndian.PutUint32(extTs, h.TimestampDelta)
			n2, err := w.Write(extTs)
			return int64(n) + int64(n2), err
		}
		
		return int64(n), nil
	
	case FmtType2:
		// 동일한 길이와 스트림 ID (3바이트 + Extended Timestamp 4바이트)
		// FmtType2는 Timestamp Delta를 사용
		delta := h.TimestampDelta
		hasExtTimestamp := delta >= ExtTimestampThreshold
		if hasExtTimestamp {
			delta = ExtTimestampThreshold
		}
		
		data := make([]byte, 3)
		WriteUint24BE(data[0:3], delta)
		n, err := w.Write(data)
		if err != nil {
			return int64(n), err
		}
		
		// Extended Timestamp (delta, 4바이트, 필요 시)
		if hasExtTimestamp {
			extTs := make([]byte, 4)
			binary.BigEndian.PutUint32(extTs, h.TimestampDelta)
			n2, err := w.Write(extTs)
			return int64(n) + int64(n2), err
		}
		
		return int64(n), nil
	
	case FmtType3:
		// 헤더 없음 (0바이트)
		return 0, nil
	
	default:
		return 0, nil
	}
}

// readMessageHeader reads a message header from reader
func readMessageHeader(r io.ByteReader, fmtType uint8, isFirstChunk bool, prevHeader *MessageHeader) (MessageHeader, error) {
	switch fmtType {
	case FmtType0:
		return readMessageHeaderFmt0(r)
	case FmtType1:
		return readMessageHeaderFmt1(r, isFirstChunk, prevHeader)
	case FmtType2:
		return readMessageHeaderFmt2(r, isFirstChunk, prevHeader)
	case FmtType3:
		return readMessageHeaderFmt3(r, isFirstChunk, prevHeader)
	default:
		return MessageHeader{}, nil
	}
}

// readMessageHeaderFmt0 reads Type 0 message header (11 bytes)
func readMessageHeaderFmt0(r io.ByteReader) (mh MessageHeader, err error) {
	// Timestamp (3 bytes)
	timestamp, err := readUint24BE(r)
	if err != nil {
		return mh, err
	}
	
	// MessageLength (3 bytes)
	mh.MessageLength, err = readUint24BE(r)
	if err != nil {
		return mh, err
	}
	
	// MessageTypeID (1 byte)
	mh.MessageTypeID, err = r.ReadByte()
	if err != nil {
		return mh, err
	}
	
	// MessageStreamID (4 bytes, little endian)
	mh.MessageStreamID, err = readUint32LE(r)
	if err != nil {
		return mh, err
	}
	
	// Extended Timestamp (4 bytes) - read after all header fields
	mh.hasExtTimestamp = hasExtTimestamp(timestamp)
	timestamp, err = readExtTimestamp(r, mh.hasExtTimestamp, timestamp)
	if err != nil {
		return mh, err
	}

	mh.Timestamp = timestamp
	// RTMP Spec: FmtType3 reuses FmtType0's timestamp as delta
	mh.TimestampDelta = timestamp

	return
}

// readMessageHeaderFmt1 reads Type 1 message header (7 bytes)
func readMessageHeaderFmt1(r io.ByteReader, isFirstChunk bool, prevHeader *MessageHeader) (mh MessageHeader, err error) {
	if prevHeader == nil {
		return mh, ErrNoPreviousHeader
	}
	
	// TimestampDelta (3 bytes)
	timestampDelta, err := readUint24BE(r)
	if err != nil {
		return mh, err
	}
	
	// MessageLength (3 bytes)
	mh.MessageLength, err = readUint24BE(r)
	if err != nil {
		return mh, err
	}
	
	// MessageTypeID (1 byte)
	mh.MessageTypeID, err = r.ReadByte()
	if err != nil {
		return mh, err
	}
	
	// Extended Timestamp (4 bytes) - read after all header fields
	mh.hasExtTimestamp = hasExtTimestamp(timestampDelta)
	timestampDelta, err = readExtTimestamp(r, mh.hasExtTimestamp, timestampDelta)
	if err != nil {
		return mh, err
	}

	// Apply delta and calculate timestamp
	applyTimestampDelta(&mh, prevHeader.Timestamp, timestampDelta, isFirstChunk)
	mh.MessageStreamID = prevHeader.MessageStreamID

	return
}

// readMessageHeaderFmt2 reads Type 2 message header (3 bytes)
func readMessageHeaderFmt2(r io.ByteReader, isFirstChunk bool, prevHeader *MessageHeader) (mh MessageHeader, err error) {
	if prevHeader == nil {
		return mh, ErrNoPreviousHeader
	}
	
	// TimestampDelta (3 bytes)
	timestampDelta, err := readUint24BE(r)
	if err != nil {
		return mh, err
	}
	
	// Extended Timestamp (4 bytes) - read after all header fields
	mh.hasExtTimestamp = hasExtTimestamp(timestampDelta)
	timestampDelta, err = readExtTimestamp(r, mh.hasExtTimestamp, timestampDelta)
	if err != nil {
		return mh, err
	}

	// Apply delta and calculate timestamp
	applyTimestampDelta(&mh, prevHeader.Timestamp, timestampDelta, isFirstChunk)
	mh.MessageLength = prevHeader.MessageLength
	mh.MessageTypeID = prevHeader.MessageTypeID
	mh.MessageStreamID = prevHeader.MessageStreamID

	return
}

// readMessageHeaderFmt3 reads Type 3 message header (0 bytes)
func readMessageHeaderFmt3(r io.ByteReader, isFirstChunk bool, prevHeader *MessageHeader) (mh MessageHeader, err error) {
	if prevHeader == nil {
		return mh, ErrNoPreviousHeader
	}
	
	mh.MessageLength = prevHeader.MessageLength
	mh.MessageTypeID = prevHeader.MessageTypeID
	mh.MessageStreamID = prevHeader.MessageStreamID
	mh.hasExtTimestamp = prevHeader.hasExtTimestamp

	// Read extended timestamp if previous chunk used it
	timestampDelta := prevHeader.TimestampDelta
	timestampDelta, err = readExtTimestamp(r, prevHeader.hasExtTimestamp, timestampDelta)
	if err != nil {
		return mh, err
	}

	// Apply delta and calculate timestamp
	applyTimestampDelta(&mh, prevHeader.Timestamp, timestampDelta, isFirstChunk)

	return
}

// hasExtTimestamp checks if timestamp requires extended timestamp
func hasExtTimestamp(timestamp uint32) bool {
	return timestamp == ExtTimestampThreshold
}

// readExtTimestamp reads extended timestamp if needed, otherwise returns timestamp
func readExtTimestamp(r io.ByteReader, hasExtTimestamp bool, timestamp uint32) (uint32, error) {
	if !hasExtTimestamp {
		return timestamp, nil
	}
	return readUint32BE(r)
}

// applyTimestampDelta calculates and sets timestamp based on delta and isFirstChunk
func applyTimestampDelta(mh *MessageHeader, prevTimestamp, delta uint32, isFirstChunk bool) {
	mh.TimestampDelta = delta
	if isFirstChunk {
		mh.Timestamp = prevTimestamp + delta
	} else {
		mh.Timestamp = prevTimestamp
	}
}
