package transport

import (
	"crypto/rand"
	"fmt"
	"io"
)

// ClientHandshake performs client-side RTMP handshake
func ClientHandshake(rw io.ReadWriter) error {
	// Send C0
	c0 := []byte{RTMPVersion}
	if _, err := rw.Write(c0); err != nil {
		return fmt.Errorf("handshake c0: %w: %w", ErrRtmpWrite, err)
	}

	// Send C1 (time + zero + random bytes)
	c1 := make([]byte, HandshakeSize)
	// First 4 bytes: time (epoch seconds)
	// Note: Using 0 is valid for simple handshake
	// Second 4 bytes: zero
	// Remaining 1528 bytes: random
	_, _ = rand.Read(c1[8:])
	if _, err := rw.Write(c1); err != nil {
		return fmt.Errorf("handshake c1: %w: %w", ErrRtmpWrite, err)
	}

	// Read S0
	s0 := make([]byte, 1)
	if _, err := io.ReadFull(rw, s0); err != nil {
		return fmt.Errorf("handshake s0: %w: %w", ErrRtmpRead, err)
	}

	if s0[0] != RTMPVersion {
		return fmt.Errorf("handshake s0 version: got %d, want %d: %w", s0[0], RTMPVersion, ErrUnsupportedVersion)
	}

	// Read S1 and save for C2
	s1 := make([]byte, HandshakeSize)
	if _, err := io.ReadFull(rw, s1); err != nil {
		return fmt.Errorf("handshake s1: %w: %w", ErrRtmpRead, err)
	}

	// Read S2 (reuse c1 buffer)
	s2 := c1
	if _, err := io.ReadFull(rw, s2); err != nil {
		return fmt.Errorf("handshake s2: %w: %w", ErrRtmpRead, err)
	}

	// Send C2 (echo S1)
	c2 := s1
	if _, err := rw.Write(c2); err != nil {
		return fmt.Errorf("handshake c2: %w: %w", ErrRtmpWrite, err)
	}

	return nil
}

// ServerHandshake performs server-side RTMP handshake
func ServerHandshake(rw io.ReadWriter) error {
	// Read C0
	c0 := make([]byte, 1)
	if _, err := io.ReadFull(rw, c0); err != nil {
		return fmt.Errorf("handshake c0: %w: %w", ErrRtmpRead, err)
	}

	if c0[0] != RTMPVersion {
		return fmt.Errorf("handshake c0 version: got %d, want %d: %w", c0[0], RTMPVersion, ErrUnsupportedVersion)
	}

	// Read C1 and save for S2
	c1 := make([]byte, HandshakeSize)
	if _, err := io.ReadFull(rw, c1); err != nil {
		return fmt.Errorf("handshake c1: %w: %w", ErrRtmpRead, err)
	}

	// Send S0 (reuse c0 buffer)
	s0 := c0
	if _, err := rw.Write(s0); err != nil {
		return fmt.Errorf("handshake s0: %w: %w", ErrRtmpWrite, err)
	}

	// Send S1 (time + zero + random bytes)
	s1 := make([]byte, HandshakeSize)
	// First 4 bytes: time (epoch seconds)
	// Note: Using 0 is valid for simple handshake
	// Second 4 bytes: zero
	// Remaining 1528 bytes: random
	_, _ = rand.Read(s1[8:])
	if _, err := rw.Write(s1); err != nil {
		return fmt.Errorf("handshake s1: %w: %w", ErrRtmpWrite, err)
	}

	// Send S2 (echo C1)
	s2 := c1
	if _, err := rw.Write(s2); err != nil {
		return fmt.Errorf("handshake s2: %w: %w", ErrRtmpWrite, err)
	}

	// Read C2 (reuse s1 buffer)
	c2 := s1
	if _, err := io.ReadFull(rw, c2); err != nil {
		return fmt.Errorf("handshake c2: %w: %w", ErrRtmpRead, err)
	}

	return nil
}
