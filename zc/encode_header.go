package zc

import (
    "encoding/binary"
    db "fractus/pkg/dbflat"
)

// encodeHeaderFixed builds a header into a fixed-size slice to avoid
// intermediate allocations.
func encodeHeaderFixed(h db.Header) []byte {
    if h.Flags&db.FlagNoSchemaID != 0 {
        buf := make([]byte, db.HeaderSize-8)
        binary.LittleEndian.PutUint32(buf[0:], h.Magic)
        binary.LittleEndian.PutUint16(buf[4:], h.Version)
        binary.LittleEndian.PutUint16(buf[6:], h.Flags)
        buf[8] = h.HotBitmap
        buf[9] = h.VTableSlots
        binary.LittleEndian.PutUint16(buf[10:], h.DataOffset)
        binary.LittleEndian.PutUint32(buf[12:], h.VTableOff)
        return buf
    }
    buf := make([]byte, db.HeaderSize)
    binary.LittleEndian.PutUint32(buf[0:], h.Magic)
    binary.LittleEndian.PutUint16(buf[4:], h.Version)
    binary.LittleEndian.PutUint16(buf[6:], h.Flags)
    binary.LittleEndian.PutUint64(buf[8:], h.SchemaID)
    buf[16] = h.HotBitmap
    buf[17] = h.VTableSlots
    binary.LittleEndian.PutUint16(buf[18:], h.DataOffset)
    binary.LittleEndian.PutUint32(buf[20:], h.VTableOff)
    return buf
}
