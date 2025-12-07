package dbflat

import (
	"encoding/binary"
	"errors"
	//"gopkg.in/yaml.v2"
)

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

// Parse Header from buf; zero copy
func ParseHeader(buf []byte) (Header, error) {
	if len(buf) < HeaderSize/2 {
		return Header{}, errors.New("buffer too short for header")
	}
	h := Header{}
	h.Magic = binary.LittleEndian.Uint32(buf[0:])
	h.Version = binary.BigEndian.Uint16(buf[4:])
	h.Flags = binary.LittleEndian.Uint16(buf[6:])
	if h.Flags&FlagNoSchemaID != 0 {
		h.HotBitmap = buf[8]
		h.VTableSlots = buf[9]
		h.DataOffset = binary.LittleEndian.Uint16(buf[10:])
		h.VTableOff = binary.LittleEndian.Uint32(buf[12:])
	} else {
		h.SchemaID = binary.LittleEndian.Uint64(buf[8:])
		h.HotBitmap = buf[16]
		h.VTableSlots = buf[17]
		h.DataOffset = binary.LittleEndian.Uint16(buf[18:])
		h.VTableOff = binary.LittleEndian.Uint32(buf[20:])
	}
	return h, nil
}
