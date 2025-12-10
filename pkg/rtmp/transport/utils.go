package transport

// ReadUint24BE reads a 24-bit big-endian integer
func ReadUint24BE(b []byte) uint32 {
	return uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])
}

// WriteUint24BE writes a 24-bit big-endian integer
func WriteUint24BE(b []byte, v uint32) {
	b[0] = byte((v >> 16) & 0xFF)
	b[1] = byte((v >> 8) & 0xFF)
	b[2] = byte(v & 0xFF)
}
