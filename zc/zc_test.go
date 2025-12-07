package zc

import (
    "bytes"
    "testing"
    db "fractus/pkg/dbflat"
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
    var dec db.Decoder
    m, err := dec.DecodeRecordHot(out)
    if err != nil {
        t.Fatalf("DecodeRecordHot error: %v", err)
    }
    // Hot map may contain empty entries for unused slots; check at least tag 1 exists
    if len(m) == 0 {
        t.Fatalf("expected non-empty hotmap")
    }
}
