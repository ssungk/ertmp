package amf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

func EncodeAMF0Sequence(values ...any) ([]byte, error) {
	buf := new(bytes.Buffer)
	for _, val := range values {
		if err := encodeValue(buf, val); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func encodeValue(buf *bytes.Buffer, value any) error {
	switch v := value.(type) {
	case nil:
		buf.WriteByte(nullMarker)
	case bool:
		b := byte(0)
		if v {
			b = 1
		}
		buf.Write([]byte{booleanMarker, b})
	case float64:
		writeNumber(buf, v)
	case float32:
		writeNumber(buf, float64(v))
	case int:
		writeNumber(buf, float64(v))
	case int32:
		writeNumber(buf, float64(v))
	case int64:
		writeNumber(buf, float64(v))
	case uint:
		writeNumber(buf, float64(v))
	case uint32:
		writeNumber(buf, float64(v))
	case uint64:
		writeNumber(buf, float64(v))
	case string:
		encodeString(buf, v)
	case map[string]any:
		return encodeObject(buf, v)
	case []any:
		return encodeStrictArray(buf, v)
	case time.Time:
		encodeDate(buf, v)
	default:
		return fmt.Errorf("unsupported AMF0 type: %T", value)
	}
	return nil
}

func writeNumber(buf *bytes.Buffer, v float64) {
	var b [9]byte
	b[0] = numberMarker
	binary.BigEndian.PutUint64(b[1:], math.Float64bits(v))
	buf.Write(b[:])
}

func encodeString(buf *bytes.Buffer, s string) {
	byteLen := len(s)
	if byteLen < 65536 {
		var b [3]byte
		b[0] = stringMarker
		binary.BigEndian.PutUint16(b[1:], uint16(byteLen))
		buf.Write(b[:])
	} else {
		var b [5]byte
		b[0] = longStringMarker
		binary.BigEndian.PutUint32(b[1:], uint32(byteLen))
		buf.Write(b[:])
	}
	buf.WriteString(s)
}

func encodeObject(buf *bytes.Buffer, obj map[string]any) error {
	buf.WriteByte(objectMarker)
	for key, val := range obj {
		if err := encodeObjectProperty(buf, key, val); err != nil {
			return err
		}
	}
	buf.Write([]byte{0x00, 0x00, objectEndMarker})
	return nil
}

func encodeObjectProperty(buf *bytes.Buffer, key string, val any) error {
	keyByteLen := len(key)
	if keyByteLen > 65535 {
		return fmt.Errorf("object key too long: %d bytes (max 65535)", keyByteLen)
	}
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], uint16(keyByteLen))
	buf.Write(b[:])
	buf.WriteString(key)
	return encodeValue(buf, val)
}

func encodeStrictArray(buf *bytes.Buffer, arr []any) error {
	var b [5]byte
	b[0] = strictArrayMarker
	binary.BigEndian.PutUint32(b[1:], uint32(len(arr)))
	buf.Write(b[:])
	for _, v := range arr {
		if err := encodeValue(buf, v); err != nil {
			return err
		}
	}
	return nil
}

func encodeDate(buf *bytes.Buffer, t time.Time) {
	var b [11]byte
	b[0] = dateMarker
	binary.BigEndian.PutUint64(b[1:], math.Float64bits(float64(t.UnixNano())/1e6))
	// b[9], b[10] = 0x00, 0x00 (timezone, always 0)
	buf.Write(b[:])
}
