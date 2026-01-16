package main

import (
	"log/slog"
	"net"

	"github.com/ssungk/ertmp/pkg/rtmp"
	"github.com/ssungk/ertmp/pkg/rtmp/transport"
)

// Session represents a client session
type Session struct {
	server    *Server
	netConn   net.Conn
	conn      *rtmp.Conn
	streamID  uint32
	streamKey string
	mode      string // "publish" or "play"
}

// NewSession creates a new client session
func NewSession(netConn net.Conn, server *Server) *Session {
	return &Session{
		server:  server,
		netConn: netConn,
	}
}

// Run handles the session (handshake + message loop)
func (s *Session) Run() {
	defer s.Close()

	// RTMP 연결 생성 (핸드셰이크 포함)
	conn, err := rtmp.AcceptConn(s.netConn)
	if err != nil {
		slog.Error("Handshake failed", "error", err, "address", s.netConn.RemoteAddr())
		return
	}
	s.conn = conn
	defer s.conn.Close()

	slog.Info("Client connected", "address", s.netConn.RemoteAddr())

	// 메시지 루프
	for {
		msg, err := s.conn.ReadMessage()
		if err != nil {
			slog.Error("Read error", "error", err)
			break
		}

		if err := s.handleMessage(msg); err != nil {
			slog.Error("Failed to handle message", "error", err)
			msg.Release()
			break
		}
		msg.Release()
	}

	slog.Info("Client disconnected", "address", s.netConn.RemoteAddr())
}

// handleMessage handles a single message
func (s *Session) handleMessage(msg *transport.Message) error {
	switch msg.Type() {
	case transport.MsgTypeAMF0Command:
		return s.handleCommand(msg)

	case transport.MsgTypeVideo:
		s.handleVideo(msg)

	case transport.MsgTypeAudio:
		s.handleAudio(msg)

	case transport.MsgTypeAMF0Data:
		s.handleMetadata(msg)

	default:
		slog.Debug("Unknown message type", "type", msg.Type())
	}

	return nil
}

// handleCommand handles AMF command messages
func (s *Session) handleCommand(msg *transport.Message) error {
	cmd, err := rtmp.DecodeCommand(msg.Data())
	if err != nil {
		slog.Warn("Failed to decode command", "error", err)
		return nil
	}

	switch cmd.Name {
	case "connect":
		return s.handleConnect(msg, cmd)

	case "createStream":
		return s.handleCreateStream(msg, cmd)

	case "publish":
		return s.handlePublish(msg, cmd)

	case "play":
		return s.handlePlay(msg, cmd)

	case "deleteStream":
		slog.Info("Stream deleted")
		err := s.Close()
		if err != nil {
			slog.Error("Failed to close session", "error", err)
		}
		return err

	default:
		slog.Debug("Unknown command", "name", cmd.Name)
	}

	return nil
}

// handleConnect handles connect command
func (s *Session) handleConnect(msg *transport.Message, cmd *rtmp.Command) error {
	slog.Info("Connect request", "txID", cmd.TransactionID)

	if err := rtmp.HandleConnect(s.conn, msg); err != nil {
		slog.Error("HandleConnect failed", "error", err)
		return err
	}

	slog.Info("Connect response sent")
	return nil
}

// handleCreateStream handles createStream command
func (s *Session) handleCreateStream(msg *transport.Message, cmd *rtmp.Command) error {
	slog.Info("CreateStream request", "txID", cmd.TransactionID)

	stream, err := rtmp.HandleCreateStream(s.conn, msg)
	if err != nil {
		return err
	}

	slog.Info("Stream created", "streamID", stream.ID())

	return nil
}

// handlePublish handles publish command
func (s *Session) handlePublish(msg *transport.Message, cmd *rtmp.Command) error {
	publishCmd, err := rtmp.ParsePublish(cmd)
	if err != nil {
		return err
	}

	slog.Info("Publish request",
		"streamKey", publishCmd.StreamKey,
		"type", publishCmd.PublishType)

	if err := rtmp.HandlePublish(s.conn, msg); err != nil {
		return err
	}

	// 세션에 스트림 ID, 키, 모드 저장
	s.streamID = msg.StreamID()
	s.streamKey = publishCmd.StreamKey
	s.mode = "publish"

	// 서버 스트림에 publisher 등록
	stream := s.server.GetOrCreateStream(publishCmd.StreamKey)
	stream.SetPublisher(s)

	slog.Info("Publish started",
		"streamID", s.streamID,
		"streamKey", publishCmd.StreamKey,
		"type", publishCmd.PublishType)

	return nil
}

// handlePlay handles play command
func (s *Session) handlePlay(msg *transport.Message, cmd *rtmp.Command) error {
	playCmd, err := rtmp.ParsePlay(cmd)
	if err != nil {
		return err
	}

	slog.Info("Play request", "streamKey", playCmd.StreamKey)

	if err := rtmp.HandlePlay(s.conn, msg); err != nil {
		return err
	}

	// 세션에 스트림 ID, 키, 모드 저장
	s.streamID = msg.StreamID()
	s.streamKey = playCmd.StreamKey
	s.mode = "play"

	// 서버 스트림에 subscriber 등록
	stream := s.server.GetOrCreateStream(playCmd.StreamKey)
	stream.AddSubscriber(s)

	// publisher가 있으면 초기화 데이터 전송
	// 1. Metadata
	if metadata := stream.GetMetadata(); metadata != nil {
		header := transport.NewMessageHeader(s.streamID, 0, transport.MsgTypeAMF0Data)
		rtmpMsg := transport.NewMessage(header, metadata)
		if err := s.conn.WriteMessage(rtmpMsg); err != nil {
			slog.Error("Failed to send metadata", "error", err)
		}
		rtmpMsg.Release()
	}

	// 2. Video sequence header
	if videoSeqHeader := stream.GetVideoSeqHeader(); videoSeqHeader != nil {
		header := transport.NewMessageHeader(s.streamID, 0, transport.MsgTypeVideo)
		rtmpMsg := transport.NewMessage(header, videoSeqHeader)
		if err := s.conn.WriteMessage(rtmpMsg); err != nil {
			slog.Error("Failed to send video sequence header", "error", err)
		}
		rtmpMsg.Release()
		slog.Info("Video sequence header sent", "streamKey", playCmd.StreamKey)
	}

	// 3. Audio sequence header
	if audioSeqHeader := stream.GetAudioSeqHeader(); audioSeqHeader != nil {
		header := transport.NewMessageHeader(s.streamID, 0, transport.MsgTypeAudio)
		rtmpMsg := transport.NewMessage(header, audioSeqHeader)
		if err := s.conn.WriteMessage(rtmpMsg); err != nil {
			slog.Error("Failed to send audio sequence header", "error", err)
		}
		rtmpMsg.Release()
		slog.Info("Audio sequence header sent", "streamKey", playCmd.StreamKey)
	}

	slog.Info("Play started",
		"streamID", s.streamID,
		"streamKey", playCmd.StreamKey)

	return nil
}

// handleVideo handles video data
func (s *Session) handleVideo(msg *transport.Message) {
	// Sequence header 감지 (FrameType=1, CodecID=7, AVCPacketType=0)
	data := msg.Data()
	if len(data) >= 2 {
		frameType := (data[0] >> 4) & 0x0F
		codecID := data[0] & 0x0F
		avcPacketType := data[1]

		// AVC sequence header (H.264)
		if frameType == 1 && codecID == 7 && avcPacketType == 0 {
			stream := s.server.GetOrCreateStream(s.streamKey)
			stream.SetVideoSeqHeader(data)
			slog.Info("Video sequence header cached", "streamKey", s.streamKey, "bytes", len(data))
		}
	}

	s.broadcastToSubscribers(msg, "video")
}

// handleAudio handles audio data
func (s *Session) handleAudio(msg *transport.Message) {
	// Sequence header 감지 (SoundFormat=10, AACPacketType=0)
	data := msg.Data()
	if len(data) >= 2 {
		soundFormat := (data[0] >> 4) & 0x0F
		aacPacketType := data[1]

		// AAC sequence header
		if soundFormat == 10 && aacPacketType == 0 {
			stream := s.server.GetOrCreateStream(s.streamKey)
			stream.SetAudioSeqHeader(data)
			slog.Info("Audio sequence header cached", "streamKey", s.streamKey, "bytes", len(data))
		}
	}

	s.broadcastToSubscribers(msg, "audio")
}

// broadcastToSubscribers broadcasts media data to all subscribers
func (s *Session) broadcastToSubscribers(msg *transport.Message, mediaType string) {
	// publish 모드가 아니면 무시
	if s.mode != "publish" || s.streamKey == "" {
		return
	}

	slog.Debug("Media data",
		"type", mediaType,
		"bytes", len(msg.Data()),
		"timestamp", msg.Timestamp(),
		"streamKey", s.streamKey)

	// 모든 subscribers에게 전송
	stream := s.server.GetOrCreateStream(s.streamKey)
	subscribers := stream.GetSubscribers()

	for _, sub := range subscribers {
		// 버퍼를 공유하는 새 메시지 생성 (zero-copy)
		sharedMsg := msg.Share(sub.streamID)
		if err := sub.conn.WriteMessage(sharedMsg); err != nil {
			slog.Error("Failed to send to subscriber", "type", mediaType, "error", err)
		}
		sharedMsg.Release()
	}
}

// handleMetadata handles metadata
func (s *Session) handleMetadata(msg *transport.Message) {
	// publish 모드가 아니면 무시
	if s.mode != "publish" || s.streamKey == "" {
		return
	}

	slog.Info("Metadata received",
		"bytes", len(msg.Data()),
		"streamKey", s.streamKey)

	// 스트림에 metadata 저장
	stream := s.server.GetOrCreateStream(s.streamKey)
	stream.SetMetadata(msg.Data())

	// 모든 subscribers에게 전송
	subscribers := stream.GetSubscribers()

	for _, sub := range subscribers {
		// 버퍼를 공유하는 새 메시지 생성 (zero-copy)
		sharedMsg := msg.Share(sub.streamID)
		if err := sub.conn.WriteMessage(sharedMsg); err != nil {
			slog.Error("Failed to send metadata to subscriber", "error", err)
		}
		sharedMsg.Release()
	}
}

// Close closes the session
func (s *Session) Close() error {
	// 스트림에서 제거
	if s.streamKey != "" {
		stream := s.server.GetOrCreateStream(s.streamKey)
		if s.mode == "publish" {
			stream.RemovePublisher()
			slog.Info("Publisher disconnected", "streamKey", s.streamKey)
		} else if s.mode == "play" {
			stream.RemoveSubscriber(s)
			slog.Info("Subscriber disconnected", "streamKey", s.streamKey)
		}
		// 스트림이 비어있으면 제거
		s.server.RemoveStream(s.streamKey)
	}

	if s.netConn != nil {
		return s.netConn.Close()
	}
	return nil
}
