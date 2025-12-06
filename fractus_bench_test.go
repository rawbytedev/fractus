package fractus

import (
	"testing"
)

func BenchmarkFractusZeroAllocs(b *testing.B) {
	type ZeroAllocs struct{ Int int8 }
	z := ZeroAllocs{Int: int8(1)}
	f := NewFractus(SafeOptions{UnsafePrimitives: false})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = f.Encode(z)
	}
}

func BenchmarkFractusEncoding(b *testing.B) {
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
	f := NewFractus(SafeOptions{})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = f.Encode(z)
	}
}

func BenchmarkFractusUnsafeEncoding(b *testing.B) {
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
	f := NewFractus(SafeOptions{UnsafePrimitives: false, UnsafeStrings: true})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = f.Encode(z)
	}
}

func BenchmarkFractusDecoding(b *testing.B) {
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
	f := NewFractus(SafeOptions{UnsafePrimitives: false})
	s := NewFractus(SafeOptions{UnsafePrimitives: false})
	b.ReportAllocs()
	res, _ := f.Encode(z)
	for i := 0; i < b.N; i++ {
		s.Decode(res, y)
	}
}

func BenchmarkFractusUnsafeDecoding(b *testing.B) {
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
}

func BenchmarkFractusBasic(b *testing.B) {
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
		res, _ := f.Encode(z)
		f.Decode(res, y)
	}
}

func BenchmarkFractusDouble(b *testing.B) {
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
	_ = err
	f.Decode(res, y)
}
