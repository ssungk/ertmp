package amf

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestEncodeAMF0Sequence_Success(t *testing.T) {
	values := []any{3.14, true, "hello", map[string]any{"foo": "bar"}}
	data, err := EncodeAMF0Sequence(values...)
	if err != nil {
		t.Fatal(err)
	}

	if len(data) == 0 {
		t.Fatal("expected non-empty encoded data")
	}

	decoded, err := DecodeAMF0Sequence(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(decoded) != len(values) {
		t.Errorf("expected %d values, got %d", len(values), len(decoded))
	}
}

func TestEncodeAMF0Sequence_Error(t *testing.T) {
	type unsupportedType struct{}
	_, err := EncodeAMF0Sequence(unsupportedType{})
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestEncodeAMF0_Number(t *testing.T) {
	data, err := EncodeAMF0Sequence(3.14)
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte{0x00, 0x40, 0x09, 0x1e, 0xb8, 0x51, 0xeb, 0x85, 0x1f}
	if !bytes.Equal(data, expected) {
		t.Errorf("expected %v, got %v", expected, data)
	}
}

func TestEncodeValue_Float32(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := encodeValue(buf, float32(3.14)); err != nil {
		t.Fatal(err)
	}
	if buf.Bytes()[0] != numberMarker {
		t.Errorf("expected numberMarker, got 0x%02x", buf.Bytes()[0])
	}
}

func TestEncodeValue_Int(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := encodeValue(buf, int(42)); err != nil {
		t.Fatal(err)
	}
	if buf.Bytes()[0] != numberMarker {
		t.Errorf("expected numberMarker, got 0x%02x", buf.Bytes()[0])
	}
}

func TestEncodeValue_Int32(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := encodeValue(buf, int32(42)); err != nil {
		t.Fatal(err)
	}
	if buf.Bytes()[0] != numberMarker {
		t.Errorf("expected numberMarker, got 0x%02x", buf.Bytes()[0])
	}
}

func TestEncodeValue_Int64(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := encodeValue(buf, int64(42)); err != nil {
		t.Fatal(err)
	}
	if buf.Bytes()[0] != numberMarker {
		t.Errorf("expected numberMarker, got 0x%02x", buf.Bytes()[0])
	}
}

func TestEncodeValue_Uint(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := encodeValue(buf, uint(42)); err != nil {
		t.Fatal(err)
	}
	if buf.Bytes()[0] != numberMarker {
		t.Errorf("expected numberMarker, got 0x%02x", buf.Bytes()[0])
	}
}

func TestEncodeValue_Uint32(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := encodeValue(buf, uint32(42)); err != nil {
		t.Fatal(err)
	}
	if buf.Bytes()[0] != numberMarker {
		t.Errorf("expected numberMarker, got 0x%02x", buf.Bytes()[0])
	}
}

func TestEncodeValue_Uint64(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := encodeValue(buf, uint64(42)); err != nil {
		t.Fatal(err)
	}
	if buf.Bytes()[0] != numberMarker {
		t.Errorf("expected numberMarker, got 0x%02x", buf.Bytes()[0])
	}
}

func TestEncodeAMF0_Boolean(t *testing.T) {
	data, err := EncodeAMF0Sequence(true)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte{0x01, 0x01}) {
		t.Errorf("expected [0x01 0x01] for true, got %v", data)
	}

	data, err = EncodeAMF0Sequence(false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte{0x01, 0x00}) {
		t.Errorf("expected [0x01 0x00] for false, got %v", data)
	}
}

func TestEncodeAMF0_String(t *testing.T) {
	data, err := EncodeAMF0Sequence("hello")
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte{0x02, 0x00, 0x05, 'h', 'e', 'l', 'l', 'o'}
	if !bytes.Equal(data, expected) {
		t.Errorf("expected %v, got %v", expected, data)
	}
}

func TestEncodeAMF0_String_Empty(t *testing.T) {
	data, err := EncodeAMF0Sequence("")
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte{0x02, 0x00, 0x00}
	if !bytes.Equal(data, expected) {
		t.Errorf("expected %v, got %v", expected, data)
	}
}

func TestEncodeAMF0_LongString(t *testing.T) {
	longStr := strings.Repeat("a", 70000)
	data, err := EncodeAMF0Sequence(longStr)
	if err != nil {
		t.Fatal(err)
	}

	if data[0] != longStringMarker {
		t.Errorf("expected longStringMarker (0x%02x), got 0x%02x", longStringMarker, data[0])
	}
	if len(data) != 70005 {
		t.Errorf("expected 70005 bytes, got %d", len(data))
	}
}

func TestEncodeAMF0_Object(t *testing.T) {
	obj := map[string]any{"foo": "bar"}
	data, err := EncodeAMF0Sequence(obj)
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte{
		0x03,
		0x00, 0x03, 'f', 'o', 'o',
		0x02, 0x00, 0x03, 'b', 'a', 'r',
		0x00, 0x00, 0x09,
	}
	if !bytes.Equal(data, expected) {
		t.Errorf("expected %v, got %v", expected, data)
	}
}

func TestEncodeAMF0_Object_UnsupportedValue(t *testing.T) {
	type unsupportedType struct{}
	_, err := EncodeAMF0Sequence(map[string]any{"key": unsupportedType{}})
	if err == nil {
		t.Fatal("expected error for unsupported value in object")
	}
}

func TestEncodeAMF0_Object_Empty(t *testing.T) {
	data, err := EncodeAMF0Sequence(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte{0x03, 0x00, 0x00, 0x09}
	if !bytes.Equal(data, expected) {
		t.Errorf("expected %v, got %v", expected, data)
	}
}

func TestEncodeObjectProperty_KeyTooLong(t *testing.T) {
	buf := new(bytes.Buffer)
	err := encodeObjectProperty(buf, strings.Repeat("a", 70000), "value")
	if err == nil {
		t.Fatal("expected error for key too long")
	}
	if !strings.Contains(err.Error(), "object key too long") {
		t.Errorf("expected 'object key too long', got %v", err.Error())
	}
}

func TestEncodeObjectProperty_ValueError(t *testing.T) {
	buf := new(bytes.Buffer)
	type unsupportedType struct{}
	err := encodeObjectProperty(buf, "key", unsupportedType{})
	if err == nil {
		t.Fatal("expected value encode error")
	}
	if !strings.Contains(err.Error(), "unsupported AMF0 type") {
		t.Errorf("expected 'unsupported AMF0 type', got %v", err.Error())
	}
}

func TestEncodeAMF0_Null(t *testing.T) {
	data, err := EncodeAMF0Sequence(nil)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, []byte{0x05}) {
		t.Errorf("expected [0x05], got %v", data)
	}
}

func TestEncodeAMF0_StrictArray(t *testing.T) {
	data, err := EncodeAMF0Sequence([]any{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte{
		0x0A,
		0x00, 0x00, 0x00, 0x02,
		0x02, 0x00, 0x01, 'a',
		0x02, 0x00, 0x01, 'b',
	}
	if !bytes.Equal(data, expected) {
		t.Errorf("expected %v, got %v", expected, data)
	}
}

func TestEncodeAMF0_StrictArray_Empty(t *testing.T) {
	data, err := EncodeAMF0Sequence([]any{})
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte{0x0A, 0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(data, expected) {
		t.Errorf("expected %v, got %v", expected, data)
	}
}

func TestEncodeAMF0_StrictArray_UnsupportedElement(t *testing.T) {
	type unsupportedType struct{}
	_, err := EncodeAMF0Sequence([]any{unsupportedType{}})
	if err == nil {
		t.Fatal("expected error for unsupported element type")
	}
}

func TestEncodeAMF0_Date(t *testing.T) {
	date := time.Date(2023, 3, 28, 19, 40, 0, 123*1e6, time.UTC)
	data, err := EncodeAMF0Sequence(date)
	if err != nil {
		t.Fatal(err)
	}

	if len(data) != 11 {
		t.Errorf("expected 11 bytes for date, got %d", len(data))
	}
	if data[0] != dateMarker {
		t.Errorf("expected dateMarker (0x%02x), got 0x%02x", dateMarker, data[0])
	}
}

func TestEncodeAMF0_UnsupportedType(t *testing.T) {
	type customType struct{ field string }
	_, err := EncodeAMF0Sequence(customType{field: "test"})
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
	if !strings.Contains(err.Error(), "unsupported AMF0 type") {
		t.Errorf("expected 'unsupported AMF0 type', got %v", err.Error())
	}
}

func TestEncodeAMF0_RoundTrip(t *testing.T) {
	testCases := []any{
		3.14,
		true,
		false,
		"hello world",
		nil,
		[]any{1.0, 2.0, 3.0},
		map[string]any{
			"name":  "test",
			"value": 123.45,
			"flag":  true,
		},
	}

	for i, original := range testCases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			encoded, err := EncodeAMF0Sequence(original)
			if err != nil {
				t.Fatalf("encoding failed: %v", err)
			}

			decoded, err := DecodeAMF0Sequence(bytes.NewReader(encoded))
			if err != nil {
				t.Fatalf("decoding failed: %v", err)
			}

			if len(decoded) != 1 {
				t.Fatalf("expected 1 decoded value, got %d", len(decoded))
			}

			switch orig := original.(type) {
			case float64:
				if decoded[0] != orig {
					t.Errorf("expected %v, got %v", orig, decoded[0])
				}
			case bool:
				if decoded[0] != orig {
					t.Errorf("expected %v, got %v", orig, decoded[0])
				}
			case string:
				if decoded[0] != orig {
					t.Errorf("expected %v, got %v", orig, decoded[0])
				}
			case nil:
				if decoded[0] != nil {
					t.Errorf("expected nil, got %v", decoded[0])
				}
			default:
				if decoded[0] == nil {
					t.Errorf("decoded value is nil")
				}
			}
		})
	}
}

func BenchmarkEncodeAMF0_Number(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = EncodeAMF0Sequence(3.14)
	}
}

func BenchmarkEncodeAMF0_String(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = EncodeAMF0Sequence("hello world")
	}
}

func BenchmarkEncodeAMF0_Object(b *testing.B) {
	obj := map[string]any{
		"name":  "test",
		"value": 123.45,
		"flag":  true,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EncodeAMF0Sequence(obj)
	}
}
