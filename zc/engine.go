package zc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"fractus/internal/common"

	"github.com/klauspost/compress/zstd"
)

type zc struct {
	buff []byte
	*HotRecord
}
type HotRecord struct {
	vt         []byte
	payloadHot []byte
	tagwalk    []byte
	out        []byte
}

func NewZeroCopy() *zc {
	return &zc{HotRecord: &HotRecord{}}
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

// encodeHeader serializes Header into buf (duplicate of dbflat.encodeHeader)
func encodeHeader(buf []byte, h Header) []byte {
	if h.Flags&FlagNoSchemaID != 0 {
		buf = append(buf, make([]byte, HeaderSize-8)...)
	} else {
		buf = append(buf, make([]byte, HeaderSize)...)
	}
	binary.LittleEndian.PutUint32(buf[0:], h.Magic)
	binary.LittleEndian.PutUint16(buf[4:], h.Version)
	binary.LittleEndian.PutUint16(buf[6:], h.Flags)
	if h.Flags&FlagNoSchemaID != 0 {
		buf[8] = h.HotBitmap
		buf[9] = h.VTableSlots
		binary.LittleEndian.PutUint16(buf[10:], h.DataOffset)
		binary.LittleEndian.PutUint32(buf[12:], h.VTableOff)
		return buf
	} else {
		binary.LittleEndian.PutUint64(buf[8:], h.SchemaID)
		buf[16] = h.HotBitmap
		buf[17] = h.VTableSlots
		binary.LittleEndian.PutUint16(buf[18:], h.DataOffset)
		binary.LittleEndian.PutUint32(buf[20:], h.VTableOff)
		return buf
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

// GenTagWalk generates a tag-walk payload (tag+compflag+[varint len]+payload...)
func (z *zc) GenTagWalk(fields []FieldValue) []byte {
	est := 0
	for _, f := range fields {
		est += 4 + len(f.Payload)
		if f.CompFlags&ArrayMask != 0 {
			est += 10
		}
	}
	tmp := make([]byte, 0, est)
	for _, field := range fields {
		// append tag (uint16 little-endian)
		tag := field.Tag
		tmp = append(tmp, byte(tag), byte(tag<<8))
		// append compFlags (uint16 little-endian)
		cf := field.CompFlags
		tmp = append(tmp, byte(cf), byte(cf<<8))

		// if array, write varint length directly into tmp without allocating
		if field.CompFlags&ArrayMask != 0 {
			tmp = common.WriteVarUintTo(tmp, uint64(len(field.Payload)))
		}

		// append payload
		tmp = append(tmp, field.Payload...)
	}
	return tmp
}

// EncodeRecordTagWalk is a convenience wrapper that returns the TagWalk payload.
func (r *zc) EncodeRecordTagWalk(fields []FieldValue) ([]byte, error) {
	return r.GenTagWalk(fields), nil
}

// EncodeRecordHot builds a hot-vtable record (header + vtable for hot fields + payloadHot + tagwalk coldfields)
// This is a simplified port of pkg/dbflat Encoder.EncodeRecordHot but kept here so zc can be independently tested.
func (r *zc) EncodeRecordHot(schemaID uint64, hotTags []uint16, fields []FieldValue) ([]byte, error) {
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
	r.payloadHot, hotOffsets = GenPayloads(hot)

	// vtable for hot
	r.vt = GeneVtables(hotOffsets)

	// tagwalk for cold
	r.tagwalk = r.GenTagWalk(cold)

	// Build header
	header := BuildHeader(r.vt, &LayoutPlan{HeaderFlags: FlagPadding | FlagModeHotVtable, SchemaID: schemaID, HotTags: hotTags})
	headerBytes := encodeHeader(nil, *header)
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
func (r *HotRecord) Reset() {
	r.vt = r.vt[:0]
	r.payloadHot = r.payloadHot[:0]
	r.tagwalk = r.tagwalk[:0]
	r.out = r.out[:0]
}

// PartitionFields splits fields into hot and cold slices based on hotTags.
// - Only fields present in both the input and hotTags are considered hot.
// - The function is O(n) in the number of fields, with O(1) hot tag lookup.
// - If a tag is in hotTags but not present in fields, it is ignored (no wasted search).
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

type LayoutPlan struct {
	Fields      []FieldValue // input fields
	HotTags     []uint16     // tags to prioritize
	HeaderFlags uint16       // layout control
	SchemaID    uint64       // optional
}

// Helpers: GenPayloads and GeneVtables duplicated but using db types
type offsetLoc struct {
	tag      uint16
	compflag uint16
	offset   uint32
}

func GenPayloads(fields []FieldValue) ([]byte, []offsetLoc) {
	var tmp []byte
	next := 0
	// fieldbuf removed; using writeVarUintTo to write directly into tmp
	var offmap []offsetLoc
	for _, field := range fields {
		// Handle compression
		if field.CompFlags&^ArrayMask != CompRaw {
			// compress payload and prefix with uncompressed length varint
			comp, err := compressData(field.CompFlags, field.Payload)
			if err == nil {
				tmp = common.WriteVarUintTo(tmp, uint64(len(comp)))
				tmp = append(tmp, comp...)
				offmap = append(offmap, offsetLoc{tag: field.Tag, compflag: field.CompFlags, offset: uint32(next)})
				next = len(tmp)
				continue
			}
			// if compression failed, fall back to raw payload
		}
		if field.CompFlags&ArrayMask != 0 {
			tmp = common.WriteVarUintTo(tmp, uint64(len(field.Payload)))
		}
		tmp = append(tmp, field.Payload...)
		offmap = append(offmap, offsetLoc{tag: field.Tag, compflag: field.CompFlags, offset: uint32(next)})
		next = len(tmp)
	}
	return tmp, offmap
}

func GeneVtables(offsets []offsetLoc) []byte {
	vtSize := len(offsets) * 8
	vtBuf := make([]byte, vtSize)
	for i, offmap := range offsets {
		idx := i * 8
		binary.LittleEndian.PutUint16(vtBuf[idx:], offmap.tag)
		binary.LittleEndian.PutUint16(vtBuf[idx+2:], offmap.compflag)
		binary.LittleEndian.PutUint32(vtBuf[idx+4:], offmap.offset)
	}
	return vtBuf
}
