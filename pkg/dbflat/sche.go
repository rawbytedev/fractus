package dbflat

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
)

/*
Schema Handling Notes:

- structFieldValue converts a struct to []FieldValue using reflection.
- Build converters for basic types (e.g., string, int, float) for efficient encoding.
- Use a map[tag]fieldName for fast field lookup and random access.
- FieldValue is passed to an encoder to produce binary data.
- Tags can be continuous or non-continuous, allowing flexible field access.
- Allow customization and extensibility for users.

Decoding Strategy:

- Load the target struct and binary data, using a map[tag]fieldName for mapping.
- Parse each element, converting to the appropriate type using reflection.
- Unknown tags are skipped for forward compatibility.
- Backward compatibility is handled by checking the length of the binary data.
- Versioning can be managed with a version tag for easier upgrades.

Partial Decoding and Random Access:

- Use maps to enable partial decoding and random field access.
- Support both fixed-size and variable-size fields.
- For version upgrades, parse known fields first, then handle additional fields as needed.
- Old versions can parse their known fields, while new versions can handle extra fields.
- Consider trade-offs between random access and version compatibility.

Future Improvements:

- Consider loading schema setups from external files for flexibility.
- Optimize reflection usage for better performance.
- Explore lightweight alternatives to reflection if available.
*/

/*
Example usage:

	func (s *StructType) Decode(data []byte) error {
		// Implement decoding logic here
		return nil
	}
*/
func BinToStruct(inter any, bin []byte, loc map[uint16]string) error {
	// inter must be a pointer to struct
	v := reflect.ValueOf(inter)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("inter must be pointer to struct")
	}
	structVal := v.Elem()

	i, err := Inspect(bin, &Decoder{})
	if err != nil {
		return err
	}
	for i.Next() {
		tag := i.Peek()
		name, ok := loc[tag]
		if !ok {
			continue // skip unknown tags
		}
		field := structVal.FieldByName(name)
		if !field.CanSet() {
			continue // skip unexported fields
		}
		setField(field, i.binary())
	}
	return nil
}

func setField(val reflect.Value, data []byte) {
	switch val.Kind() {
	case reflect.String:
		val.SetString(string(data))
	default:
		return
	}
}

// inter represent the struct to convert and loc is a map[tag]name
func StructFieldValue(inter any, loc map[uint16]string) []FieldValue {
	var fields []FieldValue
	for tag, item := range loc {
		val := reflect.ValueOf(inter).FieldByName(item)
		field := convert(val)
		field.Tag = tag
		// compflag can be set they too
		field.CompFlags = 0x8000
		fields = append(fields, field)
	}
	return fields
}

// compflags can be set over here
func convert(val reflect.Value) FieldValue {
	switch val.Kind() {
	case reflect.String:
		return FieldValue{Payload: Writer(val.String())}
	case reflect.Int:
		return FieldValue{Payload: Writer(val.Int())}
	case reflect.Uint:
		return FieldValue{Payload: Writer(val.Uint())}
	case reflect.Float64:
		return FieldValue{Payload: Writer(val.Float())}
	default:
		panic("not implemented")
	}
}
func Writer(value any) []byte {
	switch v := value.(type) {
	case uint8:
		return []byte{v}
	case uint16:
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, v)
		return buf
	case uint32:
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, v)
		return buf
	case uint64:
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, v)
		return buf
	case int8:
		return []byte{byte(v)}
	case int16:
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, uint16(v))
		return buf
	case int32:
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(v))
		return buf
	case int64:
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(v))
		return buf
	case float32:
		bits := math.Float32bits(v)
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, bits)
		return buf
	case float64:
		bits := math.Float64bits(v)
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, bits)
		return buf
	case string:
		return []byte(v)
	case []byte:
		return v
	default:
		return nil
	}
}
