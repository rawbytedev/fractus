package fractus

import (
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type MixedHStruct struct {
	Val      string
	Mod      int8
	Data     string
	Integers int16
	Float3   float32
	Float6   float64
}

func FuzzHEncodeDecode(f *testing.F) {
	f.Fuzz(fuzzHMixedTypes)
}
func fuzzHMixedTypes(t *testing.T, Val string, Mod int8, Data string, Integers int16, Float3 float32, Float6 float64) {
	val := MixedHStruct{Val: Val, Mod: Mod, Data: Data, Integers: Integers, Float3: Float3, Float6: Float6}
	res := &MixedHStruct{}
	f := NewHighPerfFractus(SafeOptions{})
	data, err := f.Encode(val)
	require.NoError(t, err)
	err = f.Decode(data, res)
	require.NoError(t, err)
	require.EqualExportedValues(t, val, *res)
}
func TestHEncodeSimpleTypes(t *testing.T) {
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
	f := NewHighPerfFractus(SafeOptions{})
	data, err := f.Encode(z)
	require.NoError(t, err)
	err = f.Decode(data, res)
	require.NoError(t, err)
	require.EqualExportedValues(t, z, *res)
}
func TestHConstant(t *testing.T) {
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
	f := NewHighPerfFractus(SafeOptions{})
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
func TestHConstantList(t *testing.T) {
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
	f := NewHighPerfFractus(SafeOptions{})
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
func TestHStructPointer(t *testing.T) {
	type StructPtr struct {
		Data string
	}
	val := &StructPtr{Data: "Hello"}
	res := &StructPtr{}
	f := NewHighPerfFractus(SafeOptions{})
	data, err := f.Encode(val)
	require.NoError(t, err)
	err = f.Decode(data, res)
	require.NoError(t, err)
	require.EqualExportedValues(t, val, res)
}
func TestHErrors(t *testing.T) {
	f := NewHighPerfFractus(SafeOptions{})
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
func TestHEncodeListOfTypes(t *testing.T) {

	type NewStruct struct {
		Val      []string
		Mod      []int8
		Integers []int16
		Float3   []float32
		Float6   []float64
	}
	f := NewHighPerfFractus(SafeOptions{})
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
func BenchmarkHZeroAllocs(b *testing.B) {
	type ZeroAllocs struct {
		Int int8
	}
	z := ZeroAllocs{Int: int8(1)}
	f := NewHighPerfFractus(SafeOptions{UnsafePrimitives: false})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = f.Encode(z)
	}
}
func BenchmarkHEncoding(b *testing.B) {
	type NewStruct struct {
		Val      []string
		Mod      []int8
		Integers []int16
		Float3   []float32
		Float6   []float64
	}
	Val := []string{"azerty", "hello", "world", "random"}
	z := NewStruct{Val: Val,
		Mod: []int8{12, 10, 13, 1}, Integers: []int16{100, 250, 300},
		Float3: []float32{12.13, 16.23, 75.1}, Float6: []float64{100.5, 165.63, 153.5}}
	f := NewHighPerfFractus(SafeOptions{})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = f.Encode(z)
	}

}
func BenchmarkHUnsafeEncoding(b *testing.B) {
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
	f := NewHighPerfFractus(SafeOptions{UnsafePrimitives: false, UnsafeStrings: true})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = f.Encode(z)
	}

}
func BenchmarkHDecoding(b *testing.B) {
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
	f := NewHighPerfFractus(SafeOptions{UnsafePrimitives: false})
	s := NewHighPerfFractus(SafeOptions{UnsafePrimitives: false})
	b.ReportAllocs()
	res, _ := f.Encode(z)
	for i := 0; i < b.N; i++ {
		s.Decode(res, y)
	}
	require.EqualValues(b, z, *y)
}

// Test Encoding
func TestCompare(t *testing.T) {
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
	f := NewHighPerfFractus(SafeOptions{UnsafePrimitives: true, UnsafeStrings: true})
	res, _ := f.Encode(z)
	r := NewHighPerfFractus(SafeOptions{})
	rres, _ := r.Encode(z)
	assert.Equal(t, res, rres)
}
func TestCompares(t *testing.T) {
	type NewStruct struct {
		Integers []uint16
	}

	z := NewStruct{Integers: []uint16{100, 250, 300}}
	y := &NewStruct{}
	f := NewHighPerfFractus(SafeOptions{UnsafePrimitives: true, UnsafeStrings: true})
	res, _ := f.Encode(z)
	r := NewHighPerfFractus(SafeOptions{})
	rres, _ := r.Encode(z)
	assert.Equal(t, res, rres)
	err := f.Decode(res, y)
	assert.NoError(t, err)
	assert.EqualExportedValues(t, z, *y)
}
func BenchmarkHUnsafeDecoding(b *testing.B) {
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
	f := NewHighPerfFractus(SafeOptions{UnsafePrimitives: true, UnsafeStrings: true})
	b.ReportAllocs()
	res, _ := f.Encode(z)
	for i := 0; i < b.N; i++ {
		f.Decode(res, y)
	}
	require.EqualValues(b, z, *y)
}
func BenchmarkHFractus(b *testing.B) {
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
	f := NewHighPerfFractus(SafeOptions{})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		res, _ := f.Encode(z) // 3allocs / res = []byte // 1 allocs
		f.Decode(res, y)      // 1allocs
	}
	require.EqualValues(b, z, *y)
}
func BenchmarkHDoubleFractus(b *testing.B) {
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
	f := NewHighPerfFractus(SafeOptions{})
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
func BenchmarkHYaml(b *testing.B) {
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
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {

		_, _ = yaml.Marshal(z)
	}
}
