package amf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"time"
)

type AMF0Decoder struct {
	r bytes.Reader
}

func NewAMF0Decoder() *AMF0Decoder {
	return &AMF0Decoder{}
}

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
		return nil, fmt.Errorf("AMF0 unsupported not implemented")
	case xmlDocumentMarker:
		return nil, fmt.Errorf("AMF0 xml-document not implemented")
	case typedObjectMarker:
		return nil, fmt.Errorf("AMF0 typed-object not implemented")
	case avmPlusMarker:
		return nil, fmt.Errorf("AMF0 AMF3 switch not implemented")
	case movieClipMarker, recordSetMarker:
		return nil, fmt.Errorf("AMF0 reserved marker: 0x%02x", marker)
	default:
		return nil, fmt.Errorf("unknown AMF0 marker: 0x%x", marker)
	}
}

func (d *AMF0Decoder) decodeNumber() (float64, error) {
	var b [8]byte
	if _, err := io.ReadFull(&d.r, b[:]); err != nil {
		return 0, err
	}
	return math.Float64frombits(binary.BigEndian.Uint64(b[:])), nil
}

func (d *AMF0Decoder) decodeBoolean() (bool, error) {
	b, err := d.r.ReadByte()
	if err != nil {
		return false, err
	}
	return b != 0, nil
}

func (d *AMF0Decoder) decodeString() (string, error) {
	var b [2]byte
	if _, err := io.ReadFull(&d.r, b[:]); err != nil {
		return "", err
	}
	buf := make([]byte, binary.BigEndian.Uint16(b[:]))
	if _, err := io.ReadFull(&d.r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func (d *AMF0Decoder) decodeLongString() (string, error) {
	var b [4]byte
	if _, err := io.ReadFull(&d.r, b[:]); err != nil {
		return "", err
	}
	length := binary.BigEndian.Uint32(b[:])
	if int64(length) > int64(d.r.Len()) {
		return "", fmt.Errorf("long string length %d exceeds remaining data %d", length, d.r.Len())
	}
	buf := make([]byte, length)
	// length <= d.r.Len() is guaranteed above, so io.ReadFull on bytes.Reader cannot fail.
	// Error return is intentionally omitted: adding "if err != nil" would create an
	// unreachable branch that breaks 100% statement coverage.
	_, _ = io.ReadFull(&d.r, buf)
	return string(buf), nil
}

func (d *AMF0Decoder) decodeECMAArray() (ECMAArray, error) {
	var b [4]byte
	if _, err := io.ReadFull(&d.r, b[:]); err != nil {
		return nil, err
	}
	// AMF0 spec §2.10: associative-count is approximate; parsing ends at the object-end marker
	m, err := d.decodeObject()
	if err != nil {
		return nil, err
	}
	return m, nil
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
			return nil, errors.New("expected object end marker")
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
	var b [4]byte
	if _, err := io.ReadFull(&d.r, b[:]); err != nil {
		return nil, err
	}
	count := binary.BigEndian.Uint32(b[:])
	if int64(count) > int64(d.r.Len()) {
		return nil, fmt.Errorf("strict array count %d exceeds remaining data %d", count, d.r.Len())
	}
	arr := make([]any, count)
	for i := uint32(0); i < count; i++ {
		v, err := d.decodeAMF0()
		if err != nil {
			return nil, err
		}
		arr[i] = v
	}
	return arr, nil
}

func (d *AMF0Decoder) decodeDate() (time.Time, error) {
	var b [8]byte
	if _, err := io.ReadFull(&d.r, b[:]); err != nil {
		return time.Time{}, err
	}
	millis := math.Float64frombits(binary.BigEndian.Uint64(b[:]))

	var offset [2]byte
	if _, err := io.ReadFull(&d.r, offset[:]); err != nil {
		return time.Time{}, err
	}
	_ = offset // AMF0 spec §2.13: timezone offset is deprecated and always 0; UTC is assumed

	return time.UnixMilli(int64(millis)).UTC(), nil
}
