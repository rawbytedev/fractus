package zc

import (
	"bytes"
	db "fractus/pkg/dbflat"
	"testing"
)

func makeTestFields(shape string) []db.FieldValue {
	switch shape {
	case "skinny":
		a, _ := db.Write(uint32(300))
		return []db.FieldValue{
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
	payload, err := EncodeRecordTagWalk(fields)
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
	out, err := EncodeRecordHot(schemaID, hotTags, fields)
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
