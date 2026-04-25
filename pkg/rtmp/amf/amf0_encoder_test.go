package amf

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)


func TestAMF0Encoder_Reuse(t *testing.T) {
	enc := NewAMF0Encoder()

	first, err := enc.Encode(3.14)
	if err != nil {
		t.Fatal(err)
	}
	second, err := enc.Encode("hello")
	if err != nil {
		t.Fatal(err)
	}

	wantFirst := []byte{0x00, 0x40, 0x09, 0x1E, 0xB8, 0x51, 0xEB, 0x85, 0x1F}
	wantSecond := []byte{0x02, 0x00, 0x05, 'h', 'e', 'l', 'l', 'o'}

	if !bytes.Equal(first, wantFirst) {
		t.Errorf("first encode: got %v, want %v", first, wantFirst)
	}
	if !bytes.Equal(second, wantSecond) {
		t.Errorf("second encode: got %v, want %v", second, wantSecond)
	}
}

func TestAMF0Encoder_Error(t *testing.T) {
	enc := NewAMF0Encoder()
	type unsupportedType struct{}
	_, err := enc.Encode(unsupportedType{})
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestEncodeAMF0_Number(t *testing.T) {
	data, err := NewAMF0Encoder().Encode(3.14)
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte{0x00, 0x40, 0x09, 0x1e, 0xb8, 0x51, 0xeb, 0x85, 0x1f}
	if !bytes.Equal(data, expected) {
		t.Errorf("expected %v, got %v", expected, data)
	}
}

func TestEncodeValue_Float32(t *testing.T) {
	data, err := NewAMF0Encoder().Encode(float32(3.14))
	if err != nil {
		t.Fatal(err)
	}
	if data[0] != numberMarker {
		t.Errorf("expected numberMarker, got 0x%02x", data[0])
	}
}

func TestEncodeValue_Int(t *testing.T) {
	data, err := NewAMF0Encoder().Encode(int(42))
	if err != nil {
		t.Fatal(err)
	}
	if data[0] != numberMarker {
		t.Errorf("expected numberMarker, got 0x%02x", data[0])
	}
}

func TestEncodeValue_Int32(t *testing.T) {
	data, err := NewAMF0Encoder().Encode(int32(42))
	if err != nil {
		t.Fatal(err)
	}
	if data[0] != numberMarker {
		t.Errorf("expected numberMarker, got 0x%02x", data[0])
	}
}

func TestEncodeValue_Int64(t *testing.T) {
	data, err := NewAMF0Encoder().Encode(int64(42))
	if err != nil {
		t.Fatal(err)
	}
	if data[0] != numberMarker {
		t.Errorf("expected numberMarker, got 0x%02x", data[0])
	}
}

func TestEncodeValue_Uint(t *testing.T) {
	data, err := NewAMF0Encoder().Encode(uint(42))
	if err != nil {
		t.Fatal(err)
	}
	if data[0] != numberMarker {
		t.Errorf("expected numberMarker, got 0x%02x", data[0])
	}
}

func TestEncodeValue_Uint32(t *testing.T) {
	data, err := NewAMF0Encoder().Encode(uint32(42))
	if err != nil {
		t.Fatal(err)
	}
	if data[0] != numberMarker {
		t.Errorf("expected numberMarker, got 0x%02x", data[0])
	}
}

func TestEncodeValue_Uint64(t *testing.T) {
	data, err := NewAMF0Encoder().Encode(uint64(42))
	if err != nil {
		t.Fatal(err)
	}
	if data[0] != numberMarker {
		t.Errorf("expected numberMarker, got 0x%02x", data[0])
	}
}

func TestEncodeValue_Int8(t *testing.T) {
	data, err := NewAMF0Encoder().Encode(int8(42))
	if err != nil {
		t.Fatal(err)
	}
	if data[0] != numberMarker {
		t.Errorf("expected numberMarker, got 0x%02x", data[0])
	}
}

func TestEncodeValue_Int16(t *testing.T) {
	data, err := NewAMF0Encoder().Encode(int16(42))
	if err != nil {
		t.Fatal(err)
	}
	if data[0] != numberMarker {
		t.Errorf("expected numberMarker, got 0x%02x", data[0])
	}
}

func TestEncodeValue_Uint8(t *testing.T) {
	data, err := NewAMF0Encoder().Encode(uint8(42))
	if err != nil {
		t.Fatal(err)
	}
	if data[0] != numberMarker {
		t.Errorf("expected numberMarker, got 0x%02x", data[0])
	}
}

func TestEncodeValue_Uint16(t *testing.T) {
	data, err := NewAMF0Encoder().Encode(uint16(42))
	if err != nil {
		t.Fatal(err)
	}
	if data[0] != numberMarker {
		t.Errorf("expected numberMarker, got 0x%02x", data[0])
	}
}

func TestEncodeAMF0_Boolean(t *testing.T) {
	enc := NewAMF0Encoder()

	data, err := enc.Encode(true)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte{0x01, 0x01}) {
		t.Errorf("expected [0x01 0x01] for true, got %v", data)
	}

	data, err = enc.Encode(false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte{0x01, 0x00}) {
		t.Errorf("expected [0x01 0x00] for false, got %v", data)
	}
}

func TestEncodeAMF0_String(t *testing.T) {
	data, err := NewAMF0Encoder().Encode("hello")
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte{0x02, 0x00, 0x05, 'h', 'e', 'l', 'l', 'o'}
	if !bytes.Equal(data, expected) {
		t.Errorf("expected %v, got %v", expected, data)
	}
}

func TestEncodeAMF0_String_Empty(t *testing.T) {
	data, err := NewAMF0Encoder().Encode("")
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
	data, err := NewAMF0Encoder().Encode(longStr)
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
	data, err := NewAMF0Encoder().Encode(obj)
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

func TestEncodeAMF0_ECMAArray_UnsupportedValue(t *testing.T) {
	type unsupportedType struct{}
	_, err := NewAMF0Encoder().Encode(ECMAArray{"key": unsupportedType{}})
	if err == nil {
		t.Fatal("expected error for unsupported value in ECMAArray")
	}
}

func TestEncodeAMF0_ECMAArray(t *testing.T) {
	arr := ECMAArray{"key": "val"}
	data, err := NewAMF0Encoder().Encode(arr)
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte{
		0x08,
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x03, 'k', 'e', 'y',
		0x02, 0x00, 0x03, 'v', 'a', 'l',
		0x00, 0x00, 0x09,
	}
	if !bytes.Equal(data, expected) {
		t.Errorf("expected %v, got %v", expected, data)
	}
}

func TestEncodeAMF0_ECMAArray_RoundTrip(t *testing.T) {
	original := ECMAArray{"width": 1920.0, "height": 1080.0}
	data, err := NewAMF0Encoder().Encode(original)
	if err != nil {
		t.Fatal(err)
	}
	if data[0] != ecmaArrayMarker {
		t.Errorf("expected ecmaArrayMarker 0x%02x, got 0x%02x", ecmaArrayMarker, data[0])
	}

	vals, err := NewAMF0Decoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	decoded, ok := vals[0].(ECMAArray)
	if !ok {
		t.Fatalf("expected ECMAArray, got %T", vals[0])
	}
	if decoded["width"] != 1920.0 || decoded["height"] != 1080.0 {
		t.Errorf("round-trip mismatch: %v", decoded)
	}
}

func TestEncodeAMF0_ECMAArray_MapDoesNotEncode(t *testing.T) {
	// map[string]any는 Object(0x03)로 인코딩되어야 함
	data, err := NewAMF0Encoder().Encode(map[string]any{"key": "val"})
	if err != nil {
		t.Fatal(err)
	}
	if data[0] != objectMarker {
		t.Errorf("expected objectMarker 0x%02x, got 0x%02x", objectMarker, data[0])
	}
}

func TestEncodeAMF0_Object_UnsupportedValue(t *testing.T) {
	type unsupportedType struct{}
	_, err := NewAMF0Encoder().Encode(map[string]any{"key": unsupportedType{}})
	if err == nil {
		t.Fatal("expected error for unsupported value in object")
	}
}

func TestEncodeAMF0_Object_Empty(t *testing.T) {
	data, err := NewAMF0Encoder().Encode(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte{0x03, 0x00, 0x00, 0x09}
	if !bytes.Equal(data, expected) {
		t.Errorf("expected %v, got %v", expected, data)
	}
}

func TestEncodeObjectProperty_EmptyKey(t *testing.T) {
	_, err := NewAMF0Encoder().Encode(map[string]any{"": "value"})
	if err == nil {
		t.Fatal("expected error for empty key")
	}
	if !strings.Contains(err.Error(), "object key must not be empty") {
		t.Errorf("expected 'object key must not be empty', got %v", err.Error())
	}
}

func TestEncodeECMAArray_EmptyKey(t *testing.T) {
	_, err := NewAMF0Encoder().Encode(ECMAArray{"": "value"})
	if err == nil {
		t.Fatal("expected error for empty key in ECMAArray")
	}
	if !strings.Contains(err.Error(), "object key must not be empty") {
		t.Errorf("expected 'object key must not be empty', got %v", err.Error())
	}
}

func TestEncodeObjectProperty_KeyTooLong(t *testing.T) {
	_, err := NewAMF0Encoder().Encode(map[string]any{strings.Repeat("a", 70000): "value"})
	if err == nil {
		t.Fatal("expected error for key too long")
	}
	if !strings.Contains(err.Error(), "object key too long") {
		t.Errorf("expected 'object key too long', got %v", err.Error())
	}
}

func TestEncodeObjectProperty_ValueError(t *testing.T) {
	type unsupportedType struct{}
	_, err := NewAMF0Encoder().Encode(map[string]any{"key": unsupportedType{}})
	if err == nil {
		t.Fatal("expected value encode error")
	}
	if !strings.Contains(err.Error(), "unsupported AMF0 type") {
		t.Errorf("expected 'unsupported AMF0 type', got %v", err.Error())
	}
}

func TestEncodeAMF0_Null(t *testing.T) {
	data, err := NewAMF0Encoder().Encode(nil)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, []byte{0x05}) {
		t.Errorf("expected [0x05], got %v", data)
	}
}

func TestEncodeAMF0_StrictArray(t *testing.T) {
	data, err := NewAMF0Encoder().Encode([]any{"a", "b"})
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
	data, err := NewAMF0Encoder().Encode([]any{})
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
	_, err := NewAMF0Encoder().Encode([]any{unsupportedType{}})
	if err == nil {
		t.Fatal("expected error for unsupported element type")
	}
}

func TestEncodeAMF0_Date(t *testing.T) {
	date := time.Date(2023, 3, 28, 19, 40, 0, 123*1e6, time.UTC)
	data, err := NewAMF0Encoder().Encode(date)
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte{
		0x0B,
		0x42, 0x78, 0x72, 0x9B, 0xC0, 0x2F, 0xB0, 0x00,
		0x00, 0x00,
	}
	if !bytes.Equal(data, expected) {
		t.Errorf("expected %v, got %v", expected, data)
	}
}

func TestEncodeAMF0_Date_RoundTrip(t *testing.T) {
	// time.Time round-trip은 AMF0가 밀리초 정밀도이므로 ms 단위까지만 보존됨
	original := time.Date(2023, 3, 28, 19, 40, 0, 123_000_000, time.UTC)
	data, err := NewAMF0Encoder().Encode(original)
	if err != nil {
		t.Fatal(err)
	}
	vals, err := NewAMF0Decoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := vals[0].(time.Time)
	if !ok {
		t.Fatalf("expected time.Time, got %T", vals[0])
	}
	if !got.Equal(original) {
		t.Errorf("expected %v, got %v", original, got)
	}
}

func TestEncodeAMF0_UnsupportedType(t *testing.T) {
	type customType struct{ field string }
	_, err := NewAMF0Encoder().Encode(customType{field: "test"})
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
	if !strings.Contains(err.Error(), "unsupported AMF0 type") {
		t.Errorf("expected 'unsupported AMF0 type', got %v", err.Error())
	}
}

func TestEncodeAMF0_RoundTrip(t *testing.T) {
	enc := NewAMF0Encoder()
	dec := NewAMF0Decoder()

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
			encoded, err := enc.Encode(original)
			if err != nil {
				t.Fatalf("encoding failed: %v", err)
			}

			decoded, err := dec.Decode(encoded)
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
			case []any:
				got, ok := decoded[0].([]any)
				if !ok {
					t.Fatalf("expected []any, got %T", decoded[0])
				}
				if !reflect.DeepEqual(got, orig) {
					t.Errorf("expected %v, got %v", orig, got)
				}
			case map[string]any:
				got, ok := decoded[0].(map[string]any)
				if !ok {
					t.Fatalf("expected map[string]any, got %T", decoded[0])
				}
				if !reflect.DeepEqual(got, orig) {
					t.Errorf("expected %v, got %v", orig, got)
				}
			}
		})
	}
}

func TestEncodeAMF0_NaN_RoundTrip(t *testing.T) {
	data, err := NewAMF0Encoder().Encode(math.NaN())
	if err != nil {
		t.Fatal(err)
	}
	vals, err := NewAMF0Decoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if !math.IsNaN(vals[0].(float64)) {
		t.Errorf("expected NaN, got %v", vals[0])
	}
}

func TestEncodeAMF0_Inf_RoundTrip(t *testing.T) {
	data, err := NewAMF0Encoder().Encode(math.Inf(1))
	if err != nil {
		t.Fatal(err)
	}
	vals, err := NewAMF0Decoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if !math.IsInf(vals[0].(float64), 1) {
		t.Errorf("expected +Inf, got %v", vals[0])
	}
}

func TestEncodeAMF0_NestedObject_RoundTrip(t *testing.T) {
	original := map[string]any{
		"meta": map[string]any{"width": 1920.0, "height": 1080.0},
		"tags": []any{"live", "hd"},
	}
	data, err := NewAMF0Encoder().Encode(original)
	if err != nil {
		t.Fatal(err)
	}
	vals, err := NewAMF0Decoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(vals[0], original) {
		t.Errorf("expected %v, got %v", original, vals[0])
	}
}

func BenchmarkEncodeAMF0_Number(b *testing.B) {
	enc := NewAMF0Encoder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enc.Encode(3.14)
	}
}

func BenchmarkEncodeAMF0_String(b *testing.B) {
	enc := NewAMF0Encoder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enc.Encode("hello world")
	}
}

func TestEncodeAMF0_ECMAArray_Empty(t *testing.T) {
	data, err := NewAMF0Encoder().Encode(ECMAArray{})
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte{0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x09}
	if !bytes.Equal(data, expected) {
		t.Errorf("expected %v, got %v", expected, data)
	}
}

func BenchmarkEncodeAMF0_Object(b *testing.B) {
	enc := NewAMF0Encoder()
	obj := map[string]any{
		"name":  "test",
		"value": 123.45,
		"flag":  true,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enc.Encode(obj)
	}
}
