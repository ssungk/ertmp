package amf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// AMF0Encoder is not safe for concurrent use.
type AMF0Encoder struct {
	buf bytes.Buffer
}

func NewAMF0Encoder() *AMF0Encoder {
	return &AMF0Encoder{}
}

func (e *AMF0Encoder) Encode(values ...any) ([]byte, error) {
	e.buf.Reset()
	for _, val := range values {
		if err := e.encodeValue(val); err != nil {
			e.buf.Reset()
			return nil, fmt.Errorf("AMF0 encode failed: %w", err)
		}
	}
	result := make([]byte, e.buf.Len())
	copy(result, e.buf.Bytes()) // independent copy: buf is reused on next Encode call
	return result, nil
}

func (e *AMF0Encoder) encodeValue(value any) error {
	switch v := value.(type) {
	case nil:
		e.buf.WriteByte(nullMarker)
	case bool:
		b := byte(0)
		if v {
			b = 1
		}
		e.buf.WriteByte(booleanMarker)
		e.buf.WriteByte(b)
	case float64:
		e.writeNumber(v)
	case float32:
		e.writeNumber(float64(v))
	case int:
		e.writeNumber(float64(v))
	case int8:
		e.writeNumber(float64(v))
	case int16:
		e.writeNumber(float64(v))
	case int32:
		e.writeNumber(float64(v))
	case int64:
		e.writeNumber(float64(v))
	case uint:
		e.writeNumber(float64(v))
	case uint8:
		e.writeNumber(float64(v))
	case uint16:
		e.writeNumber(float64(v))
	case uint32:
		e.writeNumber(float64(v))
	case uint64:
		e.writeNumber(float64(v)) // AMF0 Number is IEEE 754 double; values > 2^53 lose precision
	case string:
		e.encodeString(v)
	case ECMAArray:
		return e.encodeECMAArray(v)
	case map[string]any:
		return e.encodeObject(v)
	case []any:
		return e.encodeStrictArray(v)
	case time.Time:
		e.encodeDate(v)
	default:
		return fmt.Errorf("unsupported AMF0 type: %T", value)
	}
	return nil
}

func (e *AMF0Encoder) writeNumber(v float64) {
	var b [9]byte
	b[0] = numberMarker
	binary.BigEndian.PutUint64(b[1:], math.Float64bits(v))
	e.buf.Write(b[:])
}

func (e *AMF0Encoder) encodeString(s string) {
	byteLen := len(s)
	if byteLen <= 65535 {
		var b [3]byte
		b[0] = stringMarker
		binary.BigEndian.PutUint16(b[1:], uint16(byteLen))
		e.buf.Write(b[:])
	} else {
		var b [5]byte
		b[0] = longStringMarker
		binary.BigEndian.PutUint32(b[1:], uint32(byteLen))
		e.buf.Write(b[:])
	}
	e.buf.WriteString(s)
}

func (e *AMF0Encoder) encodeECMAArray(arr ECMAArray) error {
	// uint32(len(arr)) never truncates silently: ECMAArray with 4B+ entries requires >64GB memory, which is unreachable in practice
	var b [5]byte
	b[0] = ecmaArrayMarker
	binary.BigEndian.PutUint32(b[1:], uint32(len(arr)))
	e.buf.Write(b[:])

	return e.encodeProperties(arr)
}

func (e *AMF0Encoder) encodeObject(obj map[string]any) error {
	e.buf.WriteByte(objectMarker)
	return e.encodeProperties(obj)
}

func (e *AMF0Encoder) encodeProperties(m map[string]any) error {
	for key, val := range m {
		if err := e.encodeObjectProperty(key, val); err != nil {
			return err
		}
	}
	e.buf.Write([]byte{0x00, 0x00, objectEndMarker})
	return nil
}

func (e *AMF0Encoder) encodeObjectProperty(key string, val any) error {
	keyByteLen := len(key)
	if keyByteLen == 0 {
		return fmt.Errorf("object key must not be empty")
	}
	if keyByteLen > 65535 {
		return fmt.Errorf("object key too long: %d bytes (max 65535)", keyByteLen)
	}

	var b [2]byte
	binary.BigEndian.PutUint16(b[:], uint16(keyByteLen))
	e.buf.Write(b[:])
	e.buf.WriteString(key)
	return e.encodeValue(val)
}

func (e *AMF0Encoder) encodeStrictArray(arr []any) error {
	// uint32(len(arr)) never truncates silently: []any with 4B+ elements requires >64GB memory, which is unreachable in practice
	var b [5]byte
	b[0] = strictArrayMarker
	binary.BigEndian.PutUint32(b[1:], uint32(len(arr)))
	e.buf.Write(b[:])

	for _, v := range arr {
		if err := e.encodeValue(v); err != nil {
			return err
		}
	}
	return nil
}

func (e *AMF0Encoder) encodeDate(t time.Time) {
	var b [11]byte
	b[0] = dateMarker
	binary.BigEndian.PutUint64(b[1:], math.Float64bits(float64(t.UnixMilli()))) // AMF0 Number is IEEE 754 double; UnixMilli() values > 2^53 lose precision
	// b[9:11] = 0x0000: timezone offset, deprecated and always 0 per AMF0 spec §2.13
	e.buf.Write(b[:])
}
