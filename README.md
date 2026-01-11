# E-RTMP

A Go library implementation of the Real-Time Messaging Protocol (RTMP) with plans for Enhanced RTMP support.

## Status

**Current**: RTMP basic implementation completed
- ✅ RTMP handshake
- ✅ Chunk-based I/O (Reader/Writer)
- ✅ Transport layer with protocol control messages
- ✅ Automatic acknowledgement and window size handling
- ✅ Abort message support for canceling partial messages
- ✅ Reference-counted buffer management with pooling
- ✅ Connection management
- ✅ AMF0 encoding/decoding
- ✅ Command messages (connect, publish, play)
- ✅ Video/Audio/Metadata streaming

**Future**: Enhanced RTMP (E-RTMP) features planned
- Multiple video/audio codecs (HEVC, AV1, VP9, Opus)
- FourCC-based codec negotiation

## Requirements

- Go 1.25+

## Installation

```bash
go get github.com/ssungk/ertmp
```

## Project Structure

```
├── pkg/                    # Public packages - main library code
│   ├── amf/               # AMF0/AMF3 encoder/decoder
│   ├── common/            # Common types and constants
│   └── rtmp/              # RTMP core implementation
│       ├── buf/           # Buffer management with pooling
│       │   ├── buffer.go          # Reference-counted buffer
│       │   └── pool.go            # Memory pool for buffers
│       ├── command.go     # AMF command encoding/decoding
│       ├── config.go      # RTMP configuration
│       ├── conn.go        # RTMP connection management
│       ├── helper.go      # Helper functions
│       └── transport/     # RTMP transport layer (I/O)
│           ├── handshake.go        # RTMP handshake
│           ├── reader.go           # Chunk reader with message assembly
│           ├── writer.go           # Chunk writer
│           ├── transport.go        # Transport (Reader + Writer)
│           ├── message.go          # Message types
│           └── message_assembler.go # Assembles chunks into messages
├── cmd/                   # Command-line applications
│   ├── server/           # Example RTMP server
│   └── client/           # Example RTMP client
└── CLAUDE.md             # Development guidelines for AI assistants
```

## Development

This is a **library project**, not an application. Use standard Go commands:

### Build all packages
```bash
go build ./...
```

### Run tests
```bash
go test ./...
```

### Run specific test
```bash
go test ./pkg/rtmp/transport -v
```

### Format code
```bash
go fmt ./...
```

### Tidy dependencies
```bash
go mod tidy
```

## Architecture

### Layer Separation

The RTMP implementation is split into two clear layers:

1. **Transport Layer** (`pkg/rtmp/transport/`)
   - Low-level I/O operations
   - Chunk-based reading/writing
   - Protocol control messages
   - Message framing
   - Independent, reusable components

2. **RTMP Layer** (`pkg/rtmp/`)
   - High-level RTMP API
   - Connection management
   - Command encoding/decoding
   - Stream management
   - Business logic

### Design Principles

- **Zero-copy oriented**: Buffer pooling with reference counting, minimize allocations
- **Clear layer separation**: Transport and RTMP layers are independent
- **No circular dependencies**: Clean module structure
- **Simple and focused**: Each component has a single responsibility
- **Thread-safe buffer management**: Automatic memory pool management with reference counting

### Key Implementation Details

#### Buffer Management (`pkg/rtmp/buf/`)
- **Reference-counted buffers**: Automatic memory management using atomic reference counting
- **Tiered memory pools**: Multiple pool sizes (32B, 512B, 4KB, 16KB, 64KB) for efficient allocation
- **Zero-copy sharing**: Messages can share buffers across streams without copying

#### Message Assembly (`pkg/rtmp/transport/`)
- **MessageAssembler**: Reconstructs complete messages from interleaved chunks
- **Per-stream state**: Maintains separate assembly state for each chunk stream ID
- **Direct buffer writes**: Chunks are read directly into pre-allocated message buffers

## Testing with FFmpeg

### Publish stream
```bash
ffmpeg -re -i video.mp4 -c:v libx264 -c:a aac \
  -f flv rtmp://localhost:1935/live/stream
```

### Play stream
```bash
ffplay rtmp://localhost:1935/live/stream
```

### Get stream info
```bash
ffprobe rtmp://localhost:1935/live/stream
```

## License

MIT License - see LICENSE file for details

## Author

sjyoon (yoontjdwo@gmail.com)
