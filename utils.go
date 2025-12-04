package fractus

import (
	"encoding/binary"
	"math"
	"reflect"
	"unsafe"
)

// classify field kinds
func isFixedKind(k reflect.Kind) bool {
	switch k {
	case reflect.Bool,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func FixedSize(k reflect.Kind) int {
	switch k {
	case reflect.Bool, reflect.Int8, reflect.Uint8:
		return 1
	case reflect.Int16, reflect.Uint16:
		return 2
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		return 4
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		return 8
	default:
		return -1
	}
}
func writeFixed(dst []byte, v reflect.Value) []byte {
	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			return append(dst, 1)
		}
		return append(dst, 0)
	case reflect.Int8:
		return append(dst, byte(v.Int()))
	case reflect.Uint8:
		return append(dst, byte(v.Uint()))
	case reflect.Int16:
		tmp := make([]byte, 2)
		binary.LittleEndian.PutUint16(tmp, uint16(v.Int()))
		return append(dst, tmp...)
	case reflect.Uint16:
		tmp := make([]byte, 2)
		binary.LittleEndian.PutUint16(tmp, uint16(v.Uint()))
		return append(dst, tmp...)
	case reflect.Int32:
		tmp := make([]byte, 4)
		binary.LittleEndian.PutUint32(tmp, uint32(v.Int()))
		return append(dst, tmp...)
	case reflect.Uint32:
		tmp := make([]byte, 4)
		binary.LittleEndian.PutUint32(tmp, uint32(v.Uint()))
		return append(dst, tmp...)
	case reflect.Int64:
		tmp := make([]byte, 8)
		binary.LittleEndian.PutUint64(tmp, uint64(v.Int()))
		return append(dst, tmp...)
	case reflect.Uint64:
		tmp := make([]byte, 8)
		binary.LittleEndian.PutUint64(tmp, v.Uint())
		return append(dst, tmp...)
	case reflect.Float32:
		tmp := make([]byte, 4)
		binary.LittleEndian.PutUint32(tmp, math.Float32bits(float32(v.Float())))
		return append(dst, tmp...)
	case reflect.Float64:
		tmp := make([]byte, 8)
		binary.LittleEndian.PutUint64(tmp, math.Float64bits(v.Float()))
		return append(dst, tmp...)
	default:
		panic(ErrUnsupported)
	}
}

func writeVarUint(buf []byte, x uint64) []byte {
	for x >= 0x80 {
		buf = append(buf, byte(x)|0x80)
		x >>= 7
	}
	return append(buf, byte(x))
}

func varintLen(x uint64) int {
	n := 1
	for x >= 0x80 {
		n++
		x >>= 7
	}
	return n
}

func setUnsafeFixed(dst reflect.Value, b []byte, k reflect.Kind, sliceLen int) {
	switch k {
	case reflect.Uint16:
		val := unsafe.Slice((*uint16)(unsafe.Pointer(&b[0])), sliceLen)
		dst.Set(reflect.ValueOf(val))
	}
}
func setFixed(dst reflect.Value, b []byte, k reflect.Kind) {
	switch k {
	case reflect.Bool:
		dst.SetBool(b[0] != 0)
	case reflect.Int8:
		dst.SetInt(int64(int8(b[0])))
	case reflect.Uint8:
		dst.SetUint(uint64(b[0]))
	case reflect.Int16:
		dst.SetInt(int64(int16(binary.LittleEndian.Uint16(b))))
	case reflect.Uint16:
		dst.SetUint(uint64(binary.LittleEndian.Uint16(b)))
	case reflect.Int32:
		dst.SetInt(int64(int32(binary.LittleEndian.Uint32(b))))
	case reflect.Uint32:
		dst.SetUint(uint64(binary.LittleEndian.Uint32(b)))
	case reflect.Int64:
		dst.SetInt(int64(binary.LittleEndian.Uint64(b)))
	case reflect.Uint64:
		dst.SetUint(binary.LittleEndian.Uint64(b))
	case reflect.Float32:
		dst.SetFloat(float64(math.Float32frombits(binary.LittleEndian.Uint32(b))))
	case reflect.Float64:
		dst.SetFloat(math.Float64frombits(binary.LittleEndian.Uint64(b)))
	}
}

func readVarUint(b []byte) (uint64, int) {
	var x uint64
	var s uint
	for i, c := range b {
		x |= uint64(c&0x7F) << s
		if c&0x80 == 0 {
			return x, i + 1
		}
		s += 7
	}
	return 0, 0
}

func bitPresent(p []byte, idx int) bool {
	return p[idx/8]&(1<<(uint(idx)%8)) != 0
}
