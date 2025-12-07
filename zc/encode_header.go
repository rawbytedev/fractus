package zc

import (
	"encoding/binary"
)

const (
	MagicV1   = 0x44424633 // "DBF3"
	VersionV1 = 1

	// compFlags & 0x7F == compressor ID
	CompressionMask = 0x000F
	CompRaw         = 0x0000
	CompRLE         = 0x0001
	CompHuffman     = 0x0002
	CompLZ4         = 0x0003
	CompZstd        = 0x0004
	ArrayMask       = 0x8000 // MSB signals variable-length
	HeaderSize      = 40
	SlotSize        = 8
)

type FieldType int

const (
	TypeBool FieldType = iota
	TypeInt8
	TypeUint8
	TypeInt16
	TypeUint16
	TypeInt32
	TypeUint32
	TypeInt64
	TypeUint64
	TypeFloat32
	TypeFloat64
	TypeString
	TypeBytes
)
const (
	FlagPadding       = 0x0001 // padded layout 01
	FlagNoSchemaID    = 0x0002 // schema ID 10
	FlagModeHotVtable = 0x0004 // 100 1111
	FlagModeNoVtable  = 0x0008 // 1000
	FlagModeTagWalk   = 0x0010 //
	FlagModeDirect    = 0x0000 // means starts at 1 and next tag is plus 1
	// Extend for checksum, compression, etc.
)

// VTableSlot is 8B
type VTableSlot struct {
	Tag        uint16 // 2B
	DataOffset uint16 // 2B
	CompFlags  uint32 // 4B
}

type FieldValue struct {
	Tag       uint16 // 2B
	CompFlags uint16 // 2B
	Payload   []byte // raw or already-compressed bytes
}
type Header struct {
	Magic       uint32   // 4B
	Version     uint16   // 2B
	Flags       uint16   // 2B
	SchemaID    uint64   // 8B
	HotBitmap   byte     // 1B: presence map for tags 1–8
	VTableSlots byte     // number of slot in VTable
	DataOffset  uint16   // offset to start of Data section (from header start) 2B
	VTableOff   uint32   // offset to start VTable (from header start) 4B
	_           [16]byte // reserved for upgrade
}

// encodeHeaderFixed builds a header into a fixed-size slice to avoid
// intermediate allocations.
func encodeHeaderFixed(h Header) []byte {
	if h.Flags&FlagNoSchemaID != 0 {
		buf := make([]byte, HeaderSize-8)
		binary.LittleEndian.PutUint32(buf[0:], h.Magic)
		binary.LittleEndian.PutUint16(buf[4:], h.Version)
		binary.LittleEndian.PutUint16(buf[6:], h.Flags)
		buf[8] = h.HotBitmap
		buf[9] = h.VTableSlots
		binary.LittleEndian.PutUint16(buf[10:], h.DataOffset)
		binary.LittleEndian.PutUint32(buf[12:], h.VTableOff)
		return buf
	}
	buf := make([]byte, HeaderSize)
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
