package main

import (
	"log/slog"
	"net"
	"os"
	"sync"
)

// Server represents RTMP server
type Server struct {
	addr    string
	streams map[string]*Stream
	mu      sync.RWMutex
}

// Stream represents a publish/play stream
type Stream struct {
	key            string
	publisher      *Session
	subscribers    map[*Session]bool
	metadata       []byte
	videoSeqHeader []byte
	audioSeqHeader []byte
	mu             sync.RWMutex
}

// NewServer creates a new RTMP server
func NewServer() *Server {
	return &Server{
		addr:    ":1935",
		streams: make(map[string]*Stream),
	}
}

// Run starts the RTMP server and blocks forever
func (s *Server) Run() {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		slog.Error("Failed to start server", "error", err, "addr", s.addr)
		os.Exit(1)
	}

	slog.Info("RTMP server started", "addr", s.addr)

	for {
		netConn, err := listener.Accept()
		if err != nil {
			slog.Error("Accept failed", "error", err)
			continue
		}

		session := NewSession(netConn, s)
		go session.Run()
	}
}

// GetOrCreateStream gets or creates a stream
func (s *Server) GetOrCreateStream(key string) *Stream {
	s.mu.Lock()
	defer s.mu.Unlock()

	stream, ok := s.streams[key]
	if !ok {
		stream = &Stream{
			key:         key,
			subscribers: make(map[*Session]bool),
		}
		s.streams[key] = stream
	}
	return stream
}

// RemoveStream removes a stream if it has no publisher and subscribers
func (s *Server) RemoveStream(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stream, ok := s.streams[key]
	if !ok {
		return
	}

	stream.mu.RLock()
	hasPublisher := stream.publisher != nil
	hasSubscribers := len(stream.subscribers) > 0
	stream.mu.RUnlock()

	if !hasPublisher && !hasSubscribers {
		delete(s.streams, key)
		slog.Info("Stream removed", "streamKey", key)
	}
}

// SetPublisher sets the publisher for a stream
func (st *Stream) SetPublisher(session *Session) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.publisher = session
}

// RemovePublisher removes the publisher from a stream
func (st *Stream) RemovePublisher() {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.publisher = nil
}

// GetPublisher gets the publisher of a stream
func (st *Stream) GetPublisher() *Session {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.publisher
}

// AddSubscriber adds a subscriber to the stream
func (st *Stream) AddSubscriber(session *Session) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.subscribers[session] = true
	slog.Info("Subscriber added", "streamKey", st.key, "total", len(st.subscribers))
}

// RemoveSubscriber removes a subscriber from the stream
func (st *Stream) RemoveSubscriber(session *Session) {
	st.mu.Lock()
	defer st.mu.Unlock()
	delete(st.subscribers, session)
	slog.Info("Subscriber removed", "streamKey", st.key, "total", len(st.subscribers))
}

// GetSubscribers returns a copy of subscribers
func (st *Stream) GetSubscribers() []*Session {
	st.mu.RLock()
	defer st.mu.RUnlock()

	subscribers := make([]*Session, 0, len(st.subscribers))
	for sub := range st.subscribers {
		subscribers = append(subscribers, sub)
	}
	return subscribers
}

// SetMetadata sets the metadata for the stream
func (st *Stream) SetMetadata(data []byte) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.metadata = make([]byte, len(data))
	copy(st.metadata, data)
}

// GetMetadata returns a copy of the metadata
func (st *Stream) GetMetadata() []byte {
	st.mu.RLock()
	defer st.mu.RUnlock()
	if st.metadata == nil {
		return nil
	}
	data := make([]byte, len(st.metadata))
	copy(data, st.metadata)
	return data
}

// SetVideoSeqHeader sets the video sequence header
func (st *Stream) SetVideoSeqHeader(data []byte) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.videoSeqHeader = make([]byte, len(data))
	copy(st.videoSeqHeader, data)
}

// GetVideoSeqHeader returns a copy of the video sequence header
func (st *Stream) GetVideoSeqHeader() []byte {
	st.mu.RLock()
	defer st.mu.RUnlock()
	if st.videoSeqHeader == nil {
		return nil
	}
	data := make([]byte, len(st.videoSeqHeader))
	copy(data, st.videoSeqHeader)
	return data
}

// SetAudioSeqHeader sets the audio sequence header
func (st *Stream) SetAudioSeqHeader(data []byte) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.audioSeqHeader = make([]byte, len(data))
	copy(st.audioSeqHeader, data)
}

// GetAudioSeqHeader returns a copy of the audio sequence header
func (st *Stream) GetAudioSeqHeader() []byte {
	st.mu.RLock()
	defer st.mu.RUnlock()
	if st.audioSeqHeader == nil {
		return nil
	}
	data := make([]byte, len(st.audioSeqHeader))
	copy(data, st.audioSeqHeader)
	return data
}
