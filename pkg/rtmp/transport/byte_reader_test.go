package transport

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestByteReaderFunctions(t *testing.T) {
	// ========== readUint16LE ==========

	// Success case
	testReadUint16LE(t, []byte{0x12, 0x34}, 0x3412, nil)
	testReadUint16LE(t, []byte{0xFF, 0xFF}, 0xFFFF, nil)

	// Error cases: EOF at each byte position
	testReadUint16LE(t, []byte{}, 0, io.EOF)     // b0 fails
	testReadUint16LE(t, []byte{0x12}, 0, io.EOF) // b1 fails

	// ========== readUint16BE ==========

	// Success case
	testReadUint16BE(t, []byte{0x12, 0x34}, 0x1234, nil)
	testReadUint16BE(t, []byte{0xFF, 0xFF}, 0xFFFF, nil)

	// Error cases: EOF at each byte position
	testReadUint16BE(t, []byte{}, 0, io.EOF)     // b0 fails
	testReadUint16BE(t, []byte{0x12}, 0, io.EOF) // b1 fails

	// ========== readUint24BE ==========

	// Success case
	testReadUint24BE(t, []byte{0x12, 0x34, 0x56}, 0x123456, nil)
	testReadUint24BE(t, []byte{0xFF, 0xFF, 0xFF}, 0xFFFFFF, nil)

	// Error cases: EOF at each byte position
	testReadUint24BE(t, []byte{}, 0, io.EOF)           // b0 fails
	testReadUint24BE(t, []byte{0x12}, 0, io.EOF)       // b1 fails
	testReadUint24BE(t, []byte{0x12, 0x34}, 0, io.EOF) // b2 fails

	// ========== readUint32BE ==========

	// Success case
	testReadUint32BE(t, []byte{0x12, 0x34, 0x56, 0x78}, 0x12345678, nil)
	testReadUint32BE(t, []byte{0xFF, 0xFF, 0xFF, 0xFF}, 0xFFFFFFFF, nil)

	// Error cases: EOF at each byte position
	testReadUint32BE(t, []byte{}, 0, io.EOF)                 // b0 fails
	testReadUint32BE(t, []byte{0x12}, 0, io.EOF)             // b1 fails
	testReadUint32BE(t, []byte{0x12, 0x34}, 0, io.EOF)       // b2 fails
	testReadUint32BE(t, []byte{0x12, 0x34, 0x56}, 0, io.EOF) // b3 fails

	// ========== readUint32LE ==========

	// Success case
	testReadUint32LE(t, []byte{0x12, 0x34, 0x56, 0x78}, 0x78563412, nil)
	testReadUint32LE(t, []byte{0xFF, 0xFF, 0xFF, 0xFF}, 0xFFFFFFFF, nil)

	// Error cases: EOF at each byte position
	testReadUint32LE(t, []byte{}, 0, io.EOF)                 // b0 fails
	testReadUint32LE(t, []byte{0x12}, 0, io.EOF)             // b1 fails
	testReadUint32LE(t, []byte{0x12, 0x34}, 0, io.EOF)       // b2 fails
	testReadUint32LE(t, []byte{0x12, 0x34, 0x56}, 0, io.EOF) // b3 fails
}

// Helper functions for testing each byte reader function

func testReadUint16LE(t *testing.T, data []byte, want uint16, wantErr error) {
	t.Helper()
	r := bytes.NewReader(data)
	got, err := readUint16LE(r)

	if wantErr != nil {
		if !errors.Is(err, wantErr) {
			t.Errorf("readUint16LE(%v): expected error %v, got %v", data, wantErr, err)
		}
		return
	}

	if err != nil {
		t.Errorf("readUint16LE(%v): unexpected error: %v", data, err)
		return
	}

	if got != want {
		t.Errorf("readUint16LE(%v) = 0x%04X, want 0x%04X", data, got, want)
	}
}

func testReadUint16BE(t *testing.T, data []byte, want uint16, wantErr error) {
	t.Helper()
	r := bytes.NewReader(data)
	got, err := readUint16BE(r)

	if wantErr != nil {
		if !errors.Is(err, wantErr) {
			t.Errorf("readUint16BE(%v): expected error %v, got %v", data, wantErr, err)
		}
		return
	}

	if err != nil {
		t.Errorf("readUint16BE(%v): unexpected error: %v", data, err)
		return
	}

	if got != want {
		t.Errorf("readUint16BE(%v) = 0x%04X, want 0x%04X", data, got, want)
	}
}

func testReadUint24BE(t *testing.T, data []byte, want uint32, wantErr error) {
	t.Helper()
	r := bytes.NewReader(data)
	got, err := readUint24BE(r)

	if wantErr != nil {
		if !errors.Is(err, wantErr) {
			t.Errorf("readUint24BE(%v): expected error %v, got %v", data, wantErr, err)
		}
		return
	}

	if err != nil {
		t.Errorf("readUint24BE(%v): unexpected error: %v", data, err)
		return
	}

	if got != want {
		t.Errorf("readUint24BE(%v) = 0x%06X, want 0x%06X", data, got, want)
	}
}

func testReadUint32BE(t *testing.T, data []byte, want uint32, wantErr error) {
	t.Helper()
	r := bytes.NewReader(data)
	got, err := readUint32BE(r)

	if wantErr != nil {
		if !errors.Is(err, wantErr) {
			t.Errorf("readUint32BE(%v): expected error %v, got %v", data, wantErr, err)
		}
		return
	}

	if err != nil {
		t.Errorf("readUint32BE(%v): unexpected error: %v", data, err)
		return
	}

	if got != want {
		t.Errorf("readUint32BE(%v) = 0x%08X, want 0x%08X", data, got, want)
	}
}

func testReadUint32LE(t *testing.T, data []byte, want uint32, wantErr error) {
	t.Helper()
	r := bytes.NewReader(data)
	got, err := readUint32LE(r)

	if wantErr != nil {
		if !errors.Is(err, wantErr) {
			t.Errorf("readUint32LE(%v): expected error %v, got %v", data, wantErr, err)
		}
		return
	}

	if err != nil {
		t.Errorf("readUint32LE(%v): unexpected error: %v", data, err)
		return
	}

	if got != want {
		t.Errorf("readUint32LE(%v) = 0x%08X, want 0x%08X", data, got, want)
	}
}
