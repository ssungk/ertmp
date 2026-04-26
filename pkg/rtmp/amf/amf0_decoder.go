package amf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"time"
)

// decoderBufSize is the largest fixed-size field in AMF0 (float64 / date millis = 8 bytes).
const decoderBufSize = 8

// DefaultLongStrLimit is the default maximum byte length for AMF0 long strings (64MB).
const DefaultLongStrLimit uint32 = 64 * 1024 * 1024 // 64MB

// AMF0Decoder decodes AMF0-encoded byte slices into Go values.
// It is not safe for concurrent use.
type AMF0Decoder struct {
	r            bytes.Reader
	buf          [decoderBufSize]byte // scratch space for fixed-size field reads (float64/uint32/uint16); reused across decode calls
	longStrLimit uint32
}

// NewAMF0Decoder returns a new AMF0Decoder with the default long string limit (64MB).
func NewAMF0Decoder() *AMF0Decoder {
	return &AMF0Decoder{longStrLimit: DefaultLongStrLimit}
}

// LongStrLimit returns the current maximum allowed long string length in bytes.
// A value of 0 means no limit.
func (d *AMF0Decoder) LongStrLimit() uint32 {
	return d.longStrLimit
}

// SetLongStrLimit sets the maximum allowed long string length in bytes.
// A value of 0 means no limit. Default is 64MB.
func (d *AMF0Decoder) SetLongStrLimit(n uint32) {
	d.longStrLimit = n
}

// Decode parses AMF0-encoded data and returns the decoded values.
func (d *AMF0Decoder) Decode(data []byte) ([]any, error) {
	d.r.Reset(data)
	values := make([]any, 0, 5)
	for d.r.Len() > 0 {
		val, err := d.decodeAMF0()
		if err != nil {
			return nil, fmt.Errorf("AMF0 decode failed: %w", err)
		}
		values = append(values, val)
	}
	return values, nil
}

func (d *AMF0Decoder) decodeAMF0() (any, error) {
	marker, err := d.r.ReadByte()
	if err != nil {
		return nil, err
	}

	switch marker {
	case numberMarker:
		return d.decodeNumber()
	case booleanMarker:
		return d.decodeBoolean()
	case stringMarker:
		return d.decodeString()
	case objectMarker:
		return d.decodeObject()
	case nullMarker, undefinedMarker:
		// undefined (0x06) is mapped to nil; callers cannot distinguish from null (0x05)
		return nil, nil
	case referenceMarker:
		return nil, fmt.Errorf("AMF0 reference not implemented")
	case ecmaArrayMarker:
		return d.decodeECMAArray()
	case strictArrayMarker:
		return d.decodeStrictArray()
	case dateMarker:
		return d.decodeDate()
	case longStringMarker:
		return d.decodeLongString()
	case unsupportedMarker:
		return nil, fmt.Errorf("AMF0 unsupported-value type (0x0D) not implemented")
	case xmlDocumentMarker:
		return nil, fmt.Errorf("AMF0 xml-document not implemented")
	case typedObjectMarker:
		return nil, fmt.Errorf("AMF0 typed-object not implemented")
	case avmPlusMarker:
		return nil, fmt.Errorf("AMF0 AMF3 switch not implemented")
	case movieClipMarker, recordSetMarker:
		return nil, fmt.Errorf("AMF0 reserved marker: 0x%02x", marker)
	default:
		return nil, fmt.Errorf("unknown AMF0 marker: 0x%02x", marker)
	}
}

func (d *AMF0Decoder) decodeNumber() (float64, error) {
	if _, err := io.ReadFull(&d.r, d.buf[:8]); err != nil {
		return 0, err
	}
	return math.Float64frombits(binary.BigEndian.Uint64(d.buf[:8])), nil
}

func (d *AMF0Decoder) decodeBoolean() (bool, error) {
	b, err := d.r.ReadByte()
	if err != nil {
		return false, err
	}
	return b != 0, nil
}

func (d *AMF0Decoder) decodeString() (string, error) {
	if _, err := io.ReadFull(&d.r, d.buf[:2]); err != nil {
		return "", err
	}
	length := int(binary.BigEndian.Uint16(d.buf[:2]))
	buf := make([]byte, length)
	if _, err := io.ReadFull(&d.r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func (d *AMF0Decoder) decodeLongString() (string, error) {
	if _, err := io.ReadFull(&d.r, d.buf[:4]); err != nil {
		return "", err
	}
	length := binary.BigEndian.Uint32(d.buf[:4])
	if d.longStrLimit > 0 && length > d.longStrLimit {
		return "", fmt.Errorf("long string too large: %d bytes (max %d)", length, d.longStrLimit)
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(&d.r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func (d *AMF0Decoder) decodeECMAArray() (ECMAArray, error) {
	// AMF0 spec §2.10: associative-count is approximate; parsing ends at the object-end marker
	if _, err := io.ReadFull(&d.r, d.buf[:4]); err != nil {
		return nil, err
	}
	return d.decodeObject()
}

func (d *AMF0Decoder) decodeObject() (map[string]any, error) {
	obj := make(map[string]any)

	for {
		key, err := d.decodeString()
		if err != nil {
			return nil, err
		}
		if len(key) == 0 {
			// AMF0 object-end is 0x00 0x00 0x09: empty key signals end, next byte must be objectEndMarker
			b, err := d.r.ReadByte()
			if err != nil {
				return nil, err
			}
			if b == objectEndMarker {
				break
			}
			return nil, fmt.Errorf("expected object end marker, got 0x%02x", b)
		}
		val, err := d.decodeAMF0()
		if err != nil {
			return nil, err
		}
		obj[key] = val
	}
	return obj, nil
}

func (d *AMF0Decoder) decodeStrictArray() ([]any, error) {
	if _, err := io.ReadFull(&d.r, d.buf[:4]); err != nil {
		return nil, err
	}
	count := binary.BigEndian.Uint32(d.buf[:4])
	if uint64(count) > uint64(d.r.Len()) {
		return nil, fmt.Errorf("AMF0 strict array: count %d exceeds remaining data (%d bytes)", count, d.r.Len())
	}

	arr := make([]any, 0, int(count))
	for i := uint32(0); i < count; i++ {
		v, err := d.decodeAMF0()
		if err != nil {
			return nil, err
		}
		arr = append(arr, v)
	}
	return arr, nil
}

func (d *AMF0Decoder) decodeDate() (time.Time, error) {
	if _, err := io.ReadFull(&d.r, d.buf[:8]); err != nil {
		return time.Time{}, err
	}
	millis := math.Float64frombits(binary.BigEndian.Uint64(d.buf[:8]))
	if math.IsNaN(millis) || math.IsInf(millis, 0) {
		return time.Time{}, fmt.Errorf("AMF0 date: invalid millis %v", millis)
	}
	if millis >= float64(math.MaxInt64) || millis < float64(math.MinInt64) {
		return time.Time{}, fmt.Errorf("AMF0 date: millis out of int64 range: %v", millis)
	}

	// AMF0 spec §2.13: timezone offset is deprecated; always ignored, UTC is used
	if _, err := io.ReadFull(&d.r, d.buf[:2]); err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(int64(millis)).UTC(), nil
}
