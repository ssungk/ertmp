package rtmp

import (
	"fmt"

	"github.com/ssungk/ertmp/pkg/rtmp/buf"
	"github.com/ssungk/ertmp/pkg/rtmp/transport"
)

// SendConnectResponse sends a connect response
func SendConnectResponse(conn *Conn, txID float64, props map[string]interface{}) error {
	msg := NewConnectResponseMessage(txID, props)
	defer msg.Buffer().Release()
	return conn.WriteMessage(msg)
}

// SendCreateStreamResponse sends a createStream response
func SendCreateStreamResponse(conn *Conn, txID, streamID float64) error {
	msg := NewCreateStreamResponseMessage(txID, streamID)
	defer msg.Buffer().Release()
	return conn.WriteMessage(msg)
}

// SendOnStatus sends an onStatus message
func SendOnStatus(conn *Conn, streamID uint32, level, code, description string) error {
	msg := NewOnStatusMessage(streamID, level, code, description)
	defer msg.Buffer().Release()
	return conn.WriteMessage(msg)
}

// SendVideo sends video data
func SendVideo(conn *Conn, streamID uint32, data []byte, timestamp uint32) error {
	buffer := buf.New(data)
	header := transport.NewMessageHeader(streamID, timestamp, transport.MsgTypeVideo)
	msg := transport.NewMessage(header, buffer)
	defer msg.Buffer().Release()
	return conn.WriteMessage(msg)
}

// SendAudio sends audio data
func SendAudio(conn *Conn, streamID uint32, data []byte, timestamp uint32) error {
	buffer := buf.New(data)
	header := transport.NewMessageHeader(streamID, timestamp, transport.MsgTypeAudio)
	msg := transport.NewMessage(header, buffer)
	defer msg.Buffer().Release()
	return conn.WriteMessage(msg)
}

// SendMetadata sends metadata
func SendMetadata(conn *Conn, streamID uint32, metadata map[string]interface{}) error {
	// 메타데이터 인코딩
	cmdData, err := EncodeCommand("@setDataFrame", 0, nil, "onMetaData", metadata)
	if err != nil {
		return fmt.Errorf("failed to encode metadata: %w", err)
	}
	buffer := buf.New(cmdData)
	header := transport.NewMessageHeader(streamID, 0, transport.MsgTypeAMF0Data)
	msg := transport.NewMessage(header, buffer)
	defer msg.Buffer().Release()
	return conn.WriteMessage(msg)
}
