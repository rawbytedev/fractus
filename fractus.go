package fractus

import (
	"encoding/binary"
	"errors"
	"math"
	"reflect"
	"unsafe"
)

var (
	ErrNotStruct    = errors.New("expected struct")
	ErrNotStructPtr = errors.New("expected pointer to struct")
	ErrUnsupported  = errors.New("unsupported type")
)

type Options struct {
	UnsafeStrings bool // zero-copy strings via unsafe; caller must ensure buf lifetime
}

type Fractus struct {
	Opts    Options
	buf     []byte
	body    []byte
	plan    map[reflect.Type]FieldPlan
	scratch []byte
}
type FieldPlan struct {
	count int
	field []FieldInfo
}
type FieldInfo struct {
	idx   int
	kind  reflect.Kind
	isVar bool
}

// Header: varint N
// Presence: (N/8) bytes
// VarOffsets: varint per variable present field
// Body: fields in declaration order

func (f *Fractus) Encode(val any) ([]byte, error) {
	v := reflect.ValueOf(val)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, ErrNotStruct
	}

	t := v.Type()
	N := t.NumField()
	if f.plan == nil {
		f.plan = make(map[reflect.Type]FieldPlan)
	}
	pl, ok := f.plan[t]
	fields := make([]FieldInfo, 0, N)

	if ok {
		fields = pl.field
	} else {
		for i := 0; i < N; i++ {
			sf := t.Field(i)
			if sf.PkgPath != "" && !sf.Anonymous {
				continue // skip unexported
			}
			k := sf.Type.Kind()
			fields = append(fields, FieldInfo{
				idx:   i,
				kind:  k,
				isVar: !isFixedKind(k),
			})
		}
		f.plan[t] = FieldPlan{field: fields, count: len(fields)}
	}
	n := len(fields)
	f.Reset(n)
	// write N
	f.buf = writeVarUint(f.buf, uint64(n))

	varOffsets := make([]int, 0, n)
	for _, field := range fields {
		// call Field only once
		val := v.Field(field.idx)
		if field.isVar {
			// record offset to start
			varOffsets = append(varOffsets, len(f.body))
			switch field.kind {
			case reflect.String:
				if f.Opts.UnsafeStrings {
					s := unsafe.Slice((*byte)(unsafe.Pointer(val.UnsafeAddr())), val.Len())
					f.body = writeVarUint(f.body, uint64(len(s)))
					f.body = append(f.body, s...)
				} else {
					s := val.String()
					f.body = writeVarUint(f.body, uint64(len(s)))
					f.body = append(f.body, s...)
				}
			case reflect.Slice:
				if val.Type().Elem().Kind() == reflect.Uint8 {
					b := val.Bytes()
					f.body = writeVarUint(f.body, uint64(len(b)))
					f.body = append(f.body, b...)
				} else {
					// lists: encode count then elements
					l := val.Len()
					f.body = writeVarUint(f.body, uint64(l))
					for j := 0; j < l; j++ {
						elem := val.Index(j)
						k := elem.Kind()
						if isFixedKind(k) {
							f.writeFixed(&elem)
						} else if k == reflect.String {
							if f.Opts.UnsafeStrings {
								s := unsafe.Slice(unsafe.StringData(elem.String()), elem.Len())
								f.body = writeVarUint(f.body, uint64(len(s)))
								f.body = append(f.body, s...)
							} else {
								s := elem.String()
								f.body = writeVarUint(f.body, uint64(len(s)))
								f.body = append(f.body, s...)
							}
						} else if k == reflect.Slice && elem.Type().Elem().Kind() == reflect.Uint8 {
							b := elem.Bytes()
							f.body = writeVarUint(f.body, uint64(len(b)))
							f.body = append(f.body, b...)
						} else {
							return nil, ErrUnsupported
						}
					}
				}
			default:
				return nil, ErrUnsupported
			}
		} else {
			f.writeFixed(&val)
		}
	}

	// write varOffsets as varints
	for _, off := range varOffsets {
		f.buf = writeVarUint(f.buf, uint64(off))
	}
	// append body
	f.buf = append(f.buf, f.body...)
	return f.buf, nil
}

func (f *Fractus) writeFixed(v *reflect.Value) {
	// necessary but we lose zero allocs
	// will research another solution
	if f.scratch == nil {
		f.scratch = make([]byte, 8)
	}
	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			f.body = append(f.body, 1)
		} else {
			f.body = append(f.body, 0)
		}
	case reflect.Int8:
		f.body = append(f.body, byte(v.Int()))
	case reflect.Uint8:
		f.body = append(f.body, byte(v.Uint()))
	case reflect.Int16:
		binary.LittleEndian.PutUint16(f.scratch, uint16(v.Int()))
		f.body = append(f.body, f.scratch[:2]...)
	case reflect.Uint16:
		binary.LittleEndian.PutUint16(f.scratch, uint16(v.Uint()))
		f.body = append(f.body, f.scratch[:2]...)
	case reflect.Int32:
		binary.LittleEndian.PutUint32(f.scratch, uint32(v.Int()))
		f.body = append(f.body, f.scratch[:4]...)
	case reflect.Uint32:
		binary.LittleEndian.PutUint32(f.scratch, uint32(v.Uint()))
		f.body = append(f.body, f.scratch[:4]...)
	case reflect.Int64:
		binary.LittleEndian.PutUint64(f.scratch, uint64(v.Int()))
		f.body = append(f.body, f.scratch[:8]...)
	case reflect.Uint64:
		binary.LittleEndian.PutUint64(f.scratch, v.Uint())
		f.body = append(f.body, f.scratch[:8]...)
	case reflect.Float32:
		binary.LittleEndian.PutUint32(f.scratch, math.Float32bits(float32(v.Float())))
		f.body = append(f.body, f.scratch[:4]...)
	case reflect.Float64:
		binary.LittleEndian.PutUint64(f.scratch, math.Float64bits(v.Float()))
		f.body = append(f.body, f.scratch[:8]...)
	default:
		panic("not fixed")
	}
}
func (f *Fractus) Reset(n int) {
	if f.buf != nil && n == 0 {
		f.buf = f.buf[:0]
		f.body = f.body[:0]
	} else { // first run
		f.buf = make([]byte, 0, n*2+32)
		// collect variable field offsets while building body
		f.body = make([]byte, 0, 64)
	}
}

// Decode: compute fixed offsets; read varOffsets; slice body.
// Unsafe string mode returns string without copy.
func (f *Fractus) Decode(data []byte, out any) error {
	f.Reset(0)
	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return ErrNotStructPtr
	}
	dst := v.Elem()
	t := dst.Type()

	// read N
	N, nHdr := readVarUint(data)
	if N == 0 {
		return nil
	}
	cursor := nHdr
	if f.plan == nil {
		f.plan = make(map[reflect.Type]FieldPlan)
	}
	pl, ok := f.plan[t]
	fields := make([]FieldInfo, 0, N)
	if ok {
		fields = pl.field
	} else {
		for i := 0; i < t.NumField(); i++ {
			sf := t.Field(i)
			if sf.PkgPath != "" && !sf.Anonymous {
				continue
			}
			k := sf.Type.Kind()
			fields = append(fields, FieldInfo{idx: i, kind: k, isVar: !isFixedKind(k)})
			if len(fields) == int(N) {
				break
			}
		}
	}

	// read varOffsets
	var varOffsets []int
	for _, field := range fields {
		if field.isVar {
			off, n := readVarUint(data[cursor:])
			cursor += n
			varOffsets = append(varOffsets, int(off))
		}
	}

	f.body = data[cursor:]
	bodyPos := 0
	var varIdx int
	for _, field := range fields {
		fv := dst.Field(field.idx)
		if field.isVar {
			start := varOffsets[varIdx]
			varIdx++
			// read len
			lv, n := readVarUint(f.body[start:])
			payload := f.body[start+n : start+n+int(lv)]
			bodyPos += int(lv) + n
			switch field.kind {
			case reflect.String:
				if f.Opts.UnsafeStrings {
					str := unsafe.String(&payload[0], len(payload))
					fv.SetString(str)
				} else {
					fv.SetString(string(payload))
				}
			case reflect.Slice:
				if fv.Type().Elem().Kind() == reflect.Uint8 {
					fv.SetBytes(payload)
				} else {
					// list decoding (simple mode)
					// re-run from start: first varint is count, then elements
					cnt, n2 := readVarUint(f.body[start:])
					pos := start + n2
					elemK := fv.Type().Elem().Kind()
					slice := reflect.MakeSlice(fv.Type(), int(cnt), int(cnt))
					for i := 0; i < int(cnt); i++ {
						ev := slice.Index(i)
						if isFixedKind(elemK) {
							sz := FixedSize(elemK)
							setFixed(ev, f.body[pos:pos+sz], elemK)
							pos += sz
						} else if elemK == reflect.String {
							ll, ln := readVarUint(f.body[pos:])
							pos += ln
							ev.SetString(string(f.body[pos : pos+int(ll)]))
							pos += int(ll)
						} else if elemK == reflect.Uint8 {
							ll, ln := readVarUint(f.body[pos:])
							pos += ln
							ev.SetBytes(f.body[pos : pos+int(ll)])
							pos += int(ll)
						} else {
							return ErrUnsupported
						}
					}
					fv.Set(slice)
				}
			}
		} else {
			sz := FixedSize(field.kind)
			setFixed(fv, f.body[bodyPos:bodyPos+sz], field.kind)
			bodyPos += sz
		}
	}
	return nil
}
