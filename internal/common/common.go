package common

import (
    "encoding/binary"
    "math"
    "reflect"
    "unsafe"
)

// IsFixedKind reports whether k is a fixed-size primitive kind.
func IsFixedKind(k reflect.Kind) bool {
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

// FixedSize returns the byte width for fixed-size primitive kinds.
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

// WriteVarUintTo appends varint-encoded x to dst using a small stack scratch.
func WriteVarUintTo(dst []byte, x uint64) []byte {
    var scratch [10]byte
    i := 0
    for x >= 0x80 {
        scratch[i] = byte(x) | 0x80
        x >>= 7
        i++
    }
    scratch[i] = byte(x)
    i++
    return append(dst, scratch[:i]...)
}

// WriteVarUint appends a varint to buf (allocating if needed).
func WriteVarUint(buf []byte, x uint64) []byte {
    for x >= 0x80 {
        buf = append(buf, byte(x)|0x80)
        x >>= 7
    }
    return append(buf, byte(x))
}

// ReadVarUint decodes a varint from b returning value and bytes consumed.
func ReadVarUint(b []byte) (uint64, int) {
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

// SetUnsafeFixed aliases b into dst as a slice of the given kind without copying.
func SetUnsafeFixed(dst reflect.Value, b []byte, k reflect.Kind, sliceLen int) {
    switch k {
    case reflect.Bool:
        val := unsafe.Slice((*bool)(unsafe.Pointer(&b[0])), sliceLen)
        dst.Set(reflect.ValueOf(val))
    case reflect.Int8:
        val := unsafe.Slice((*int8)(unsafe.Pointer(&b[0])), sliceLen)
        dst.Set(reflect.ValueOf(val))
    case reflect.Uint8:
        val := unsafe.Slice((*uint8)(unsafe.Pointer(&b[0])), sliceLen)
        dst.Set(reflect.ValueOf(val))
    case reflect.Int16:
        val := unsafe.Slice((*int16)(unsafe.Pointer(&b[0])), sliceLen)
        dst.Set(reflect.ValueOf(val))
    case reflect.Uint16:
        val := unsafe.Slice((*uint16)(unsafe.Pointer(&b[0])), sliceLen)
        dst.Set(reflect.ValueOf(val))
    case reflect.Int32:
        val := unsafe.Slice((*int32)(unsafe.Pointer(&b[0])), sliceLen)
        dst.Set(reflect.ValueOf(val))
    case reflect.Uint32:
        val := unsafe.Slice((*uint32)(unsafe.Pointer(&b[0])), sliceLen)
        dst.Set(reflect.ValueOf(val))
    case reflect.Int64:
        val := unsafe.Slice((*int64)(unsafe.Pointer(&b[0])), sliceLen)
        dst.Set(reflect.ValueOf(val))
    case reflect.Uint64:
        val := unsafe.Slice((*uint64)(unsafe.Pointer(&b[0])), sliceLen)
        dst.Set(reflect.ValueOf(val))
    case reflect.Float32:
        val := unsafe.Slice((*float32)(unsafe.Pointer(&b[0])), sliceLen)
        dst.Set(reflect.ValueOf(val))
    case reflect.Float64:
        val := unsafe.Slice((*float64)(unsafe.Pointer(&b[0])), sliceLen)
        dst.Set(reflect.ValueOf(val))
    }
}

// SetFixed decodes a fixed-width primitive from b and sets dst.
func SetFixed(dst reflect.Value, b []byte, k reflect.Kind) {
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
