package transport

import (
	"io"
)

// ByteReader utility functions for reading fixed-size data efficiently

// readUint16LE reads 2 bytes and returns as uint16 (little endian)
func readUint16LE(r io.ByteReader) (uint16, error) {
	b0, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	b1, err := r.ReadByte()
	if err != nil {
		return 0, err
	}

	return uint16(b0) | uint16(b1)<<8, nil
}

// readUint16BE reads 2 bytes and returns as uint16 (big endian)
func readUint16BE(r io.ByteReader) (uint16, error) {
	b0, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	b1, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	return uint16(b0)<<8 | uint16(b1), nil
}

// readUint24BE reads 3 bytes and returns as uint32 (big endian)
func readUint24BE(r io.ByteReader) (uint32, error) {
	b0, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	b1, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	b2, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	return uint32(b0)<<16 | uint32(b1)<<8 | uint32(b2), nil
}

// readUint32BE reads 4 bytes and returns as uint32 (big endian)
func readUint32BE(r io.ByteReader) (uint32, error) {
	b0, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	b1, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	b2, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	b3, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	return uint32(b0)<<24 | uint32(b1)<<16 | uint32(b2)<<8 | uint32(b3), nil
}

// readUint32LE reads 4 bytes and returns as uint32 (little endian)
func readUint32LE(r io.ByteReader) (uint32, error) {
	b0, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	b1, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	b2, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	b3, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	return uint32(b0) | uint32(b1)<<8 | uint32(b2)<<16 | uint32(b3)<<24, nil
}
