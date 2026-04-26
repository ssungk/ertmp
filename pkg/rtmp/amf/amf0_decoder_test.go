package amf

import (
	"math"
	"testing"
	"time"
)

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
	if first[0].(float64) != 3.14 {
		t.Errorf("first decode wrong: %v", first[0])
	}
	if second[0].(string) != "hello" {
		t.Errorf("second decode wrong: %v", second[0])
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

	if values[0].(float64) != 3.14 {
		t.Errorf("expected 3.14, got %v", values[0])
	}
	if values[1].(bool) != true {
		t.Errorf("expected true, got %v", values[1])
	}
	if values[2].(string) != "hello" {
		t.Errorf("expected \"hello\", got %v", values[2])
	}
	m, ok := values[3].(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", values[3])
	}
	if m["foo"] != "bar" {
		t.Errorf("expected foo=bar, got %v", m["foo"])
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

func TestDecodeAMF0_UnsupportedMarkers(t *testing.T) {
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
	if !vals[0].(bool) {
		t.Errorf("expected true, got %v", vals[0])
	}
}

func TestDecodeAMF0_Boolean_False(t *testing.T) {
	vals, err := NewAMF0Decoder().Decode([]byte{0x01, 0x00})
	if err != nil {
		t.Fatal(err)
	}
	if vals[0].(bool) {
		t.Errorf("expected false, got true")
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

func TestDecodeAMF0_Object_Empty(t *testing.T) {
	data := []byte{0x03, 0x00, 0x00, 0x09}
	vals, err := NewAMF0Decoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	obj, ok := vals[0].(map[string]any)
	if !ok || len(obj) != 0 {
		t.Errorf("expected empty map, got %v", vals[0])
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

func TestDecodeAMF0_ECMAArray_TypeAssertion(t *testing.T) {
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
	if _, ok := vals[0].(ECMAArray); !ok {
		t.Errorf("expected ECMAArray assertion to succeed, got %T", vals[0])
	}
	if _, ok := vals[0].(map[string]any); ok {
		t.Errorf("ECMAArray should not be assertable as map[string]any")
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

func TestDecodeAMF0_StrictArray_Empty(t *testing.T) {
	data := []byte{0x0A, 0x00, 0x00, 0x00, 0x00}
	vals, err := NewAMF0Decoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	arr, ok := vals[0].([]any)
	if !ok || len(arr) != 0 {
		t.Errorf("expected empty []any, got %v", vals[0])
	}
}

func TestDecodeAMF0_StrictArray_MalformedLength(t *testing.T) {
	_, err := NewAMF0Decoder().Decode([]byte{0x0A, 0x00, 0x00, 0x00})
	if err == nil {
		t.Fatal("expected error for malformed strict array length")
	}
}

func TestDecodeAMF0_StrictArray_OversizedCount(t *testing.T) {
	_, err := NewAMF0Decoder().Decode([]byte{0x0A, 0xFF, 0xFF, 0xFF, 0xFF})
	if err == nil {
		t.Fatal("expected error for count exceeding remaining data")
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

func TestDecodeAMF0_Date_NaN(t *testing.T) {
	// NaN millis must be rejected; int64(NaN) silently produces 0 without this check
	data := []byte{
		0x0B,
		0x7F, 0xF8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // quiet NaN
		0x00, 0x00,
	}
	_, err := NewAMF0Decoder().Decode(data)
	if err == nil {
		t.Error("expected error for NaN millis")
	}
}

func TestDecodeAMF0_Date_Inf(t *testing.T) {
	// Inf millis must be rejected; int64(+Inf) silently produces MinInt64 without this check
	data := []byte{
		0x0B,
		0x7F, 0xF0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // +Inf
		0x00, 0x00,
	}
	_, err := NewAMF0Decoder().Decode(data)
	if err == nil {
		t.Error("expected error for Inf millis")
	}
}

func TestDecodeAMF0_Date_NegInf(t *testing.T) {
	data := []byte{
		0x0B,
		0xFF, 0xF0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // -Inf
		0x00, 0x00,
	}
	_, err := NewAMF0Decoder().Decode(data)
	if err == nil {
		t.Error("expected error for -Inf millis")
	}
}

func TestDecodeAMF0_Date_OverflowInt64(t *testing.T) {
	// float64(math.MaxInt64) rounds up to 2^63; millis = 2^63 passes "> MaxInt64" but overflows int64
	// use >= float64(math.MaxInt64) to catch this boundary
	boundary := math.Float64bits(float64(math.MaxInt64)) // = 2^63
	data := []byte{
		0x0B,
		byte(boundary >> 56), byte(boundary >> 48), byte(boundary >> 40), byte(boundary >> 32),
		byte(boundary >> 24), byte(boundary >> 16), byte(boundary >> 8), byte(boundary),
		0x00, 0x00,
	}
	_, err := NewAMF0Decoder().Decode(data)
	if err == nil {
		t.Error("expected error for millis at int64 boundary (2^63)")
	}

	// also verify a clearly large value is rejected
	overflow := math.Float64bits(1e30)
	data = []byte{
		0x0B,
		byte(overflow >> 56), byte(overflow >> 48), byte(overflow >> 40), byte(overflow >> 32),
		byte(overflow >> 24), byte(overflow >> 16), byte(overflow >> 8), byte(overflow),
		0x00, 0x00,
	}
	_, err = NewAMF0Decoder().Decode(data)
	if err == nil {
		t.Error("expected error for millis out of int64 range")
	}

	// negative overflow: -1e30 is finite but below MinInt64
	negOverflow := math.Float64bits(-1e30)
	data = []byte{
		0x0B,
		byte(negOverflow >> 56), byte(negOverflow >> 48), byte(negOverflow >> 40), byte(negOverflow >> 32),
		byte(negOverflow >> 24), byte(negOverflow >> 16), byte(negOverflow >> 8), byte(negOverflow),
		0x00, 0x00,
	}
	_, err = NewAMF0Decoder().Decode(data)
	if err == nil {
		t.Error("expected error for negative millis out of int64 range")
	}
}

func TestDecodeAMF0_Date_NonZeroTimezone(t *testing.T) {
	// AMF0 spec §2.13: timezone offset is deprecated and must be ignored
	millis := 1_000.0
	data := []byte{
		0x0B,
		0x40, 0x8F, 0x40, 0x00, 0x00, 0x00, 0x00, 0x00, // 1000.0 ms
		0x02, 0x1C, // KST +540 minutes (non-zero, must be ignored)
	}
	vals, err := NewAMF0Decoder().Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := vals[0].(time.Time)
	if !ok {
		t.Fatalf("expected time.Time, got %T", vals[0])
	}
	if got.Location() != time.UTC {
		t.Errorf("expected UTC location, got %v", got.Location())
	}
	if got.UnixMilli() != int64(millis) {
		t.Errorf("expected %v ms, got %v ms", int64(millis), got.UnixMilli())
	}
}

func TestDecodeAMF0_EmptyInput(t *testing.T) {
	vals, err := NewAMF0Decoder().Decode([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 0 {
		t.Errorf("expected empty slice, got %v", vals)
	}
}

func TestDecodeAMF0_Boolean_NonStandardTruthy(t *testing.T) {
	// AMF0 spec defines only 0x00/0x01, but any non-zero byte is treated as true
	vals, err := NewAMF0Decoder().Decode([]byte{0x01, 0xFF})
	if err != nil {
		t.Fatal(err)
	}
	if !vals[0].(bool) {
		t.Errorf("expected non-zero byte to be true")
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

func TestDecodeAMF0_LongString_OversizedLength(t *testing.T) {
	_, err := NewAMF0Decoder().Decode([]byte{0x0c, 0xFF, 0xFF, 0xFF, 0xFF})
	if err == nil {
		t.Fatal("expected error for length exceeding max long string len")
	}
}

func TestDecodeAMF0_LongString_CustomMaxLen(t *testing.T) {
	d := NewAMF0Decoder()
	d.SetLongStrLimit(4)
	if d.LongStrLimit() != 4 {
		t.Fatalf("expected LongStrLimit=4, got %d", d.LongStrLimit())
	}
	// length=5, exceeds custom max of 4
	_, err := d.Decode([]byte{0x0c, 0x00, 0x00, 0x00, 0x05, 'h', 'e', 'l', 'l', 'o'})
	if err == nil {
		t.Fatal("expected error for length exceeding custom max")
	}
}

func TestDecodeAMF0_LongString_ZeroMaxLen(t *testing.T) {
	d := NewAMF0Decoder()
	d.SetLongStrLimit(0) // 0 = no limit
	_, err := d.Decode([]byte{0x0c, 0x00, 0x00, 0x00, 0x05, 'h', 'e', 'l', 'l', 'o'})
	if err != nil {
		t.Fatalf("expected success with no limit, got %v", err)
	}
}

func TestDecodeAMF0_LongString_MalformedShortData(t *testing.T) {
	_, err := NewAMF0Decoder().Decode([]byte{0x0c, 0x00, 0x00, 0x00, 0x05, 'h', 'e', 'l'})
	if err == nil {
		t.Fatal("expected error for incomplete long string data")
	}
}

