# E-RTMP

A Go library implementation of the Real-Time Messaging Protocol (RTMP) with plans for Enhanced RTMP support.

## Status

**Current**: RTMP basic implementation completed
- ✅ RTMP handshake
- ✅ Chunk-based I/O (Reader/Writer)
- ✅ Transport layer
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
│       ├── command.go     # AMF command encoding/decoding
│       ├── config.go      # RTMP configuration
│       ├── conn.go        # RTMP connection management
│       ├── helper.go      # Helper functions
│       └── transport/     # RTMP transport layer (I/O)
│           ├── handshake.go        # RTMP handshake
│           ├── reader.go           # Chunk reader
│           ├── writer.go           # Chunk writer
│           ├── transport.go        # Transport (Reader + Writer)
│           └── message.go          # Message types
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

- **Zero-copy oriented**: Buffer pooling, minimize allocations
- **Clear layer separation**: Transport and RTMP layers are independent
- **No circular dependencies**: Clean module structure
- **Simple and focused**: Each component has a single responsibility

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
