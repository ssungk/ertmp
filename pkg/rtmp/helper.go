package rtmp

import (
	"encoding/binary"
	"fmt"

	"github.com/ssungk/ertmp/pkg/rtmp/buf"
	"github.com/ssungk/ertmp/pkg/rtmp/transport"
)

// SendConnectResponse sends a connect response
func SendConnectResponse(conn *Conn, txID float64, props map[string]interface{}) error {
	msg := NewConnectResponseMessage(txID, props)
	defer msg.Release()
	return conn.WriteMessage(msg)
}

// SendCreateStreamResponse sends a createStream response
func SendCreateStreamResponse(conn *Conn, txID, streamID float64) error {
	msg := NewCreateStreamResponseMessage(txID, streamID)
	defer msg.Release()
	return conn.WriteMessage(msg)
}

// SendOnStatus sends an onStatus message
func SendOnStatus(conn *Conn, streamID uint32, level, code, description string) error {
	msg := NewOnStatusMessage(streamID, level, code, description)
	defer msg.Release()
	return conn.WriteMessage(msg)
}

// SendVideo sends video data
func SendVideo(conn *Conn, streamID uint32, data []byte, timestamp uint32) error {
	header := transport.NewMessageHeader(streamID, timestamp, transport.MsgTypeVideo)
	msg := transport.NewMessage(header, data)
	defer msg.Release()
	return conn.WriteMessage(msg)
}

// SendAudio sends audio data
func SendAudio(conn *Conn, streamID uint32, data []byte, timestamp uint32) error {
	header := transport.NewMessageHeader(streamID, timestamp, transport.MsgTypeAudio)
	msg := transport.NewMessage(header, data)
	defer msg.Release()
	return conn.WriteMessage(msg)
}

// SendMetadata sends metadata
func SendMetadata(conn *Conn, streamID uint32, metadata map[string]interface{}) error {
	// 메타데이터 인코딩
	cmdData, err := EncodeCommand("@setDataFrame", 0, nil, "onMetaData", metadata)
	if err != nil {
		return fmt.Errorf("failed to encode metadata: %w", err)
	}
	header := transport.NewMessageHeader(streamID, 0, transport.MsgTypeAMF0Data)
	msg := transport.NewMessage(header, cmdData)
	defer msg.Release()
	return conn.WriteMessage(msg)
}

// SendSetChunkSize sends a SetChunkSize message
func SendSetChunkSize(conn *Conn, size uint32) error {
	buffer := buf.NewPooled(4)
	binary.BigEndian.PutUint32(buffer.Data(), size&0x7FFFFFFF)
	header := transport.NewMessageHeader(0, 0, transport.MsgTypeSetChunkSize)
	msg := transport.NewMessageFromBuffer(header, buffer)
	defer msg.Release()

	// 전송 후 transport의 outgoing 청크 크기 업데이트
	if err := conn.WriteMessage(msg); err != nil {
		return err
	}
	return conn.transport.SetOutChunkSize(size)
}

// SendWindowAckSize sends a WindowAckSize message
func SendWindowAckSize(conn *Conn, size uint32) error {
	buffer := buf.NewPooled(4)
	binary.BigEndian.PutUint32(buffer.Data(), size)
	header := transport.NewMessageHeader(0, 0, transport.MsgTypeWindowAckSize)
	msg := transport.NewMessageFromBuffer(header, buffer)
	defer msg.Release()
	return conn.WriteMessage(msg)
}

// SendSetPeerBW sends a SetPeerBandwidth message
func SendSetPeerBW(conn *Conn, size uint32, limitType uint8) error {
	buffer := buf.NewPooled(5)
	binary.BigEndian.PutUint32(buffer.Data(), size)
	buffer.Data()[4] = limitType
	header := transport.NewMessageHeader(0, 0, transport.MsgTypeSetPeerBW)
	msg := transport.NewMessageFromBuffer(header, buffer)
	defer msg.Release()
	return conn.WriteMessage(msg)
}

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
