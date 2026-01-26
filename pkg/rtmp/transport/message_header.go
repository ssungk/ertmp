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

// writeMessageHeaderToBuffer writes message header to buffer (zero heap allocation)
// Returns number of bytes written
func writeMessageHeaderToBuffer(buf []byte, h MessageHeader, fmtType uint8) int {
	switch fmtType {
	case FmtType0:
		// 11바이트 헤더
		ts := h.Timestamp
		hasExtTimestamp := ts >= ExtTimestampThreshold
		if hasExtTimestamp {
			ts = ExtTimestampThreshold
		}

		WriteUint24BE(buf[0:3], ts)
		WriteUint24BE(buf[3:6], h.MessageLength)
		buf[6] = h.MessageTypeID
		binary.LittleEndian.PutUint32(buf[7:11], h.MessageStreamID)

		n := 11
		// Extended Timestamp (4바이트, 필요 시)
		if hasExtTimestamp {
			binary.BigEndian.PutUint32(buf[11:15], h.Timestamp)
			n += 4
		}
		return n

	case FmtType1:
		// 7바이트 헤더
		delta := h.TimestampDelta
		hasExtTimestamp := delta >= ExtTimestampThreshold
		if hasExtTimestamp {
			delta = ExtTimestampThreshold
		}

		WriteUint24BE(buf[0:3], delta)
		WriteUint24BE(buf[3:6], h.MessageLength)
		buf[6] = h.MessageTypeID

		n := 7
		// Extended Timestamp (delta, 4바이트, 필요 시)
		if hasExtTimestamp {
			binary.BigEndian.PutUint32(buf[7:11], h.TimestampDelta)
			n += 4
		}
		return n

	case FmtType2:
		// 3바이트 헤더
		delta := h.TimestampDelta
		hasExtTimestamp := delta >= ExtTimestampThreshold
		if hasExtTimestamp {
			delta = ExtTimestampThreshold
		}

		WriteUint24BE(buf[0:3], delta)

		n := 3
		// Extended Timestamp (delta, 4바이트, 필요 시)
		if hasExtTimestamp {
			binary.BigEndian.PutUint32(buf[3:7], h.TimestampDelta)
			n += 4
		}
		return n

	case FmtType3:
		// Extended Timestamp만 (헤더 없음)
		if h.hasExtTimestamp {
			binary.BigEndian.PutUint32(buf[0:4], h.TimestampDelta)
			return 4
		}
		return 0

	default:
		return 0
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
