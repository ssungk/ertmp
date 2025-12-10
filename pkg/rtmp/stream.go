package rtmp

// StreamMode represents the stream mode
type StreamMode int

const (
	StreamModeNone StreamMode = iota
	StreamModePublish
	StreamModePlay
)

// Stream represents an RTMP stream
type Stream struct {
	id       uint32
	key      string
	mode     StreamMode
	metadata map[string]interface{}
}

// NewStream creates a new stream
func NewStream(id uint32) *Stream {
	return &Stream{
		id:       id,
		metadata: make(map[string]interface{}),
	}
}

// ID returns the stream ID
func (s *Stream) ID() uint32 {
	return s.id
}

// Key returns the stream key
func (s *Stream) Key() string {
	return s.key
}

// SetKey sets the stream key
func (s *Stream) SetKey(key string) {
	s.key = key
}

// Mode returns the stream mode
func (s *Stream) Mode() StreamMode {
	return s.mode
}

// SetMode sets the stream mode
func (s *Stream) SetMode(mode StreamMode) {
	s.mode = mode
}

// Metadata returns the stream metadata
func (s *Stream) Metadata() map[string]interface{} {
	return s.metadata
}

// SetMetadata sets the stream metadata
func (s *Stream) SetMetadata(metadata map[string]interface{}) {
	s.metadata = metadata
}
