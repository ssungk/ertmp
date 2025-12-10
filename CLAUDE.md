# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the E-RTMP (Enhanced Real-Time Messaging Protocol) library written in Go.

**IMPORTANT**: This is a **LIBRARY project**, NOT an application with a main executable.

### Current Status
- ✅ RTMP basic protocol implementation completed
- ✅ FFmpeg-tested and production-ready
- ⏳ Enhanced RTMP (E-RTMP) features planned for future

## Project Structure

```
├── pkg/                    # Public packages - main library code
│   ├── amf/               # AMF0/AMF3 encoder/decoder (완성)
│   ├── common/            # Common types and constants
│   └── rtmp/              # RTMP core implementation (완성)
│       ├── command.go     # AMF command encoding/decoding
│       ├── config.go      # RTMP configuration
│       ├── conn.go        # RTMP connection management
│       ├── helper.go      # Helper functions
│       └── transport/     # RTMP transport layer (I/O)
│           ├── handshake.go        # RTMP handshake
│           ├── reader.go           # Chunk reader
│           ├── writer.go           # Chunk writer
│           ├── transport.go        # Transport (Reader + Writer)
│           ├── message.go          # Message types
│           ├── buffer.go           # Buffer pool
│           └── constants.go        # Protocol constants
├── cmd/                   # Command-line applications (examples)
│   ├── server/           # Example RTMP server
│   └── client/           # Example RTMP client
└── test/                 # Test files
```

## Development Commands

```bash
# Build all packages
go build ./...

# Run all tests
go test ./...

# Run specific package tests
go test ./pkg/rtmp/transport -v

# Format code
go fmt ./...

# Tidy dependencies
go mod tidy
```

## Architecture & Design Principles

### Layer Separation

The implementation is split into two clear layers:

1. **Transport Layer** (`pkg/rtmp/transport/`)
   - Low-level I/O operations
   - Chunk-based reading/writing
   - Protocol control messages
   - Message framing
   - Completely independent, no dependencies on upper layers

2. **RTMP Layer** (`pkg/rtmp/`)
   - High-level RTMP API
   - Connection management
   - Command encoding/decoding
   - Stream management
   - Depends on transport layer

### Key Design Principles

1. **Zero-copy oriented**: Use buffer pooling, minimize allocations
2. **No circular dependencies**: Transport layer is independent
3. **Simple and focused**: Each component has a single responsibility
4. **Thread-safe**: Reference counting for message buffers
5. **Test-driven**: All critical paths have unit tests

### Important Implementation Details

**Message Reference Counting:**
- Messages use `atomic.Int32` for thread-safe reference counting
- Use `Share()` to share message buffers between subscribers (zero-copy)
- Always call `Release()` after using a message
- Last `Release()` returns buffer to pool

**Transport Protocol Control:**
- Transport automatically handles `SetChunkSize`, `WindowAckSize`, `SetPeerBandwidth`
- Automatic acknowledgement when reaching window size
- Automatic flush after write

**Reader Logic:**
- `PrevHeader` updates only when message is complete (not per chunk)
- `cs.BytesRead == 0` indicates new message start
- Buffer ownership transfers via `MoveBuffers()`

## Important Guidelines

### What NOT to do

❌ Do not create main executables in pkg/ directory
❌ Do not use Makefile - use standard Go commands
❌ Do not create unnecessary abstraction layers
❌ Do not modify completed packages (pkg/amf/, pkg/rtmp/) without good reason
❌ Do not add markdown documentation files unless explicitly requested

### What TO do

✅ Keep code simple and focused
✅ Add tests for new features
✅ Follow existing naming conventions
✅ Use buffer pool for large allocations
✅ Call `Release()` on messages after use
✅ Put example code in cmd/ directory
✅ Test with FFmpeg when changing protocol logic

## Testing Guidelines

### Unit Testing

All critical components have unit tests:
- `handshake_test.go` - Handshake with all error cases
- `reader_test.go` - Chunk reading logic
- `writer_test.go` - Chunk writing logic
- `command_test.go` - AMF command encoding/decoding

### Integration Testing with FFmpeg

```bash
# Start example server
go run cmd/server/main.go

# Publish stream
ffmpeg -re -i video.mp4 -c:v libx264 -c:a aac \
  -f flv rtmp://localhost:1935/live/stream

# Play stream
ffplay rtmp://localhost:1935/live/stream

# Get stream info
ffprobe rtmp://localhost:1935/live/stream
```

## Coding Conventions

### Go Standards
- Go version: 1.25+
- Follow standard Go conventions
- Use `gofmt` for formatting
- Use meaningful variable names

### Naming Patterns
- Handshake test variables: `h0` (version), `h1` (handshake data), `i0` (invalid)
- Test limits: `noLimit`, `failImmediately`, `failAfterVersion`, `failAfterC0C1`
- Read/Write ordering: read-related parameters before write-related

### Error Handling
- Use `errors.Is()` for error checking
- Wrap errors with context using `fmt.Errorf()`
- Return sentinel errors from transport layer: `ErrRead`, `ErrWrite`, `ErrUnsupportedVersion`

## Future Work: Enhanced RTMP (E-RTMP)

### Planned Features
- Multiple video codecs: HEVC (hvc1), AV1 (av01), VP9 (vp09)
- Multiple audio codecs: Opus, AC-3, EAC-3
- FourCC-based codec negotiation
- Enhanced metadata

### Implementation Approach

**Package Structure:**
```
pkg/ertmp/
├── conn.go           # Enhanced RTMP connection (embeds rtmp.Conn)
├── message.go        # Enhanced Video/Audio message types
├── codec.go          # FourCC codec definitions
└── parser.go         # Enhanced message parser
```

**Design Strategy:**
1. Embed `rtmp.Conn` in `ertmp.Conn` (composition, not modification)
2. `rtmp.ConnectCommand` already supports `FourCcList` and `CapsEx` fields
3. Auto-detect Enhanced vs Legacy mode during connection negotiation
4. Parse Enhanced headers only for Enhanced mode messages
5. Keep `pkg/rtmp/` unchanged (backward compatible)

**Enhanced Message Detection:**
- Check first byte `& 0x80` for IsExHeader flag
- Parse PacketType and FourCC for Enhanced messages
- Fall back to legacy parsing for non-Enhanced messages

### Testing Requirements for E-RTMP
- Enhanced/Legacy message parsing
- FourCC negotiation
- Capability detection
- Legacy fallback behavior
- Multiple codec support

---

**Note**: Do not implement E-RTMP features yet. The basic RTMP implementation is complete and stable. E-RTMP will be implemented as a separate enhancement layer when needed.
