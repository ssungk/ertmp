# E-RTMP

A Go library implementation of the Real-Time Messaging Protocol (RTMP) with plans for Enhanced RTMP support.

## Status

**Current**: RTMP basic implementation completed
- ✅ RTMP handshake (C0/C1/C2/S0/S1/S2)
- ✅ Chunk-based I/O (Reader/Writer)
- ✅ Transport layer with automatic protocol control message handling
- ✅ Automatic acknowledgement and window size management
- ✅ Abort message support for canceling partial chunks
- ✅ Ping/Pong auto-response for connection keep-alive
- ✅ Extended Timestamp support (streams > 4.6 hours)
- ✅ Reference-counted buffer management with pooling
- ✅ Connection and stream management
- ✅ AMF0/AMF3 encoding/decoding
- ✅ Command messages (connect, createStream, publish, play)
- ✅ Video/Audio/Metadata streaming

**Future**: Enhanced RTMP (E-RTMP) features planned
- Multiple video/audio codecs (HEVC, AV1, VP9, Opus)
- FourCC-based codec negotiation

## Features

### Core RTMP Protocol
- **Complete RTMP handshake** (C0/C1/C2 and S0/S1/S2)
- **Chunk-based streaming** with configurable chunk sizes
- **Type 0/1/2/3 chunk headers** with optimal header compression
- **Extended Timestamp support** for long-running streams (>4.6 hours)
- **Message interleaving** for concurrent streams

### Automatic Protocol Handling
- **Auto-acknowledgement**: Sends ACK messages based on window size
- **Auto-response to ping**: Keeps connections alive automatically
- **Protocol control encapsulation**: Safe APIs prevent state corruption
- **Abort message handling**: Properly clears partial chunks

### Performance Optimizations
- **Zero-copy buffer sharing**: Messages share buffers across streams
- **Tiered memory pools**: Efficient allocation with 5 pool sizes
- **Reference counting**: Automatic buffer lifecycle management
- **Minimal allocations**: Reuses buffers and headers where possible

### Safety & Correctness
- **Value-type messages**: Prevents accidental null pointer dereferences
- **Explicit buffer ownership**: Clear lifecycle with Retain/Release
- **No circular dependencies**: Clean architecture with layer separation
- **Pit of success design**: Hard to misuse APIs

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
   - Chunk-based reading/writing with message assembly
   - **Automatic protocol control message handling**
     - Incoming: SetChunkSize, WindowAckSize, Abort, UserControl
     - Outgoing: Acknowledgement auto-send, PingResponse auto-reply
   - Message framing and buffering
   - Independent, reusable components

2. **RTMP Layer** (`pkg/rtmp/`)
   - High-level RTMP API
   - Connection and stream management
   - Safe protocol control methods (SetChunkSize, SetWindowAckSize, SetPeerBandwidth)
   - Command encoding/decoding (connect, publish, play)
   - Application-level business logic

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
- **Explicit lifecycle management**: Use `buffer.Retain()` to share, `buffer.Release()` to free
- **Zero-copy sharing**: Multiple messages can share the same buffer without copying data

```go
// Example: Sharing a buffer across streams
buffer := msg.Buffer()
buffer.Retain()  // Increment reference count
header := transport.NewMessageHeader(newStreamID, timestamp, msgType)
sharedMsg := transport.NewMessage(header, buffer)
// Both msg and sharedMsg now share the same buffer
```

#### Message Type (`pkg/rtmp/transport/`)
- **Value type**: `Message` is a struct (not pointer) for better performance
- **Buffer ownership**: `NewMessage(header, buffer)` takes ownership of buffer
- **Explicit buffer access**: Use `msg.Buffer()` to access underlying buffer for lifecycle management
- **Zero-copy transfer**: Pass buffers directly without intermediate copying

```go
// Creating a message
data := []byte("payload")
buffer := buf.New(data)  // Wrap data in pooled buffer
header := transport.NewMessageHeader(streamID, timestamp, msgType)
msg := transport.NewMessage(header, buffer)  // Takes ownership

// Release when done
defer msg.Buffer().Release()
```

#### Message Assembly (`pkg/rtmp/transport/`)
- **MessageAssembler**: Reconstructs complete messages from interleaved chunks
- **Per-stream state**: Maintains separate assembly state for each chunk stream ID
- **Direct buffer writes**: Chunks are read directly into pre-allocated message buffers
- **Header caching**: Reuses message headers across Type 3 chunks for efficiency

## Usage Example

### Basic Server

```go
package main

import (
    "log"
    "net"

    "github.com/ssungk/ertmp/pkg/rtmp"
    "github.com/ssungk/ertmp/pkg/rtmp/transport"
)

func main() {
    listener, err := net.Listen("tcp", ":1935")
    if err != nil {
        log.Fatal(err)
    }
    defer listener.Close()

    for {
        netConn, err := listener.Accept()
        if err != nil {
            continue
        }

        go handleConnection(netConn)
    }
}

func handleConnection(netConn net.Conn) {
    defer netConn.Close()

    // RTMP handshake
    if err := transport.Handshake(netConn, true); err != nil {
        log.Printf("Handshake failed: %v", err)
        return
    }

    // Create RTMP connection
    conn := rtmp.NewConn(netConn, rtmp.DefaultConfig())
    defer conn.Close()

    // Read and handle messages
    for {
        msg, err := conn.ReadMessage()
        if err != nil {
            break
        }

        // Handle message based on type
        switch msg.Type() {
        case transport.MsgTypeAMF0Command:
            handleCommand(conn, msg)
        case transport.MsgTypeVideo:
            handleVideo(msg)
        case transport.MsgTypeAudio:
            handleAudio(msg)
        }

        msg.Buffer().Release()  // Always release buffers
    }
}
```

### Protocol Control

```go
// Set chunk size (must use dedicated method, not WriteMessage)
conn.SetChunkSize(4096)

// Set window acknowledgement size
conn.SetWindowAckSize(2500000)

// Set peer bandwidth
conn.SetPeerBandwidth(2500000, transport.LimitTypeDynamic)
```

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

## Contributing

Contributions are welcome! Please follow these guidelines:

### Code Quality
- Write tests for new features
- Maintain test coverage above 80%
- Follow Go best practices and idioms
- Use `go fmt` and `go vet` before committing

### Buffer Management
- Always use buffer pools (`buf.New()`, `buf.NewFromPool()`)
- Call `buffer.Release()` when done with buffers
- Use `buffer.Retain()` when sharing buffers
- Test buffer lifecycle carefully to avoid leaks

### Architecture
- Respect layer separation (Transport vs RTMP)
- Keep Transport layer independent and reusable
- Use safe protocol control APIs, don't bypass them
- Avoid circular dependencies

## Roadmap

### Short-term
- [ ] Increase test coverage to 80%+
- [ ] Add CI/CD with GitHub Actions
- [ ] Improve AMF encoder buffer pool usage
- [ ] Add more integration tests

### Long-term
- [ ] Client API implementation
- [ ] Enhanced RTMP (E-RTMP) support
  - HEVC/H.265 codec
  - AV1 codec
  - VP9 codec
  - Opus audio codec
- [ ] Performance benchmarks and optimizations

## License

MIT License - see LICENSE file for details

## Author

sjyoon (yoontjdwo@gmail.com)
