package rtmp

import (
	"bytes"
	"fmt"

	"github.com/ssungk/ertmp/pkg/amf"
	"github.com/ssungk/ertmp/pkg/rtmp/transport"
)

// Command represents an RTMP command (connect, publish, play, etc.)
type Command struct {
	Name          string
	TransactionID float64
	Object        map[string]interface{}
	Arguments     []interface{}
}

// ConnectCommand represents a connect command
type ConnectCommand struct {
	App            string
	TcUrl          string
	FlashVer       string
	ObjectEncoding float64
	FourCcList     []string
	CapsEx         map[string]interface{}
}

// PublishCommand represents a publish command
type PublishCommand struct {
	StreamKey   string
	PublishType string // "live", "record", "append"
}

// PlayCommand represents a play command
type PlayCommand struct {
	StreamKey string
	Start     float64
	Duration  float64
	Reset     bool
}

// DecodeCommand decodes AMF0 command from message data
func DecodeCommand(data []byte) (*Command, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty command data")
	}

	reader := bytes.NewReader(data)
	values, err := amf.DecodeAMF0Sequence(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode AMF0: %w", err)
	}

	if len(values) < 2 {
		return nil, fmt.Errorf("invalid command: need at least 2 values")
	}

	cmd := &Command{}

	// 커맨드 이름 (문자열)
	name, ok := values[0].(string)
	if !ok {
		return nil, fmt.Errorf("command name must be string")
	}
	cmd.Name = name

	// 트랜잭션 ID (숫자)
	txID, ok := values[1].(float64)
	if !ok {
		return nil, fmt.Errorf("transaction ID must be number")
	}
	cmd.TransactionID = txID

	// 커맨드 객체 (선택사항)
	if len(values) > 2 {
		if obj, ok := values[2].(map[string]interface{}); ok {
			cmd.Object = obj
		}
	}

	// 추가 인자들
	if len(values) > 3 {
		cmd.Arguments = values[3:]
	}

	return cmd, nil
}

// EncodeCommand encodes a command to AMF0 bytes
func EncodeCommand(name string, txID float64, obj map[string]interface{}, args ...interface{}) ([]byte, error) {
	values := []interface{}{name, txID}
	if obj != nil {
		values = append(values, obj)
	} else {
		values = append(values, nil)
	}
	values = append(values, args...)

	data, err := amf.EncodeAMF0Sequence(values...)
	if err != nil {
		return nil, fmt.Errorf("failed to encode command: %w", err)
	}

	return data, nil
}

// ParseConnect parses a connect command
func ParseConnect(cmd *Command) (*ConnectCommand, error) {
	if cmd.Name != "connect" {
		return nil, fmt.Errorf("not a connect command: %s", cmd.Name)
	}

	cc := &ConnectCommand{}

	if cmd.Object != nil {
		if v, ok := cmd.Object["app"].(string); ok {
			cc.App = v
		}
		if v, ok := cmd.Object["tcUrl"].(string); ok {
			cc.TcUrl = v
		}
		if v, ok := cmd.Object["flashVer"].(string); ok {
			cc.FlashVer = v
		}
		if v, ok := cmd.Object["objectEncoding"].(float64); ok {
			cc.ObjectEncoding = v
		}

		// Enhanced RTMP 기능
		if fourCcList, ok := cmd.Object["fourCcList"].([]interface{}); ok {
			for _, fcc := range fourCcList {
				if str, ok := fcc.(string); ok {
					cc.FourCcList = append(cc.FourCcList, str)
				}
			}
		}
		if capsEx, ok := cmd.Object["capsEx"].(map[string]interface{}); ok {
			cc.CapsEx = capsEx
		}
	}

	return cc, nil
}

// ParsePublish parses a publish command
func ParsePublish(cmd *Command) (*PublishCommand, error) {
	if cmd.Name != "publish" {
		return nil, fmt.Errorf("not a publish command: %s", cmd.Name)
	}

	pc := &PublishCommand{
		PublishType: "live", // 기본값
	}

	if len(cmd.Arguments) > 0 {
		if streamKey, ok := cmd.Arguments[0].(string); ok {
			pc.StreamKey = streamKey
		}
	}
	if len(cmd.Arguments) > 1 {
		if publishType, ok := cmd.Arguments[1].(string); ok {
			pc.PublishType = publishType
		}
	}

	return pc, nil
}

// ParsePlay parses a play command
func ParsePlay(cmd *Command) (*PlayCommand, error) {
	if cmd.Name != "play" {
		return nil, fmt.Errorf("not a play command: %s", cmd.Name)
	}

	pc := &PlayCommand{
		Start:    -2, // default: live or recorded
		Duration: -1, // default: play until end
	}

	if len(cmd.Arguments) > 0 {
		if streamKey, ok := cmd.Arguments[0].(string); ok {
			pc.StreamKey = streamKey
		}
	}
	if len(cmd.Arguments) > 1 {
		if start, ok := cmd.Arguments[1].(float64); ok {
			pc.Start = start
		}
	}
	if len(cmd.Arguments) > 2 {
		if duration, ok := cmd.Arguments[2].(float64); ok {
			pc.Duration = duration
		}
	}
	if len(cmd.Arguments) > 3 {
		if reset, ok := cmd.Arguments[3].(bool); ok {
			pc.Reset = reset
		}
	}

	return pc, nil
}

// NewConnectResponseMessage creates a connect response message
func NewConnectResponseMessage(txID float64, props map[string]interface{}) *transport.Message {
	if props == nil {
		props = make(map[string]interface{})
	}

	info := map[string]interface{}{
		"level":       "status",
		"code":        "NetConnection.Connect.Success",
		"description": "Connection succeeded",
	}

	cmdData, _ := EncodeCommand("_result", txID, props, info)
	return transport.NewMessage(0, 0, transport.MsgTypeAMF0Command, cmdData)
}

// NewCreateStreamResponseMessage creates a createStream response message
func NewCreateStreamResponseMessage(txID float64, streamID float64) *transport.Message {
	cmdData, _ := EncodeCommand("_result", txID, nil, streamID)
	return transport.NewMessage(0, 0, transport.MsgTypeAMF0Command, cmdData)
}

// NewOnStatusMessage creates an onStatus command message
func NewOnStatusMessage(streamID uint32, level, code, description string) *transport.Message {
	info := map[string]interface{}{
		"level":       level,
		"code":        code,
		"description": description,
	}

	cmdData, _ := EncodeCommand("onStatus", 0, nil, info)
	return transport.NewMessage(streamID, 0, transport.MsgTypeAMF0Command, cmdData)
}
