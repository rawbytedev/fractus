package zc

import (
	"bytes"
	"encoding/binary"
	db "fractus/pkg/dbflat"
	"testing"
)

func makeTestFields(shape string) []FieldValue {
	switch shape {
	case "skinny":
		a, _ := db.Write(uint32(300))
		return []FieldValue{
			{Tag: uint16(1), Payload: []byte("Hello I'm Test 1"), CompFlags: 0x8000},
			{Tag: uint16(2), Payload: []byte("Hello I'm Test 2"), CompFlags: 0x0000 | 0x8000},
			{Tag: uint16(3), Payload: []byte("Hello I'm Test Comp+10"), CompFlags: 0x0000 | 0x8000},
			{Tag: uint16(192), Payload: a, CompFlags: 0x0000},
		}
	default:
		return nil
	}
}

func TestGenTagWalk_DecodeRoundTrip(t *testing.T) {
	fields := makeTestFields("skinny")
	zc := NewZeroCopy()
	payload, err := zc.EncodeRecordTagWalk(fields)
	if err != nil {
		t.Fatalf("EncodeRecordTagWalk error: %v", err)
	}
	var dec db.Decoder
	m, off, err := dec.DecodeRecordTagWalk(payload, 0, nil)
	if err != nil {
		t.Fatalf("DecodeRecordTagWalk error: %v", err)
	}
	if off == 0 {
		t.Fatalf("expected non-zero offset")
	}
	// ensure tags decoded present
	if !bytes.Equal(m[1], fields[0].Payload) {
		t.Fatalf("tag 1 payload mismatch")
	}
}

func TestEncodeRecordHot_Basic(t *testing.T) {
	fields := makeTestFields("skinny")
	hotTags := []uint16{1, 2}
	schemaID := uint64(112)
	zc := NewZeroCopy()
	out, err := zc.EncodeRecordHot(schemaID, hotTags, fields)
	if err != nil {
		t.Fatalf("EncodeRecordHot error: %v", err)
	}
	// Basic structural checks: header + vtable present and flags set
	hdr, err := db.ParseHeader(out)
	if err != nil {
		t.Fatalf("ParseHeader error: %v", err)
	}
	if hdr.VTableSlots == 0 {
		t.Fatalf("expected VTableSlots > 0")
	}
	if hdr.Flags&db.FlagModeHotVtable == 0 {
		t.Fatalf("expected FlagModeHotVtable to be set in header flags")
	}
}

func TestEncodeHot_CompressionRoundTrip(t *testing.T) {
	// Hot compressed field should round-trip (decoder decompresses)
	orig := []byte("This is some compressible data: hello hello hello hello")
	fields := []FieldValue{
		{Tag: 1, CompFlags: db.CompZstd, Payload: orig},
		{Tag: 9, CompFlags: 0x8000, Payload: []byte("cold field")},
	}
	fields2 := []db.FieldValue{
		{Tag: 1, CompFlags: db.CompZstd, Payload: orig},
		{Tag: 9, CompFlags: 0x8000, Payload: []byte("cold field")},
	}
	hot := []uint16{1}
	zc := NewZeroCopy()
	out, err := zc.EncodeRecordHot(0x1234, hot, fields)
	if err != nil {
		t.Fatalf("EncodeRecordHot error: %v", err)
	}
	var enc db.Encoder
	out2, err := enc.EncodeRecordHot(0x1234, hot, fields2)
	if !bytes.Equal(out, out2) {
		t.Log(out)
		t.Log("2Nd")
		t.Log(out2)
		t.Fatal("incorrect")
	}
	var dec db.Decoder
	m, err := dec.DecodeRecord(out, nil)
	if err != nil {
		t.Fatalf("DecodeRecord error: %v", err)
	}
	if !bytes.Equal(m[1], orig) {
		t.Fatalf("compressed hot field roundtrip mismatch: got %v", m[1])
	}
}

func TestTagWalk_ArrayRoundTrip(t *testing.T) {
	// array payload should be returned verbatim by TagWalk decoder
	// build payload with two uint32s
	payload := make([]byte, 8)
	binary.LittleEndian.PutUint32(payload[0:], 0xDEADBEEF)
	binary.LittleEndian.PutUint32(payload[4:], 0xCAFEBABE)
	fields := []FieldValue{{Tag: 1, CompFlags: db.ArrayMask, Payload: payload}}
	zc := NewZeroCopy()
	enc := zc.GenTagWalk(fields)
	var dec db.Decoder
	m, _, err := dec.DecodeRecordTagWalk(enc, 0, nil)
	if err != nil {
		t.Fatalf("DecodeRecordTagWalk error: %v", err)
	}
	if !bytes.Equal(m[1], payload) {
		t.Fatalf("tagwalk array payload mismatch")
	}
}
