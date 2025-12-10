package rtmp

import "github.com/ssungk/ertmp/pkg/rtmp/transport"

// Config holds RTMP protocol configuration
type Config struct {
	WindowAckSize uint32
	PeerBandwidth uint32
	ChunkSize     uint32
}

// DefaultConfig returns default RTMP configuration
func DefaultConfig() Config {
	return Config{
		WindowAckSize: transport.DefaultWindowAckSize,
		PeerBandwidth: transport.DefaultPeerBandwidth,
		ChunkSize:     transport.DefaultChunkSize,
	}
}