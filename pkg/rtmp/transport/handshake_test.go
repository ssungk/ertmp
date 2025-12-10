package transport

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"testing"
)

const (
	noLimit          = -1
	failImmediately  = 0
	failAfterVersion = 1
	failAfterC0C1    = 1 + HandshakeSize
)

var (
	i0 = []byte{4}                // invalid version
	h0 = []byte{RTMPVersion}      // valid handshake version (C0/S0)
	h1 = makeTestHandshakeData()  // handshake data (C1/S1)
)

func TestClientHandshake(t *testing.T) {
	// Success
	testClientHandshake(t, h0, h1, noLimit, noLimit, nil)

	// Error cases
	testClientHandshake(t, h0, h1, noLimit, failImmediately, ErrWrite)         // C0 write fails
	testClientHandshake(t, h0, h1, noLimit, failAfterVersion, ErrWrite)        // C1 write fails
	testClientHandshake(t, h0, h1, failImmediately, noLimit, ErrRead)          // S0 read fails
	testClientHandshake(t, i0, h1, failAfterVersion, noLimit, ErrUnsupportedVersion) // S0 unsupported version
	testClientHandshake(t, h0, h1, failAfterVersion, noLimit, ErrRead)         // S1 read fails
	testClientHandshake(t, h0, h1, failAfterC0C1, noLimit, ErrRead)            // S2 read fails (readLimit)
	testClientHandshake(t, h0, h1, noLimit, failAfterC0C1, ErrWrite)           // C2 write fails
}

func TestServerHandshake(t *testing.T) {
	// Success
	testServerHandshake(t, h0, h1, noLimit, noLimit, nil)

	// Error cases
	testServerHandshake(t, h0, h1, failImmediately, noLimit, ErrRead)          // C0 read fails
	testServerHandshake(t, i0, h1, failAfterVersion, noLimit, ErrUnsupportedVersion) // C0 unsupported version
	testServerHandshake(t, h0, h1, failAfterVersion, noLimit, ErrRead)         // C1 read fails
	testServerHandshake(t, h0, h1, noLimit, failImmediately, ErrWrite)         // S0 write fails
	testServerHandshake(t, h0, h1, noLimit, failAfterVersion, ErrWrite)        // S1 write fails
	testServerHandshake(t, h0, h1, noLimit, failAfterC0C1, ErrWrite)           // S2 write fails
	testServerHandshake(t, h0, h1, failAfterC0C1, noLimit, ErrRead)            // C2 read fails (readLimit)
}

func testClientHandshake(t *testing.T, s0, s1 []byte, readLimit, writeLimit int, wantErr error) {
	t.Helper()

	rw := newTestReadWriter(s0, s1, readLimit, writeLimit)

	err := ClientHandshake(rw)

	if err != nil {
		if wantErr == nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !errors.Is(err, wantErr) {
			t.Errorf("expected error %v, got: %v", wantErr, err)
		}
		return
	}

	if wantErr != nil {
		t.Fatal("expected error, got nil")
	}
}

func testServerHandshake(t *testing.T, c0, c1 []byte, readLimit, writeLimit int, wantErr error) {
	t.Helper()

	rw := newTestReadWriter(c0, c1, readLimit, writeLimit)

	err := ServerHandshake(rw)

	if err != nil {
		if wantErr == nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !errors.Is(err, wantErr) {
			t.Errorf("expected error %v, got: %v", wantErr, err)
		}
		return
	}

	if wantErr != nil {
		t.Fatal("expected error, got nil")
	}
}

type testReadWriter struct {
	readBuf   *bytes.Buffer
	readBytes int
	readLimit int

	writeBuf   *bytes.Buffer
	writeBytes int
	writeLimit int
}

func newTestReadWriter(h0, h1 []byte, readLimit, writeLimit int) *testReadWriter {
	return &testReadWriter{
		readBuf:    newTestReader(h0, h1),
		readBytes:  0,
		readLimit:  readLimit,
		writeBuf:   bytes.NewBuffer(nil),
		writeBytes: 0,
		writeLimit: writeLimit,
	}
}

func (rw *testReadWriter) Read(p []byte) (int, error) {
	// Check read limit
	if rw.readLimit >= 0 && rw.readBytes >= rw.readLimit {
		return 0, io.EOF
	}

	n, err := rw.readBuf.Read(p)
	rw.readBytes += n
	return n, err
}

func (rw *testReadWriter) Write(p []byte) (int, error) {
	// Check write limit
	if rw.writeLimit >= 0 && rw.writeBytes >= rw.writeLimit {
		return 0, io.ErrShortWrite
	}

	n := len(p)
	if rw.writeLimit >= 0 && rw.writeBytes+n > rw.writeLimit {
		n = rw.writeLimit - rw.writeBytes
	}

	written, _ := rw.writeBuf.Write(p[:n])
	rw.writeBytes += written

	// Provide echo when C0+C1 are fully written
	if rw.writeBuf.Len() == 1+HandshakeSize {
		c1 := rw.writeBuf.Bytes()[1 : 1+HandshakeSize]
		rw.readBuf.Write(c1)
	}

	if rw.writeLimit >= 0 && written < len(p) {
		return written, io.ErrShortWrite
	}

	return written, nil
}

func makeTestHandshakeData() []byte {
	data := make([]byte, HandshakeSize)
	_, _ = rand.Read(data)
	return data
}

func newTestReader(h0, h1 []byte) *bytes.Buffer {
	buf := bytes.NewBuffer(h0)
	if len(h1) > 0 {
		buf.Write(h1)
	}
	return buf
}
