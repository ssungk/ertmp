package transport

import "errors"

var (
	// I/O errors
	ErrRtmpRead  = errors.New("rtmp read failed")
	ErrRtmpWrite = errors.New("rtmp write failed")

	// Protocol errors
	ErrUnsupportedVersion = errors.New("unsupported RTMP version")

	// Message header errors
	ErrNoPreviousHeader = errors.New("format type requires previous header")
)
