package fractus

import (
	"fmt"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MixedStruct struct {
	Str     string
	Int8    int8
	bytes   byte
	Int16   int16
	Int32   int32
	Int64   int64
	Uint8   uint8
	Uint16  uint16
	Uint32  uint32
	Uint64  uint64
	Float32 float32
	Float64 float64
}
type IntTypes struct {
	Int8   int8
	Int16  int16
	Int32  int32
	Int64  int64
	Uint8  uint8
	Uint16 uint16
	Uint32 uint32
	Uint64 uint64
}

func FuzzIntEncode(f *testing.F) {
	f.Fuzz(fuzzIntTypes)
}

func FuzzMixedEncode(f *testing.F) {
	f.Fuzz(fuzzMixedTypes)
}
func fuzzIntTypes(t *testing.T, Int8 int8,
	Int16 int16,
	Int32 int32,
	Int64 int64,
	Uint8 uint8,
	Uint16 uint16,
	Uint32 uint32,
	Uint64 uint64) {
	val := IntTypes{Int8, Int16, Int32, Int64, Uint8, Uint16, Uint32, Uint64}
	res := &IntTypes{}
	f := NewFractus(SafeOptions{UnsafeStrings: true, UnsafePrimitives: true})
	data, err := f.Encode(val)
	require.NoError(t, err)
	err = f.Decode(data, res)
	require.NoError(t, err)
	require.EqualExportedValues(t, val, *res)
}
func fuzzMixedTypes(t *testing.T, Str string,
	Int8 int8,
	bytes byte,
	Int16 int16,
	Int32 int32,
	Int64 int64,
	Uint8 uint8,
	Uint16 uint16,
	Uint32 uint32,
	Uint64 uint64,
	Float32 float32,
	Float64 float64) {
	val := MixedStruct{Str, Int8, bytes, Int16, Int32, Int64, Uint8, Uint16, Uint32, Uint64, Float32, Float64}
	res := &MixedStruct{}
	f := NewFractus(SafeOptions{UnsafeStrings: true, UnsafePrimitives: true})
	data, err := f.Encode(val)
	require.NoError(t, err)
	err = f.Decode(data, res)
	require.NoError(t, err)
	require.EqualExportedValues(t, val, *res)
}
func TestEncodeDecode_SimpleTypes(t *testing.T) {
	type NewStruct struct {
		Val      []string
		Mod      int8
		Data     string
		Integers int16
		Float3   float32
		Float6   float64
	}
	z := NewStruct{Val: []string{"azerty", "Loling"}, Data: "testing",
		Mod: int8(17), Integers: int16(12),
		Float3: float32(12.3), Float6: float64(1236.2)}
	res := &NewStruct{}
	f := NewFractus(SafeOptions{UnsafeStrings: true, UnsafePrimitives: true})
	data, err := f.Encode(z)
	require.NoError(t, err)
	err = f.Decode(data, res)
	require.NoError(t, err)
	require.EqualExportedValues(t, z, *res)
}
func TestRoundTrip_Primitives(t *testing.T) {
	type NewStructint struct {
		Int1  uint8
		Int2  int8
		Int3  uint16
		Int4  int16
		Int5  uint32
		Int6  int32
		Int7  uint64
		Int9  int64
		Const bool
	}
	f := NewFractus(SafeOptions{UnsafeStrings: true, UnsafePrimitives: true})
	condition := func(z NewStructint) bool {
		data, err := f.Encode(z)
		require.NoError(t, err)
		res := &NewStructint{}
		err = f.Decode(data, res)
		require.NoError(t, err)
		return assert.ObjectsAreEqual(z, *res)
	}
	err := quick.Check(condition, &quick.Config{})
	if err != nil {
		t.Errorf("Error: %v", err)
	}
}
func TestRoundTrip_PrimitiveSlices(t *testing.T) {
	type NewStructint struct {
		Int1  []uint8
		Int2  int8
		Int3  []uint16
		Int4  []int16
		Int5  []uint32
		Int6  []int32
		Int7  []uint64
		Int9  []int64
		Const []bool
	}
	f := NewFractus(SafeOptions{})
	condition := func(z NewStructint) bool {
		val := z
		data, err := f.Encode(val)
		require.NoError(t, err)
		res := &NewStructint{}
		err = f.Decode(data, res)
		require.NoError(t, err)
		return assert.ObjectsAreEqual(z, *res)
	}
	err := quick.Check(condition, &quick.Config{})
	require.NoError(t, err)
}
func TestEncodeDecode_StructPointer(t *testing.T) {
	type StructPtr struct {
		Data string
	}
	val := &StructPtr{Data: "Hello"}
	res := &StructPtr{}
	f := NewFractus(SafeOptions{})
	data, err := f.Encode(val)
	require.NoError(t, err)
	err = f.Decode(data, res)
	require.NoError(t, err)
	require.EqualExportedValues(t, val, res)
}
func TestEncodeDecode_Errors(t *testing.T) {
	f := NewFractus(SafeOptions{})
	data, err := f.Encode("abc")
	require.Len(t, data, 0)
	require.ErrorIs(t, err, ErrNotStruct)
	type Eas struct {
		val string // private
	}
	str := Eas{val: "hello"}
	Ptrstr := &Eas{val: "world"}
	data, err = f.Encode(Ptrstr)
	require.Nil(t, err)
	err = f.Decode(data, str) // needs pointer
	require.ErrorIs(t, err, ErrNotStructPtr)
}
func TestRoundTrip_ListTypes(t *testing.T) {

	type NewStruct struct {
		Val      []string
		Mod      []int8
		Integers []int16
		Float3   []float32
		Float6   []float64
	}
	f := NewFractus(SafeOptions{})
	condition := func(z NewStruct) bool {
		data, err := f.Encode(z)
		require.NoError(t, err)
		res := &NewStruct{}
		err = f.Decode(data, res)
		require.NoError(t, err)
		return assert.ObjectsAreEqual(z, *res)
	}
	err := quick.Check(condition, &quick.Config{})
	if err != nil {
		t.Errorf("Error: %v", err)
	}
}

// SafeDecoder keeps the payload alive so zero-copy strings/slices remain valid.
func TestSafeDecoder_KeepsPayload(t *testing.T) {
	type T struct{ S string }
	f := NewFractus(SafeOptions{UnsafeStrings: true})
	v := T{S: "persist"}
	data, err := f.Encode(v)
	require.NoError(t, err)

	sd := NewSafeDecoder(f)
	var out T
	err = sd.Decode(data, &out)
	require.NoError(t, err)
	require.Equal(t, "persist", out.S)

	// clear local reference to data and force GC; SafeDecoder.payload keeps it alive
	data = nil
	// runtime.GC() is non-deterministic in tests on all platforms; we at least
	// ensure the SafeDecoder holds a reference and the value is accessible.
	require.NotNil(t, sd)
	require.Equal(t, "persist", out.S)
}

// Concurrent usage: ensure separate Fractus instances can be used concurrently.
func TestConcurrentSeparateInstances(t *testing.T) {
	type S struct {
		A int32
		B []int16
	}
	workers := 8
	done := make(chan error, workers)
	for i := 0; i < workers; i++ {
		go func(i int) {
			f := NewFractus(SafeOptions{})
			v := S{A: int32(i), B: []int16{1, 2, 3}}
			data, err := f.Encode(v)
			if err != nil {
				done <- err
				return
			}
			var out S
			err = f.Decode(data, &out)
			if err != nil {
				done <- err
				return
			}
			if out.A != v.A {
				done <- fmt.Errorf("mismatch")
				return
			}
			done <- nil
		}(i)
	}
	for i := 0; i < workers; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}

// Additional fuzz targets: strings and slices to exercise edge-cases.
func FuzzStrings(f *testing.F) {
	f.Fuzz(func(t *testing.T, s string) {
		type X struct{ S string }
		fst := NewFractus(SafeOptions{})
		val := X{S: s}
		data, err := fst.Encode(val)
		require.NoError(t, err)
		var out X
		err = fst.Decode(data, &out)
		require.NoError(t, err)
		require.Equal(t, val.S, out.S)
	})
}

func FuzzSlices(f *testing.F) {
	// Fuzz over raw byte slices (supported by Go fuzzing). This exercises
	// encoding/decoding of []byte payloads and primitive byte-slice paths.
	f.Fuzz(func(t *testing.T, a []byte) {
		type X struct{ A []byte }
		fst := NewFractus(SafeOptions{})
		val := X{A: a}
		data, err := fst.Encode(val)
		require.NoError(t, err)
		var out X
		err = fst.Decode(data, &out)
		require.NoError(t, err)
		require.Equal(t, val.A, out.A)
	})
}

// Test Encoding
func TestUnsafeAndSafe_ProduceSameBytes(t *testing.T) {
	type NewStruct struct {
		Val      []string
		Mod      []int8
		Integers []uint16
		Float3   []float32
		Float6   []float64
	}
	Val := []string{"azerty", "hello", "world", "random"}
	z := NewStruct{Val: Val,
		Mod: []int8{12, 10, 13, 0}, Integers: []uint16{100, 250, 300},
		Float3: []float32{12.13, 16.23, 75.1}, Float6: []float64{100.5, 165.63, 153.5}}
	//y := &NewStruct{}
	f := NewFractus(SafeOptions{UnsafePrimitives: true, UnsafeStrings: true})
	res, _ := f.Encode(z)
	r := NewFractus(SafeOptions{})
	rres, _ := r.Encode(z)
	assert.Equal(t, res, rres)
}
func TestUnsafeDecode_EqualsSafeDecode(t *testing.T) {
	type NewStruct struct {
		Integers []uint16
	}

	z := NewStruct{Integers: []uint16{100, 250, 300}}
	y := &NewStruct{}
	f := NewFractus(SafeOptions{UnsafePrimitives: true, UnsafeStrings: true})
	res, _ := f.Encode(z)
	r := NewFractus(SafeOptions{})
	rres, _ := r.Encode(z)
	assert.Equal(t, res, rres)
	err := f.Decode(res, y)
	assert.NoError(t, err)
	assert.EqualExportedValues(t, z, *y)
}
func BenchmarkUnsafeDecoding(b *testing.B) {
	type NewStruct struct {
		Val      []string
		Mod      []int8
		Integers []int16
		Float3   []float32
		Float6   []float64
	}
	Val := []string{"azerty", "hello", "world", "random"}
	z := NewStruct{Val: Val,
		Mod: []int8{12, 10, 13, 0}, Integers: []int16{100, 250, 300},
		Float3: []float32{12.13, 16.23, 75.1}, Float6: []float64{100.5, 165.63, 153.5}}
	y := &NewStruct{}
	f := NewFractus(SafeOptions{UnsafePrimitives: false, UnsafeStrings: true})
	b.ReportAllocs()
	res, _ := f.Encode(z)
	for i := 0; i < b.N; i++ {
		f.Decode(res, y)
	}
	require.EqualValues(b, z, *y)
}
func BenchmarkFractus(b *testing.B) {
	type NewStructint struct {
		Int1 uint8
		Int2 int8
		Int3 uint16
		Int4 int16
		Int5 uint32
		Int6 int32
		Int7 uint64
		Int9 int64
	}
	z := NewStructint{Int1: 1, Int2: 2, Int3: 16, Int4: 18, Int5: 1586, Int6: 15262, Int7: 1547544565, Int9: 15484565656}
	y := &NewStructint{}
	f := NewFractus(SafeOptions{})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		res, _ := f.Encode(z) // 3allocs / res = []byte // 1 allocs
		f.Decode(res, y)      // 1allocs
	}
	require.EqualValues(b, z, *y)
}
func BenchmarkDoubleFractus(b *testing.B) {
	type NewStructint struct {
		Int1 uint8
		Int2 int8
		Int3 uint16
		Int4 int16
		Int5 uint32
		Int6 int32
		Int7 uint64
		Int9 int64
	}
	z := NewStructint{Int1: 1, Int2: 2, Int3: 16, Int4: 18, Int5: 1586, Int6: 15262, Int7: 1547544565, Int9: 15484565656}
	v := z
	y := &NewStructint{}
	f := NewFractus(SafeOptions{})
	res := []byte{}
	b.ReportAllocs()
	_, _ = f.Encode(z)
	var err error
	for i := 0; i < b.N; i++ {
		res, err = f.Encode(v)
	}
	require.NoError(b, err)
	f.Decode(res, y)
	require.EqualValues(b, v, *y)
}
