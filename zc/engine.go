package zc

import (
	"encoding/binary"
	"errors"
	"fmt"
	db "fractus/pkg/dbflat"
	"github.com/klauspost/compress/zstd"
)

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
func encodeHeader(buf []byte, h db.Header) []byte {
	if h.Flags&db.FlagNoSchemaID != 0 {
		buf = append(buf, make([]byte, db.HeaderSize-8)...)
	} else {
		buf = append(buf, make([]byte, db.HeaderSize)...)
	}
	binary.LittleEndian.PutUint32(buf[0:], h.Magic)
	binary.LittleEndian.PutUint16(buf[4:], h.Version)
	binary.LittleEndian.PutUint16(buf[6:], h.Flags)
	if h.Flags&db.FlagNoSchemaID != 0 {
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
	switch compFlags &^ db.ArrayMask {
	case db.CompRaw:
		return raw, nil
	case db.CompRLE:
		return rleEncode(raw), nil
	case db.CompHuffman:
		return huffmanEncode(raw)
	case db.CompZstd:
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
func GenTagWalk(fields []db.FieldValue) []byte {
	var tmp []byte
	var varint []byte
	for _, field := range fields {
		tmp = append(tmp, db.ToBytes(field.Tag)...)
		tmp = append(tmp, db.ToBytes(field.CompFlags)...)
		if field.CompFlags&db.ArrayMask != 0 {
			tmp = append(tmp, writeVarUint(varint, uint64(len(field.Payload)))...)
			varint = varint[0:]
		}
		tmp = append(tmp, field.Payload...)

	}
	return tmp
}

// EncodeRecordTagWalk is a convenience wrapper that returns the TagWalk payload.
func EncodeRecordTagWalk(fields []db.FieldValue) ([]byte, error) {
	return GenTagWalk(fields), nil
}

// EncodeRecordHot builds a hot-vtable record (header + vtable for hot fields + payloadHot + tagwalk coldfields)
// This is a simplified port of pkg/dbflat Encoder.EncodeRecordHot but kept here so zc can be independently tested.
func EncodeRecordHot(schemaID uint64, hotTags []uint16, fields []db.FieldValue) ([]byte, error) {
	// validate hotTags
	for _, h := range hotTags {
		if h == 0 || h > 8 {
			return nil, fmt.Errorf("invalid hot field tag: %d", h)
		}
	}

	// Partition
	hot, cold := db.PartitionFields(fields, hotTags)

	// Generate hot payload and offsets
	payloadHot, hotOffsets := GenPayloads(hot)

	// vtable for hot
	vt := GeneVtables(hotOffsets)

	// tagwalk for cold
	tagwalk := GenTagWalk(cold)

	// Build header
	header := db.BuildHeader(vt, &db.LayoutPlan{HeaderFlags: db.FlagPadding | db.FlagModeHotVtable, SchemaID: schemaID, HotTags: hotTags})
	headerBytes := encodeHeader(nil, *header)

	// Join header + vtable + payloadHot + tagwalk
	out := make([]byte, 0, len(headerBytes)+len(vt)+len(payloadHot)+len(tagwalk))
	out = append(out, headerBytes...)
	out = append(out, vt...)
	out = append(out, payloadHot...)
	out = append(out, tagwalk...)

	return out, nil
}

// Helpers: GenPayloads and GeneVtables duplicated but using db types
type offsetLoc struct {
	tag      uint16
	compflag uint16
	offset   uint32
}

func GenPayloads(fields []db.FieldValue) ([]byte, []offsetLoc) {
	var tmp []byte
	next := 0
	var fieldbuf []byte
	var offmap []offsetLoc
	for _, field := range fields {
		if field.CompFlags&db.ArrayMask != 0 {
			fieldbuf = writeVarUint(fieldbuf, uint64(len(field.Payload)))
			tmp = append(tmp, fieldbuf...)
			fieldbuf = fieldbuf[:0]
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
