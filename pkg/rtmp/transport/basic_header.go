package transport

import (
	"io"
)

// basicHeader represents the basic header of a chunk
type basicHeader struct {
	fmt           uint8
	chunkStreamID uint32
}

// newBasicHeader creates a new basic header
func newBasicHeader(fmt uint8, chunkStreamID uint32) basicHeader {
	return basicHeader{
		fmt:           fmt,
		chunkStreamID: chunkStreamID,
	}
}

// WriteTo writes the basic header to writer
func (h *basicHeader) WriteTo(w io.ByteWriter) (int, error) {
	if h.chunkStreamID < 64 {
		// 1바이트 헤더: fmt(2bit) + csid(6bit)
		err := w.WriteByte((h.fmt << 6) | byte(h.chunkStreamID))
		if err != nil {
			return 0, err
		}
		return 1, nil

	} else if h.chunkStreamID < 320 {
		// 2바이트 헤더: fmt(2bit) + 0(6bit) + csid-64(8bit)
		if err := w.WriteByte((h.fmt << 6) | 0); err != nil {
			return 0, err
		}
		if err := w.WriteByte(byte(h.chunkStreamID - 64)); err != nil {
			return 1, err
		}
		return 2, nil

	} else {
		// 3바이트 헤더: fmt(2bit) + 1(6bit) + csid-64(16bit little-endian)
		csid := h.chunkStreamID - 64
		if err := w.WriteByte((h.fmt << 6) | 1); err != nil {
			return 0, err
		}
		if err := w.WriteByte(byte(csid & 0xFF)); err != nil {
			return 1, err
		}
		if err := w.WriteByte(byte((csid >> 8) & 0xFF)); err != nil {
			return 2, err
		}
		return 3, nil
	}
}

// readBasicHeader reads a basic header from reader
func readBasicHeader(r io.ByteReader) (bh basicHeader, err error) {
	b, err := r.ReadByte()
	if err != nil {
		return
	}

	bh.fmt = (b >> 6) & 0x03
	csid := b & 0x3F

	switch csid {
	default: // 1-byte basic header (actual csid = 2-63)
		bh.chunkStreamID = uint32(csid)
	case 0: // 2-byte basic header (actual csid = 64-319)
		b, err = r.ReadByte()
		if err != nil {
			return
		}
		bh.chunkStreamID = uint32(b) + 64
	case 1: // 3-byte basic header (actual csid = 64-65599)
		val, err := readUint16LE(r)
		if err != nil {
			return bh, err
		}
		bh.chunkStreamID = uint32(val) + 64
	}

	return
}
