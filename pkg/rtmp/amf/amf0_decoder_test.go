package amf

import (
	"testing"
	"time"
)

func TestAMF0Decoder_Decode(t *testing.T) {
	dec := NewAMF0Decoder()
	data := []byte{0x00, 0x40, 0x09, 0x1e, 0xb8, 0x51, 0xeb, 0x85, 0x1f}

	values, err := dec.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(values))
	}
	if values[0].(float64) != 3.14 {
		t.Errorf("expected 3.14, got %v", values[0])
	}
}

func TestAMF0Decoder_Reuse(t *testing.T) {
	dec := NewAMF0Decoder()

	first, err := dec.Decode([]byte{0x00, 0x40, 0x09, 0x1e, 0xb8, 0x51, 0xeb, 0x85, 0x1f})
	if err != nil {
		t.Fatal(err)
	}
	second, err := dec.Decode([]byte{0x02, 0x00, 0x05, 'h', 'e', 'l', 'l', 'o'})
	if err != nil {
		t.Fatal(err)
	}
	if first[0] == second[0] {
		t.Error("reuse should produce different results")
	}
}

func TestAMF0Decoder_Error(t *testing.T) {
	dec := NewAMF0Decoder()
	_, err := dec.Decode([]byte{0x00, 0x40, 0x09}) // truncated number
	if err == nil {
		t.Fatal("expected error for malformed data")
	}
}

func TestDecodeAMF0Sequence(t *testing.T) {
	data := []byte{
		0x00, 0x40, 0x09, 0x1e, 0xb8, 0x51, 0xeb, 0x85, 0x1f,
		0x01, 0x01,
		0x02, 0x00, 0x05, 'h', 'e', 'l', 'l', 'o',
		0x03, 0x00, 0x03, 'f', 'o', 'o', 0x02, 0x00, 0x03, 'b', 'a', 'r', 0x00, 0x00, 0x09,
	}
	values, err := NewAMF0Decoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := values[0].(float64); !ok {
		t.Fatalf("expected float64, got %T", values[0])
	}
	if _, ok := values[1].(bool); !ok {
		t.Fatalf("expected bool, got %T", values[1])
	}
	if _, ok := values[2].(string); !ok {
		t.Fatalf("expected string, got %T", values[2])
	}
	if _, ok := values[3].(map[string]any); !ok {
		t.Fatalf("expected map, got %T", values[3])
	}
}

func TestDecodeAMF0Sequence_MalformedData(t *testing.T) {
	data := []byte{0x00, 0x40, 0x09, 0x1e, 0xb8, 0x51}
	_, err := NewAMF0Decoder().Decode(data)
	if err == nil {
		t.Fatal("expected error for malformed data")
	}
}

func TestDecodeAMF0_UnknownMarker(t *testing.T) {
	_, err := NewAMF0Decoder().Decode([]byte{0xff})
	if err == nil {
		t.Fatal("expected error for unknown marker")
	}
}

func TestDecodeAMF0_NotImplementedMarkers(t *testing.T) {
	testCases := []struct {
		name   string
		marker byte
	}{
		{"reference", referenceMarker},
		{"movieClip", movieClipMarker},
		{"unsupported", unsupportedMarker},
		{"recordSet", recordSetMarker},
		{"xmlDocument", xmlDocumentMarker},
		{"typedObject", typedObjectMarker},
		{"avmPlus", avmPlusMarker},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewAMF0Decoder().Decode([]byte{tc.marker})
			if err == nil {
				t.Fatalf("expected error for marker 0x%02x", tc.marker)
			}
		})
	}
}

func TestDecodeAMF0_Number(t *testing.T) {
	data := []byte{0x00, 0x40, 0x09, 0x1e, 0xb8, 0x51, 0xeb, 0x85, 0x1f}
	vals, err := NewAMF0Decoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if vals[0].(float64) != 3.14 {
		t.Errorf("expected 3.14, got %v", vals[0])
	}
}

func TestDecodeAMF0_Number_MalformedData(t *testing.T) {
	data := []byte{0x00, 0x40, 0x09, 0x1e, 0xb8, 0x51, 0xeb, 0x85}
	_, err := NewAMF0Decoder().Decode(data)
	if err == nil {
		t.Fatal("expected error for malformed number data")
	}
}

func TestDecodeAMF0_Boolean(t *testing.T) {
	vals, err := NewAMF0Decoder().Decode([]byte{0x01, 0x01})
	if err != nil {
		t.Fatal(err)
	}
	if vals[0].(bool) != true {
		t.Errorf("expected true, got %v", vals[0])
	}
}

func TestDecodeAMF0_Boolean_MalformedData(t *testing.T) {
	_, err := NewAMF0Decoder().Decode([]byte{0x01})
	if err == nil {
		t.Fatal("expected error for malformed boolean data")
	}
}

func TestDecodeAMF0_String(t *testing.T) {
	vals, err := NewAMF0Decoder().Decode([]byte{0x02, 0x00, 0x05, 'h', 'e', 'l', 'l', 'o'})
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := vals[0].(string); !ok || s != "hello" {
		t.Errorf("expected 'hello', got %v", vals[0])
	}
}

func TestDecodeAMF0_String_MalformedShortLength(t *testing.T) {
	_, err := NewAMF0Decoder().Decode([]byte{0x02, 0x00})
	if err == nil {
		t.Fatal("expected error for incomplete string length")
	}
}

func TestDecodeAMF0_String_MalformedShortData(t *testing.T) {
	_, err := NewAMF0Decoder().Decode([]byte{0x02, 0x00, 0x05, 'h', 'e', 'l'})
	if err == nil {
		t.Fatal("expected error for incomplete string data")
	}
}

func TestDecodeAMF0_Object(t *testing.T) {
	data := []byte{
		0x03,
		0x00, 0x03, 'f', 'o', 'o',
		0x02, 0x00, 0x03, 'b', 'a', 'r',
		0x00, 0x00, 0x09,
	}
	vals, err := NewAMF0Decoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	obj, ok := vals[0].(map[string]any)
	if !ok || obj["foo"] != "bar" {
		t.Errorf("expected foo=bar, got %v", vals[0])
	}
}

func TestDecodeAMF0_Object_MalformedShortKey(t *testing.T) {
	_, err := NewAMF0Decoder().Decode([]byte{0x03, 0x00, 0x03, 'f', 'o'})
	if err == nil {
		t.Fatal("expected error for incomplete object key")
	}
}

func TestDecodeAMF0_Object_MalformedMissingValueMarker(t *testing.T) {
	_, err := NewAMF0Decoder().Decode([]byte{0x03, 0x00, 0x03, 'f', 'o', 'o'})
	if err == nil {
		t.Fatal("expected error for missing value marker")
	}
}

func TestDecodeAMF0_Object_MalformedShortValue(t *testing.T) {
	data := []byte{0x03, 0x00, 0x03, 'f', 'o', 'o', 0x02, 0x00}
	_, err := NewAMF0Decoder().Decode(data)
	if err == nil {
		t.Fatal("expected error for incomplete object value")
	}
}

func TestDecodeAMF0_Object_MissingEndMarker(t *testing.T) {
	data := []byte{
		0x03,
		0x00, 0x03, 'f', 'o', 'o',
		0x02, 0x00, 0x03, 'b', 'a', 'r',
		0x00, 0x00,
	}
	_, err := NewAMF0Decoder().Decode(data)
	if err == nil {
		t.Fatal("expected error for missing object end marker")
	}
}

func TestDecodeAMF0_Object_InvalidEndMarker(t *testing.T) {
	data := []byte{
		0x03,
		0x00, 0x03, 'f', 'o', 'o',
		0x02, 0x00, 0x03, 'b', 'a', 'r',
		0x00, 0x00, 0x00,
	}
	_, err := NewAMF0Decoder().Decode(data)
	if err == nil {
		t.Fatal("expected error for invalid object end marker")
	}
}

func TestDecodeAMF0_Null(t *testing.T) {
	vals, err := NewAMF0Decoder().Decode([]byte{0x05})
	if err != nil {
		t.Fatal(err)
	}
	if vals[0] != nil {
		t.Errorf("expected nil, got %v", vals[0])
	}
}

func TestDecodeAMF0_Undefined(t *testing.T) {
	vals, err := NewAMF0Decoder().Decode([]byte{0x06})
	if err != nil {
		t.Fatal(err)
	}
	if vals[0] != nil {
		t.Errorf("expected nil, got %v", vals[0])
	}
}

func TestDecodeAMF0_ECMAArray(t *testing.T) {
	data := []byte{
		0x08,
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x03, 'k', 'e', 'y',
		0x02, 0x00, 0x03, 'v', 'a', 'l',
		0x00, 0x00, 0x09,
	}
	vals, err := NewAMF0Decoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	m, ok := vals[0].(ECMAArray)
	if !ok || m["key"] != "val" {
		t.Errorf("expected ECMAArray with key=val, got %T %v", vals[0], vals[0])
	}
}

func TestDecodeAMF0_ECMAArray_MalformedLength(t *testing.T) {
	_, err := NewAMF0Decoder().Decode([]byte{0x08, 0x00, 0x00})
	if err == nil {
		t.Fatal("expected error for malformed ECMA array length")
	}
}

func TestDecodeAMF0_ECMAArray_MalformedBody(t *testing.T) {
	_, err := NewAMF0Decoder().Decode([]byte{0x08, 0x00, 0x00, 0x00, 0x01, 0x00, 0x03, 'k'})
	if err == nil {
		t.Fatal("expected error for malformed ECMA array body")
	}
}

func TestDecodeAMF0_StrictArray(t *testing.T) {
	data := []byte{
		0x0A,
		0x00, 0x00, 0x00, 0x02,
		0x02, 0x00, 0x01, 'a',
		0x02, 0x00, 0x01, 'b',
	}
	vals, err := NewAMF0Decoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	arr, ok := vals[0].([]any)
	if !ok || len(arr) != 2 || arr[0] != "a" || arr[1] != "b" {
		t.Errorf("expected [a b], got %v", vals[0])
	}
}

func TestDecodeAMF0_StrictArray_MalformedLength(t *testing.T) {
	_, err := NewAMF0Decoder().Decode([]byte{0x0A, 0x00, 0x00, 0x00})
	if err == nil {
		t.Fatal("expected error for malformed strict array length")
	}
}

func TestDecodeAMF0_StrictArray_MalformedElement(t *testing.T) {
	data := []byte{0x0A, 0x00, 0x00, 0x00, 0x02,
		0x02, 0x00, 0x01, 'a',
		0x02, 0x00, 0x01,
	}
	_, err := NewAMF0Decoder().Decode(data)
	if err == nil {
		t.Fatal("expected error for malformed strict array element")
	}
}

func TestDecodeAMF0_Date(t *testing.T) {
	expected := time.Date(2023, 3, 28, 19, 40, 0, 123*1e6, time.UTC)

	data := []byte{
		0x0B,
		0x42, 0x78, 0x72, 0x9B,
		0xC0, 0x2F, 0xB0, 0x00,
		0x00, 0x00,
	}

	vals, err := NewAMF0Decoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}

	got, ok := vals[0].(time.Time)
	if !ok {
		t.Fatalf("expected time.Time, got %T", vals[0])
	}
	if !got.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func TestDecodeAMF0_Date_MalformedShortData(t *testing.T) {
	_, err := NewAMF0Decoder().Decode([]byte{0x0B, 0x00, 0x01})
	if err == nil {
		t.Error("expected error for malformed date data")
	}
}

func TestDecodeAMF0_Date_MalformedMissingOffset(t *testing.T) {
	data := make([]byte, 9)
	data[0] = 0x0B
	_, err := NewAMF0Decoder().Decode(data)
	if err == nil {
		t.Error("expected error for missing date offset")
	}
}

func TestDecodeAMF0_LongString(t *testing.T) {
	vals, err := NewAMF0Decoder().Decode([]byte{0x0c, 0x00, 0x00, 0x00, 0x05, 'h', 'e', 'l', 'l', 'o'})
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := vals[0].(string); !ok || s != "hello" {
		t.Errorf("expected 'hello', got %v", vals[0])
	}
}

func TestDecodeAMF0_LongString_MalformedShortLength(t *testing.T) {
	_, err := NewAMF0Decoder().Decode([]byte{0x0c, 0x00, 0x00, 0x00})
	if err == nil {
		t.Fatal("expected error for incomplete long string length")
	}
}

func TestDecodeAMF0_LongString_MalformedShortData(t *testing.T) {
	_, err := NewAMF0Decoder().Decode([]byte{0x0c, 0x00, 0x00, 0x00, 0x05, 'h', 'e', 'l'})
	if err == nil {
		t.Fatal("expected error for incomplete long string data")
	}
}

func BenchmarkDecodeAMF0_Number(b *testing.B) {
	dec := NewAMF0Decoder()
	data := []byte{0x00, 0x40, 0x09, 0x1e, 0xb8, 0x51, 0xeb, 0x85, 0x1f}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dec.Decode(data)
	}
}

func BenchmarkDecodeAMF0_Object(b *testing.B) {
	dec := NewAMF0Decoder()
	data := []byte{
		0x03,
		0x00, 0x04, 'n', 'a', 'm', 'e',
		0x02, 0x00, 0x04, 't', 'e', 's', 't',
		0x00, 0x00, 0x09,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dec.Decode(data)
	}
}
