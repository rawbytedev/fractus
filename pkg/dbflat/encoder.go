package dbflat

import (
	"encoding/binary"
	"fmt"
	"slices"
	"sort"
)

// buildHotBitmap for tags 1–8
// HotFields
func buildHotBitmap(tags []uint16) byte {
	var bm byte
	for _, t := range tags {
		if t >= 1 && t <= 8 {
			bm |= 1 << (t - 1)
		}
	}
	return bm
}

type Encoder struct {
	headerBuf   []byte
	compressed  []byte
	vtBuf       []byte
	dataBuf     []byte
	fieldBuf    []byte   // for per-field varints
	offsets     []uint32 //reused
	out         []byte
	zeroPadding [8]byte
	tmpfields   []FieldValue
	headerflag  uint16 // 2B: bit0=align8,bit1=schemaID
}

// ------------------------------------------------------------------------------
// Encoders Section
// ------------------------------------------------------------------------------

// --- Default Encoder optional flag for header is align8(create 8byte padding) or not
// Avoid using padding to save up space considerably
// sorted tags offer faster lookup than normal due to use of vtable
// position can be computed directly vtable(tag:2B+comFlags:2B+offset:4B)
func (e *Encoder) EncodeRecordFull(schemaID uint64, hotTags []uint16, fields []FieldValue) ([]byte, error) {
	if cap(e.tmpfields) < len(fields) {
		e.tmpfields = make([]FieldValue, len(fields))
	} else {
		e.tmpfields = e.tmpfields[:len(fields)]
	}
	copy(e.tmpfields, fields)
	if !isSortedByTag(e.tmpfields) {
		sort.Slice(e.tmpfields, func(i, j int) bool { return e.tmpfields[i].Tag < e.tmpfields[j].Tag })
	}

	// Reset buffers
	e.headerBuf = e.headerBuf[:0]
	e.vtBuf = e.vtBuf[:0]
	e.dataBuf = e.dataBuf[:0]
	e.fieldBuf = e.fieldBuf[:0]
	e.compressed = e.compressed[:0]

	// Ensure offsets slice fits
	if cap(e.offsets) < len(e.tmpfields) {
		e.offsets = make([]uint32, len(e.tmpfields))
	} else {
		e.offsets = e.offsets[:len(e.tmpfields)]
	}

	// --- Encode field payloads ---
	for i, f := range e.tmpfields {
		// Align to 8 bytes if flag set
		if e.headerflag&0x0001 != 0 {
			pad := align(len(e.dataBuf), 8) - len(e.dataBuf)
			e.dataBuf = append(e.dataBuf, e.zeroPadding[:pad]...)
		}
		e.offsets[i] = uint32(len(e.dataBuf))

		// Compress or array logic
		switch {
		case f.CompFlags&CompressionMask != 0:

			var err error
			e.compressed, err = compressData(f.CompFlags, f.Payload)
			if err != nil {
				return nil, err
			}
			e.fieldBuf = e.fieldBuf[:0]
			e.writeVarUint(uint64(len(e.compressed)))
			e.dataBuf = append(e.dataBuf, e.fieldBuf...)
			e.dataBuf = append(e.dataBuf, e.compressed...)

		case f.CompFlags&ArrayMask != 0:
			e.fieldBuf = e.fieldBuf[:0]
			e.writeVarUint(uint64(len(f.Payload)))
			e.dataBuf = append(e.dataBuf, e.fieldBuf...)
			e.dataBuf = append(e.dataBuf, f.Payload...)

		default:
			e.dataBuf = append(e.dataBuf, f.Payload...)
		}
	}

	// --- Encode vtable ---
	vtSize := len(e.tmpfields) * 8
	if cap(e.vtBuf) < vtSize {
		e.vtBuf = make([]byte, vtSize)
	}
	e.vtBuf = e.vtBuf[:vtSize]
	for i, f := range e.tmpfields {
		idx := i * 8
		binary.LittleEndian.PutUint16(e.vtBuf[idx:], f.Tag)
		binary.LittleEndian.PutUint16(e.vtBuf[idx+2:], f.CompFlags)
		binary.LittleEndian.PutUint32(e.vtBuf[idx+4:], e.offsets[i])
	}
	sizeH := HeaderSize
	if e.headerflag&FlagNoSchemaID != 0 {
		sizeH = sizeH - 8 // schema is 8B
	}
	// --- Encode header ---
	e.headerBuf = encodeHeader(e.headerBuf[:0], Header{
		Magic:       MagicV1,
		Version:     VersionV1,
		Flags:       e.headerflag,
		SchemaID:    schemaID,
		HotBitmap:   buildHotBitmap(hotTags),
		VTableSlots: byte(len(fields)),
		DataOffset:  uint16(sizeH + len(e.vtBuf)),
		VTableOff:   uint32(sizeH),
	})

	// --- Final payload ---
	total := len(e.headerBuf) + len(e.vtBuf) + len(e.dataBuf)
	if cap(e.out) < total {
		e.out = make([]byte, 0, total*2)
	} else {
		e.out = e.out[:0]
	}
	e.out = append(e.out, e.headerBuf...)
	e.out = append(e.out, e.vtBuf...)
	e.out = append(e.out, e.dataBuf...)

	return e.out, nil
}

// HotVtable Mode Encoder
func (e *Encoder) EncodeRecordHot(schemaID uint64, hotTags []uint16, fields []FieldValue) ([]byte, error) {
	if cap(e.tmpfields) < len(fields) {
		e.tmpfields = make([]FieldValue, len(fields))
	} else {
		e.tmpfields = e.tmpfields[:len(fields)]
	}
	copy(e.tmpfields, fields)
	if !isSortedByTag(e.tmpfields) {
		sort.Slice(e.tmpfields, func(i, j int) bool { return e.tmpfields[i].Tag < e.tmpfields[j].Tag })
	}

	// Reset buffers
	e.headerBuf = e.headerBuf[:0]
	e.vtBuf = e.vtBuf[:0]
	e.dataBuf = e.dataBuf[:0]
	e.fieldBuf = e.fieldBuf[:0]
	e.compressed = e.compressed[:0]

	// --- Encode vtable for hot fields only---
	// comes first
	for _, h := range hotTags {
		if h == 0 || h > 8 {
			return nil, fmt.Errorf("invalid hot field tag: %d", h)
		}
	}
	var hotfields []FieldValue
	e.tmpfields = slices.DeleteFunc(e.tmpfields, func(s FieldValue) bool {
		if s.Tag == 0 || s.Tag > 8 {
			return false
		}
		hotfields = append(hotfields, FieldValue{Tag: s.Tag, CompFlags: s.CompFlags, Payload: s.Payload})
		return true
	})
	/*
		for i, f := range e.tmpfields {
			if f.Tag != 0 && f.Tag < 8 {
				hotfields = append(hotfields, FieldValue{Tag: f.Tag, CompFlags: f.CompFlags, Payload: f.Payload})
				e.tmpfields = append(e.tmpfields[:i], e.tmpfields[i+1:]...)
			}
		}*/

	// Ensure offsets slice fits
	if cap(e.offsets) < len(hotfields) {
		e.offsets = make([]uint32, len(hotfields))
	} else {
		e.offsets = e.offsets[:len(hotfields)]
	}

	// --- Encode field hotfields ---
	for i, f := range hotfields {
		pad := align(len(e.dataBuf), 8) - len(e.dataBuf)
		e.dataBuf = append(e.dataBuf, e.zeroPadding[:pad]...)
		e.offsets[i] = uint32(len(e.dataBuf))
		// Compress or array logic
		switch {
		case f.CompFlags&CompressionMask != 0:

			var err error
			e.compressed, err = compressData(f.CompFlags, f.Payload)
			if err != nil {
				return nil, err
			}
			e.fieldBuf = e.fieldBuf[:0]
			e.writeVarUint(uint64(len(e.compressed)))
			e.dataBuf = append(e.dataBuf, e.fieldBuf...)
			e.dataBuf = append(e.dataBuf, e.compressed...)

		case f.CompFlags&ArrayMask != 0:
			e.fieldBuf = e.fieldBuf[:0]
			e.writeVarUint(uint64(len(f.Payload)))
			e.dataBuf = append(e.dataBuf, e.fieldBuf...)
			e.dataBuf = append(e.dataBuf, f.Payload...)

		default:
			e.dataBuf = append(e.dataBuf, f.Payload...)
		}
	}
	vtSize := len(hotTags) * 8
	if cap(e.vtBuf) < vtSize {
		e.vtBuf = make([]byte, vtSize)
	}
	e.vtBuf = e.vtBuf[:vtSize]
	for i, f := range hotfields {
		idx := i * 8
		binary.LittleEndian.PutUint16(e.vtBuf[idx:], f.Tag)
		binary.LittleEndian.PutUint16(e.vtBuf[idx+2:], f.CompFlags)
		binary.LittleEndian.PutUint32(e.vtBuf[idx+4:], e.offsets[i])
	}

	// --- Encode field coldfields ---
	for i, f := range e.tmpfields {
		pad := align(len(e.dataBuf), 8) - len(e.dataBuf)
		e.dataBuf = append(e.dataBuf, e.zeroPadding[:pad]...)
		e.offsets[i] = uint32(len(e.dataBuf))
		// Compress or array logic
		switch {
		case f.CompFlags&CompressionMask != 0:

			var err error
			e.compressed, err = compressData(f.CompFlags, f.Payload)
			if err != nil {
				return nil, err
			}
			e.fieldBuf = e.fieldBuf[:0]
			e.writeVarUint(uint64(len(e.compressed)))
			e.dataBuf = append(e.dataBuf, byte(f.Tag))
			e.dataBuf = append(e.dataBuf, byte(f.CompFlags))
			e.dataBuf = append(e.dataBuf, e.fieldBuf...)
			e.dataBuf = append(e.dataBuf, e.compressed...)

		case f.CompFlags&ArrayMask != 0:
			e.fieldBuf = e.fieldBuf[:0]
			e.writeVarUint(uint64(len(f.Payload)))
			e.dataBuf = append(e.dataBuf, byte(f.Tag))       // tag
			e.dataBuf = append(e.dataBuf, byte(f.CompFlags)) // comflag
			e.dataBuf = append(e.dataBuf, e.fieldBuf...)     // length
			e.dataBuf = append(e.dataBuf, f.Payload...)      // payload

		default:
			e.dataBuf = append(e.dataBuf, byte(f.Tag))
			e.dataBuf = append(e.dataBuf, byte(f.CompFlags))
			e.dataBuf = append(e.dataBuf, f.Payload...)
		}
	}
	// --- Encode header ---

	e.headerBuf = encodeHeader(e.headerBuf[:0], Header{
		Magic:       MagicV1,
		Version:     VersionV1,
		Flags:       0x0001 | 0x0004,
		SchemaID:    schemaID,
		HotBitmap:   buildHotBitmap(hotTags),
		VTableSlots: byte(len(hotfields)),
		DataOffset:  uint16(HeaderSize + len(e.vtBuf)),
		VTableOff:   uint32(HeaderSize),
	})

	// --- Final payload ---
	total := len(e.headerBuf) + len(e.vtBuf) + len(e.dataBuf)
	if cap(e.out) < total {
		e.out = make([]byte, 0, total*2)
	} else {
		e.out = e.out[:0]
	}
	e.out = append(e.out, e.headerBuf...)
	e.out = append(e.out, e.vtBuf...)
	e.out = append(e.out, e.dataBuf...)

	return e.out, nil
}

// Tag Walk Mode Encoder
func (e *Encoder) EncodeRecordTagWorK(fields []FieldValue) ([]byte, error) {
	if cap(e.tmpfields) < len(fields) {
		e.tmpfields = make([]FieldValue, len(fields))
	} else {
		e.tmpfields = e.tmpfields[:len(fields)]
	}
	copy(e.tmpfields, fields)
	if !isSortedByTag(e.tmpfields) {
		sort.Slice(e.tmpfields, func(i, j int) bool { return e.tmpfields[i].Tag < e.tmpfields[j].Tag })
	}
	// Reset buffers
	e.dataBuf = e.dataBuf[:0]
	e.fieldBuf = e.fieldBuf[:0]
	e.compressed = e.compressed[:0]
	// --- Encode field payloads ---
	for _, f := range e.tmpfields {
		// Align to 8 bytes if flag set
		if e.headerflag&0x0001 != 0 {
			pad := align(len(e.dataBuf), 8) - len(e.dataBuf)
			e.dataBuf = append(e.dataBuf, e.zeroPadding[:pad]...)
		}

		// Compress or array logic
		switch {
		case f.CompFlags&CompressionMask != 0:

			var err error
			e.compressed, err = compressData(f.CompFlags, f.Payload)
			if err != nil {
				return nil, err
			}
			e.fieldBuf = e.fieldBuf[:0]
			e.writeVarUint(uint64(len(e.compressed)))
			e.dataBuf = append(e.dataBuf, ToBytes(f.Tag)...)       // tag
			e.dataBuf = append(e.dataBuf, ToBytes(f.CompFlags)...) // comflag
			e.dataBuf = append(e.dataBuf, e.fieldBuf...)
			e.dataBuf = append(e.dataBuf, e.compressed...)

		case f.CompFlags&ArrayMask != 0:
			e.fieldBuf = e.fieldBuf[:0]
			e.writeVarUint(uint64(len(f.Payload)))
			e.dataBuf = append(e.dataBuf, ToBytes(f.Tag)...)       // tag
			e.dataBuf = append(e.dataBuf, ToBytes(f.CompFlags)...) // comflag
			e.dataBuf = append(e.dataBuf, e.fieldBuf...)
			e.dataBuf = append(e.dataBuf, f.Payload...)

		default:
			e.dataBuf = append(e.dataBuf, ToBytes(f.Tag)...)       // tag
			e.dataBuf = append(e.dataBuf, ToBytes(f.CompFlags)...) // comflag
			e.dataBuf = append(e.dataBuf, f.Payload...)
		}
	}
	return e.dataBuf, nil
}

// Check if field list is sorted
func isSortedByTag(fields []FieldValue) bool {
	for i := 1; i < len(fields); i++ {
		if fields[i-1].Tag > fields[i].Tag {
			return false
		}
	}
	return true
}

func (e *Encoder) writeVarUint(x uint64) {
	e.fieldBuf = e.fieldBuf[:0]
	for x >= 0x80 {
		e.fieldBuf = append(e.fieldBuf, byte(x)|0x80)
		x >>= 7
	}
	e.fieldBuf = append(e.fieldBuf, byte(x))
}

func ToBytes(val any) []byte {
	a, _ := Write(val)
	return a
}
