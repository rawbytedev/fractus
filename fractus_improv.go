package fractus

import (
	"encoding/binary"
	"errors"
	"math"
	"reflect"
	"sync"
	"unsafe"
)

var (
	ErrNotStruct    = errors.New("expected struct")
	ErrNotStructPtr = errors.New("expected pointer to struct")
	ErrUnsupported  = errors.New("unsupported type")
)

type SafeOptions struct {
	UnsafeStrings    bool
	UnsafePrimitives bool
	CheckAlignment   bool
}

type HighPerfFractus struct {
	Opts    SafeOptions
	plan    map[reflect.Type]*HFieldPlan
	mu      sync.RWMutex
	scratch []byte
	buf     []byte
	body    []byte
}

type HFieldPlan struct {
	fieldCount int
	varCount   int
	fixedSize  int
	fields     []HFieldInfo
}

type HFieldInfo struct {
	idx       int
	kind      reflect.Kind
	isVar     bool
	size      int
	alignment int
}

// NewHighPerfFractus returns a new HighPerfFractus with options
// HighPerfFractus can be used for both encoding and decoding
func NewHighPerfFractus(opts SafeOptions) *HighPerfFractus {
	return &HighPerfFractus{
		Opts:    opts,
		plan:    make(map[reflect.Type]*HFieldPlan),
		scratch: make([]byte, 8),
	}
}

// getPlan return a FieldPlan
// FieldPlans are used internally
// and not expected to be used externally
// It is concurrent safe
func (f *HighPerfFractus) getPlan(t reflect.Type) *HFieldPlan {
	f.mu.RLock()
	if plan, ok := f.plan[t]; ok {
		f.mu.RUnlock()
		return plan
	}
	f.mu.RUnlock()

	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check
	if plan, ok := f.plan[t]; ok {
		return plan
	}

	plan := &HFieldPlan{}
	fixedSize := 0
	varCount := 0

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" && !sf.Anonymous {
			continue
		}

		kind := sf.Type.Kind()
		isVar := !isFixedKind(kind)
		size := FixedSize(kind)
		alignment := getAlignment(kind)

		fieldInfo := HFieldInfo{
			idx:       i,
			kind:      kind,
			isVar:     isVar,
			size:      size,
			alignment: alignment,
		}

		plan.fields = append(plan.fields, fieldInfo)

		if isVar {
			varCount++
		} else {
			fixedSize += size
		}
	}

	plan.fieldCount = len(plan.fields)
	plan.varCount = varCount
	plan.fixedSize = fixedSize

	f.plan[t] = plan
	return plan
}

// Reset clears all buffer used during encoding/decoding
func (f *HighPerfFractus) Reset() {
	if f.buf != nil {
		f.buf = f.buf[:0]
	}
	if f.body != nil {
		f.body = f.body[:0]
	}
}

// Encode turns the value provided into a byte slice(Fractus format binary)
// The binary reflect the value encoded
// only struct are accepted, only exported values are encoded
func (f *HighPerfFractus) Encode(in any) (out []byte, err error) {
	v := reflect.ValueOf(in)
	// basics checks
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	// only accept structs
	if v.Kind() != reflect.Struct {
		return nil, ErrNotStruct
	}

	t := v.Type()
	// retrieve plan
	plan := f.getPlan(t)
	estimatedSize := 16 + plan.fixedSize + (plan.varCount * 32) // Base + fixed + var
	f.Reset()                                                   // reset buffer

	// increase buffers size if needed
	if cap(f.buf) < estimatedSize {
		f.buf = make([]byte, 0, estimatedSize)
	}
	if cap(f.body) < estimatedSize {
		f.body = make([]byte, 0, estimatedSize)
	}

	// Write number of field discovered
	f.buf = writeVarUint(f.buf, uint64(plan.fieldCount))

	// Encoding each fields
	for _, field := range plan.fields {
		fieldValue := v.Field(field.idx)
		if field.isVar {
			// Encode directly into body
			switch field.kind {
			// string
			case reflect.String:
				if f.Opts.UnsafeStrings {
					// unsafe encoding of strings
					str := fieldValue.String()
					strData := unsafe.Slice(unsafe.StringData(str), len(str))
					f.body = writeVarUint(f.body, uint64(len(strData)))
					f.body = append(f.body, strData...)
				} else {
					str := fieldValue.String()
					f.body = writeVarUint(f.body, uint64(len(str)))
					f.body = append(f.body, str...)
				}
			// slices
			case reflect.Slice:
				elemKind := fieldValue.Type().Elem().Kind()
				length := fieldValue.Len()
				f.body = writeVarUint(f.body, uint64(length))

				if f.Opts.UnsafePrimitives && isFixedKind(elemKind) && length > 0 {
					// unsafe encoding for slices of fixed Size
					if !f.Opts.CheckAlignment || f.checkSliceAlignment(fieldValue, elemKind) {
						fslice := fieldValue.Slice(0, fieldValue.Len())
						if fslice.Len() != 0 {
							elemSize := FixedSize(elemKind)
							byteSlice := unsafe.Slice((*byte)(unsafe.Pointer(fslice.Pointer())), length*elemSize)
							f.body = append(f.body, byteSlice...)
						}
					} else {
						// Safe copy for unaligned data
						for i := 0; i < length; i++ {
							elem := fieldValue.Index(i)
							f.body = f.encodeFixedToBuffer(elem, elemKind, f.body)
						}
					}
				} else {
					// Encode each element
					for i := 0; i < length; i++ {
						elem := fieldValue.Index(i)
						if isFixedKind(elemKind) {
							f.body = f.encodeFixedToBuffer(elem, elemKind, f.body)
						} else if elemKind == reflect.String {
							if f.Opts.UnsafeStrings {
								strData := unsafe.Slice(unsafe.StringData(elem.String()), len(elem.String()))
								f.body = writeVarUint(f.body, uint64(len(strData)))
								f.body = append(f.body, strData...)
							} else {
								str := elem.String()
								f.body = writeVarUint(f.body, uint64(len(str)))
								f.body = append(f.body, str...)
							}
						} else {
							return nil, ErrUnsupported
						}
					}
				}
			default:
				return nil, ErrUnsupported
			}
		} else {
			// Fixed field - encode directly to body
			f.encodeFixedToBody(fieldValue, field.kind)
		}
	}

	// Append body to buffer
	f.buf = append(f.buf, f.body...)
	return f.buf, nil
}

// / *** Merging both into a single function is possible
// encodes value based on their types
func (f *HighPerfFractus) encodeFixedToBody(v reflect.Value, kind reflect.Kind) {
	switch kind {
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
		panic("unsupported fixed kind")
	}
}

// encodes value based on their types
func (f *HighPerfFractus) encodeFixedToBuffer(v reflect.Value, kind reflect.Kind, dst []byte) []byte {
	switch kind {
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
		binary.LittleEndian.PutUint16(f.scratch, uint16(v.Int()))
		return append(dst, f.scratch[:2]...)
	case reflect.Uint16:
		binary.LittleEndian.PutUint16(f.scratch, uint16(v.Uint()))
		return append(dst, f.scratch[:2]...)
	case reflect.Int32:
		binary.LittleEndian.PutUint32(f.scratch, uint32(v.Int()))
		return append(dst, f.scratch[:4]...)
	case reflect.Uint32:
		binary.LittleEndian.PutUint32(f.scratch, uint32(v.Uint()))
		return append(dst, f.scratch[:4]...)
	case reflect.Int64:
		binary.LittleEndian.PutUint64(f.scratch, uint64(v.Int()))
		return append(dst, f.scratch[:8]...)
	case reflect.Uint64:
		binary.LittleEndian.PutUint64(f.scratch, v.Uint())
		return append(dst, f.scratch[:8]...)
	case reflect.Float32:
		binary.LittleEndian.PutUint32(f.scratch, math.Float32bits(float32(v.Float())))
		return append(dst, f.scratch[:4]...)
	case reflect.Float64:
		binary.LittleEndian.PutUint64(f.scratch, math.Float64bits(v.Float()))
		return append(dst, f.scratch[:8]...)
	default:
		panic("unsupported fixed kind")
	}
}

// / ***
// Decode turns the provided byte slice into value
// The type of the decoded values should be compatible with the respective values in out.
func (f *HighPerfFractus) Decode(in []byte, out any) (err error) {
	f.Reset()
	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return ErrNotStructPtr
	}

	dst := v.Elem()
	t := dst.Type()
	plan := f.getPlan(t)

	// Read field count
	N, cursor := readVarUint(in)
	if N == 0 {
		return nil
	}

	// Set body reference
	f.body = in[cursor:]
	bodyPos := 0
	var varIdx int

	// Decode fields in order
	for _, field := range plan.fields {
		fv := dst.Field(field.idx)
		if field.isVar {
			varIdx++
			// Read length prefix
			switch field.kind {
			case reflect.String:
				length, n := readVarUint(f.body[bodyPos:])
				payload := f.body[bodyPos+n : bodyPos+n+int(length)]
				bodyPos += int(length) + n
				if f.Opts.UnsafeStrings {
					if len(payload) > 0 {
						str := unsafe.String(&payload[0], len(payload))
						fv.SetString(str)
					} else {
						fv.SetString("")
					}
				} else {
					fv.SetString(string(payload))
				}
			case reflect.Slice:
				elemKind := fv.Type().Elem().Kind()
				count, n2 := readVarUint(f.body[bodyPos:])
				pos := bodyPos + n2

				bodyPos += n2
				if f.Opts.UnsafePrimitives && isFixedKind(elemKind) && int(count) > 0 {
					// Zero-copy for primitive slices
					elemSize := FixedSize(elemKind)
					requiredSize := int(count) * elemSize
					if pos+requiredSize <= len(f.body) {
						setUnsafeFixed(fv, f.body[bodyPos:], elemKind, int(count))
						pos += requiredSize
					} else { // fallback
						//  allocate only when needed
						slice := reflect.MakeSlice(fv.Type(), int(count), int(count))
						// Fall back to safe decoding
						for i := 0; i < int(count); i++ {
							elem := slice.Index(i)
							setFixed(elem, f.body[pos:pos+elemSize], elemKind)
							pos += elemSize
						}
						fv.Set(slice)
					}
					bodyPos += requiredSize
				} else {
					//  allocate only when needed
					slice := reflect.MakeSlice(fv.Type(), int(count), int(count))
					// Safe element-by-element decoding
					for i := 0; i < int(count); i++ {
						elem := slice.Index(i)
						if isFixedKind(elemKind) {
							size := FixedSize(elemKind)
							setFixed(elem, f.body[pos:pos+size], elemKind)
							pos += size
							bodyPos += size
						} else if elemKind == reflect.String {
							strLen, n3 := readVarUint(f.body[pos:])
							pos += n3
							strData := f.body[pos : pos+int(strLen)]
							if f.Opts.UnsafeStrings {
								elem.SetString(unsafe.String(&strData[0], len(strData)))
							} else {
								elem.SetString(string(strData))
							}
							pos += int(strLen)
							bodyPos += int(strLen) + n3
						} else {
							return ErrUnsupported
						}
					}
					fv.Set(slice)
				}
			}
		} else {
			// Fixed field
			size := FixedSize(field.kind)
			setFixed(fv, f.body[bodyPos:bodyPos+size], field.kind)
			bodyPos += size
		}
	}

	return nil
}

// Checks alignement to avoid common issues
// internal function
func (f *HighPerfFractus) checkSliceAlignment(v reflect.Value, elemKind reflect.Kind) bool {
	if v.Len() == 0 {
		return true
	}
	addr := v.Index(0).UnsafeAddr()
	alignment := getAlignment(elemKind)
	return addr%uintptr(alignment) == 0
}

// getAlignment return the size of the type(fixed size types)
// internal
func getAlignment(kind reflect.Kind) int {
	switch kind {
	case reflect.Int8, reflect.Uint8, reflect.Bool:
		return 1
	case reflect.Int16, reflect.Uint16:
		return 2
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		return 4
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		return 8
	default:
		return 1
	}
}

// SafeDecoder for memory lifetime management
// Ensure that Decoding using unsafe doesn't
// affect values
type SafeDecoder struct {
	fractus *HighPerfFractus
	payload []byte
}

func NewSafeDecoder(fractus *HighPerfFractus) *SafeDecoder {
	return &SafeDecoder{
		fractus: fractus,
	}
}

func (s *SafeDecoder) Decode(data []byte, out any) error {
	s.payload = data
	return s.fractus.Decode(data, out)
}
