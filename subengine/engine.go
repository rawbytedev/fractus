package engine

import (
	"encoding/binary"
	"errors"
	"fmt"
	"fractus/internal/common"

	"github.com/klauspost/compress/zstd"
)

type Record struct {
	payloadHot []byte
	vt         []byte
	tagwalk    []byte
	out        []byte
	//tmp        []byte
	header []byte
}
type offsetLoc struct {
	tag      uint16
	compflag uint16
	offset   uint32
}
type LayoutPlan struct {
	Fields      []FieldValue // input fields
	HotTags     []uint16     // tags to prioritize
	HeaderFlags uint16       // layout control
	SchemaID    uint64       // optional
}

// EncodeRecordHot builds a hot-vtable record (header + vtable for hot fields + payloadHot + tagwalk coldfields)
// This is a simplified port of pkg/dbflat Encoder.EncodeRecordHot but kept here so zc can be independently tested.
func (r *Record) EncodeRecordHot(schemaID uint64, hotTags []uint16, fields []FieldValue) ([]byte, error) {
	r.Reset()
	// validate hotTags
	for _, h := range hotTags {
		if h == 0 || h > 8 {
			return nil, fmt.Errorf("invalid hot field tag: %d", h)
		}
	}
	// Partition
	hot, cold := PartitionFields(fields, hotTags)
	var hotOffsets []offsetLoc
	// Generate hot payload and offsets
	r.payloadHot, hotOffsets = r.GenPayloads(hot)

	// vtable for hot
	r.vt = r.GeneVtables(hotOffsets)

	// tagwalk for cold
	r.tagwalk = r.GenTagWalk(cold)

	// Build header
	header := BuildHeader(r.vt, &LayoutPlan{HeaderFlags: FlagPadding | FlagModeHotVtable, SchemaID: schemaID, HotTags: hotTags})
	headerBytes := r.encodeHeader(*header)
	// Join header + vtable + payloadHot + tagwalk
	maxSize := len(headerBytes) + len(r.vt) + len(r.payloadHot) + len(r.tagwalk)
	if cap(r.out) < maxSize {
		r.out = make([]byte, 0, maxSize)
	} else {
		r.out = r.out[:maxSize]
	}
	r.out = append(r.out, headerBytes...)
	r.out = append(r.out, r.vt...)
	r.out = append(r.out, r.payloadHot...)
	r.out = append(r.out, r.tagwalk...)

	return r.out, nil
}

// GenTagWalk generates a tag-walk payload (tag+compflag+[varint len]+payload...)
func (r *Record) GenTagWalk(fields []FieldValue) []byte {
	est := 0
	for _, f := range fields {
		est += 4 + len(f.Payload)
		if f.CompFlags&ArrayMask != 0 {
			est += 10
		}
	}
	if cap(r.tagwalk) < est {
		r.tagwalk = make([]byte, 0, est)
	} else {
		r.tagwalk = r.tagwalk[:est]
	}
	for _, field := range fields {
		// append tag (uint16 little-endian)
		tag := field.Tag
		r.tagwalk = append(r.tagwalk, byte(tag) , byte(tag<<8))
		// append compFlags (uint16 little-endian)
		cf := field.CompFlags
		r.tagwalk = append(r.tagwalk, byte(cf) , byte(cf<<8))

		// if array, write varint length directly into tmp without allocating
		if field.CompFlags&ArrayMask != 0 {
			r.tagwalk = common.WriteVarUintTo(r.tagwalk, uint64(len(field.Payload)))
		}

		// append payload
		r.tagwalk = append(r.tagwalk, field.Payload...)
	}
	return r.tagwalk
}

func (r *Record) Reset() {
	r.vt = r.vt[:0]
	r.payloadHot = r.payloadHot[:0]
	r.tagwalk = r.tagwalk[:0]
	r.out = r.out[:0]
	//r.tmp = r.tmp[:0]
	r.header = r.header[:0]
}
func BuildHeader(vtable []byte, plan *LayoutPlan) *Header {
	if vtable != nil {
		h := &Header{
			Magic:       MagicV1,
			Version:     VersionV1,
			Flags:       plan.HeaderFlags,
			SchemaID:    plan.SchemaID,
			HotBitmap:   buildHotBitmap(plan.HotTags),
			VTableSlots: byte(len(vtable) / 8),
			DataOffset:  uint16(HeaderSize + len(vtable)),
			VTableOff:   uint32(HeaderSize),
		}
		return h
	}
	return nil

}

func PartitionFields(fields []FieldValue, hotTags []uint16) ([]FieldValue, []FieldValue) {
	// Build a set for fast hot tag lookup
	hotSet := make(map[uint16]struct{}, len(hotTags))
	for _, tag := range hotTags {
		hotSet[tag] = struct{}{}
	}
	var hot, cold []FieldValue
	for _, f := range fields {
		if _, isHot := hotSet[f.Tag]; isHot {
			hot = append(hot, f)
		} else {
			cold = append(cold, f)
		}
	}
	return hot, cold
}
func (r *Record) GenPayloads(fields []FieldValue) ([]byte, []offsetLoc) {
	next := 0
	// fieldbuf removed; using writeVarUintTo to write directly into tmp
	var offmap []offsetLoc
	for _, field := range fields {
		// Handle compression
		if field.CompFlags&^ArrayMask != CompRaw {
			// compress payload and prefix with uncompressed length varint
			comp, err := compressData(field.CompFlags, field.Payload)
			if err == nil {
				r.payloadHot = common.WriteVarUintTo(r.payloadHot, uint64(len(comp)))
				r.payloadHot = append(r.payloadHot, comp...)
				offmap = append(offmap, offsetLoc{tag: field.Tag, compflag: field.CompFlags, offset: uint32(next)})
				next = len(r.payloadHot)
				continue
			}
			// if compression failed, fall back to raw payload
		}
		if field.CompFlags&ArrayMask != 0 {
			r.payloadHot = common.WriteVarUintTo(r.payloadHot, uint64(len(field.Payload)))
		}
		r.payloadHot = append(r.payloadHot, field.Payload...)
		offmap = append(offmap, offsetLoc{tag: field.Tag, compflag: field.CompFlags, offset: uint32(next)})
		next = len(r.payloadHot)
	}
	return r.payloadHot, offmap
}

func (r *Record) GeneVtables(offsets []offsetLoc) []byte {
	vtSize := len(offsets) * 8
	if cap(r.vt) < vtSize {
		r.vt = make([]byte, vtSize)
	} else {
		r.vt = r.vt[:vtSize]
	}
	for i, offmap := range offsets {
		idx := i * 8
		binary.LittleEndian.PutUint16(r.vt[idx:], offmap.tag)
		binary.LittleEndian.PutUint16(r.vt[idx+2:], offmap.compflag)
		binary.LittleEndian.PutUint32(r.vt[idx+4:], offmap.offset)
	}
	return r.vt
}

// buildHotBitmap for tags 1–8 (duplicate from dbflat)
func buildHotBitmap(tags []uint16) byte {
	var bm byte
	for _, t := range tags {
		if t >= 1 && t <= 8 {
			bm |= 1 << (t - 1)
		}
	}
	return bm
}

// encodeHeader serializes Header into buf
func (r *Record) encodeHeader(h Header) []byte {
	if cap(r.header) < HeaderSize {
		r.header = make([]byte, HeaderSize)
	}
	if h.Flags&FlagNoSchemaID != 0 {
		r.header = r.header[:HeaderSize-8]
	}
	binary.LittleEndian.PutUint32(r.header[0:], h.Magic)
	binary.LittleEndian.PutUint16(r.header[4:], h.Version)
	binary.LittleEndian.PutUint16(r.header[6:], h.Flags)
	if h.Flags&FlagNoSchemaID != 0 {
		r.header[8] = h.HotBitmap
		r.header[9] = h.VTableSlots
		binary.LittleEndian.PutUint16(r.header[10:], h.DataOffset)
		binary.LittleEndian.PutUint32(r.header[12:], h.VTableOff)
		return r.header
	} else {
		binary.LittleEndian.PutUint64(r.header[8:], h.SchemaID)
		r.header[16] = h.HotBitmap
		r.header[17] = h.VTableSlots
		binary.LittleEndian.PutUint16(r.header[18:], h.DataOffset)
		binary.LittleEndian.PutUint32(r.header[20:], h.VTableOff)
		return r.header
	}
}

// compressData / decompressData copied from dbflat.compress.go
func compressData(compFlags uint16, raw []byte) ([]byte, error) {
	switch compFlags &^ ArrayMask {
	case CompRaw:
		return raw, nil
	case CompRLE:
		return rleEncode(raw), nil
	case CompHuffman:
		return huffmanEncode(raw)
	case CompZstd:
		bestLevel := zstd.WithEncoderLevel(zstd.SpeedBetterCompression)
		enc, err := zstd.NewWriter(nil, bestLevel)
		if err != nil {
			return nil, err
		}
		return enc.EncodeAll(raw, nil), nil
	default:
		return nil, errors.New("unknown compFlags")
	}
}

func rleEncode(src []byte) []byte              { return make([]byte, 0) }
func huffmanEncode(src []byte) ([]byte, error) { return make([]byte, 0), nil }
