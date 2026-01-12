package rtmp

import (
	"fmt"

	"github.com/ssungk/ertmp/pkg/rtmp/transport"
)

// HandleConnect handles a connect command (server side)
func HandleConnect(conn *Conn, msg *transport.Message) error {
	cmd, err := DecodeCommand(msg.Data())
	if err != nil {
		return fmt.Errorf("failed to decode connect command: %w", err)
	}

	connectCmd, err := ParseConnect(cmd)
	if err != nil {
		return fmt.Errorf("failed to parse connect: %w", err)
	}

	// 프로토콜 제어 메시지 전송
	if err := SendWindowAckSize(conn, conn.config.WindowAckSize); err != nil {
		return err
	}
	if err := SendSetPeerBW(conn, conn.config.PeerBandwidth, 2); err != nil {
		return err
	}
	if err := SendSetChunkSize(conn, conn.config.ChunkSize); err != nil {
		return err
	}

	// 응답 속성 구성
	props := map[string]interface{}{
		"fmsVer":       "FMS/3,0,1,123",
		"capabilities": 31.0,
	}

	// Enhanced RTMP 지원
	if len(connectCmd.FourCcList) > 0 {
		props["fourCcList"] = connectCmd.FourCcList
	}
	if connectCmd.CapsEx != nil {
		props["capsEx"] = connectCmd.CapsEx
	}

	// connect 응답 전송
	return SendConnectResponse(conn, cmd.TransactionID, props)
}

// HandleCreateStream handles a createStream command (server side)
func HandleCreateStream(conn *Conn, msg *transport.Message) (*Stream, error) {
	cmd, err := DecodeCommand(msg.Data())
	if err != nil {
		return nil, fmt.Errorf("failed to decode createStream command: %w", err)
	}

	// 새 스트림 생성
	stream := conn.createStream()

	// 응답 전송
	if err := SendCreateStreamResponse(conn, cmd.TransactionID, float64(stream.ID())); err != nil {
		return nil, err
	}

	return stream, nil
}

// HandlePublish handles a publish command (server side)
func HandlePublish(conn *Conn, msg *transport.Message) error {
	cmd, err := DecodeCommand(msg.Data())
	if err != nil {
		return fmt.Errorf("failed to decode publish command: %w", err)
	}

	publishCmd, err := ParsePublish(cmd)
	if err != nil {
		return fmt.Errorf("failed to parse publish: %w", err)
	}

	// 메시지 스트림 ID로 스트림 조회
	streamID := msg.StreamID()
	stream := conn.GetStream(streamID)
	if stream == nil {
		return fmt.Errorf("stream not found: %d", streamID)
	}

	// 스트림 정보 설정
	stream.SetKey(publishCmd.StreamKey)
	stream.SetMode(StreamModePublish)

	return SendOnStatus(conn, streamID, "status", "NetStream.Publish.Start", "Publishing")
}

// HandlePlay handles a play command (server side)
func HandlePlay(conn *Conn, msg *transport.Message) error {
	cmd, err := DecodeCommand(msg.Data())
	if err != nil {
		return fmt.Errorf("failed to decode play command: %w", err)
	}

	playCmd, err := ParsePlay(cmd)
	if err != nil {
		return fmt.Errorf("failed to parse play: %w", err)
	}

	// 메시지 스트림 ID로 스트림 조회
	streamID := msg.StreamID()
	stream := conn.GetStream(streamID)
	if stream == nil {
		return fmt.Errorf("stream not found: %d", streamID)
	}

	// 스트림 정보 설정
	stream.SetKey(playCmd.StreamKey)
	stream.SetMode(StreamModePlay)

	return SendOnStatus(conn, streamID, "status", "NetStream.Play.Start", "Playing")
}
