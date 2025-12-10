package transport

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestReadBasicHeader(t *testing.T) {
	// Error: empty data
	checkError(t, []byte{}, io.EOF)
	
	// 1-byte header (csid 2-63)
	checkBasicHeader(t, []byte{2}, 0, 2)
	checkBasicHeader(t, []byte{5}, 0, 5)
	checkBasicHeader(t, []byte{10}, 0, 10)
	checkBasicHeader(t, []byte{63}, 0, 63)
	
	// 2-byte header (csid 64-319)
	checkBasicHeader(t, []byte{0, 0}, 0, 64)    // csid = 0 + 64
	checkBasicHeader(t, []byte{0, 36}, 0, 100)  // csid = 36 + 64
	checkBasicHeader(t, []byte{0, 136}, 0, 200) // csid = 136 + 64
	checkBasicHeader(t, []byte{0, 255}, 0, 319) // csid = 255 + 64
	
	// Error: 2-byte header incomplete
	checkError(t, []byte{0}, io.EOF) // missing second byte
	
	// 3-byte header (csid 320-65599)
	checkBasicHeader(t, []byte{1, 0, 1}, 0, 320)       // (0 | (1 << 8)) + 64 = 320
	checkBasicHeader(t, []byte{1, 180, 1}, 0, 500)     // (180 | (1 << 8)) + 64 = 500
	checkBasicHeader(t, []byte{1, 168, 3}, 0, 1000)    // (168 | (3 << 8)) + 64 = 1000
	checkBasicHeader(t, []byte{1, 255, 255}, 0, 65599) // (255 | (255 << 8)) + 64 = 65599
	
	// Error: 3-byte header incomplete
	checkError(t, []byte{1}, io.EOF)    // missing b0 and b1
	checkError(t, []byte{1, 0}, io.EOF) // missing b1
}

// Helper function to test successful basic header parsing
func checkBasicHeader(t *testing.T, data []byte, wantFmt uint8, wantCsid uint32) {
	t.Helper()
	r := bytes.NewReader(data)
	bh, err := readBasicHeader(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bh.fmt != wantFmt {
		t.Errorf("fmt = %d, want %d", bh.fmt, wantFmt)
	}
	if bh.chunkStreamID != wantCsid {
		t.Errorf("chunkStreamID = %d, want %d", bh.chunkStreamID, wantCsid)
	}
}

// Helper function to test error cases
func checkError(t *testing.T, data []byte, wantErr error) {
	t.Helper()
	r := bytes.NewReader(data)
	_, err := readBasicHeader(r)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}
