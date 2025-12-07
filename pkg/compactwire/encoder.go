package compactwire

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
)

// EncodeDataFrame serializes a payload with optional offset table.
func (d *DataFrame) EncodeDataFrame(payload []byte, flags byte, offsets []uint32) ([]byte, error) {
	d.buf = &bytes.Buffer{}
	writePreamble(d.buf, TypeData)

	// reserve length + flags
	binary.Write(d.buf, binary.LittleEndian, uint32(0)) // length placeholder
	d.buf.WriteByte(flags)

	// NEW: write offset count + entries
	if flags&FlagHasOffsetTable != 0 {
		binary.Write(d.buf, binary.LittleEndian, uint16(len(offsets)))
		for _, off := range offsets {
			binary.Write(d.buf, binary.LittleEndian, off)
		}
	}

	// write the actual payload
	d.buf.Write(payload)

	// fill in length (includes everything up to + including CRC)
	out := d.buf.Bytes()
	total := uint32(len(out) + 4) // +4 for CRC
	binary.LittleEndian.PutUint32(out[3:], total)

	// compute & append CRC32 (over bytes 4 .. end-of-payload)
	crc := crc32.ChecksumIEEE(out[4:])
	out = append(out, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(out[len(out)-4:], crc)
	return out, nil
}

// EncodeErrorFrame builds an Error Frame with code and custom data.
func (e *ErrorFrame) EncodeErrorFrame(code byte, data []byte) ([]byte, error) {
	e.buf = &bytes.Buffer{}
	writePreamble(e.buf, TypeError)

	// TLV Length = code(1) + dataLen(2) + len(data)
	tlv := uint32(1 + 2 + len(data))
	binary.Write(e.buf, binary.LittleEndian, tlv)
	e.buf.WriteByte(code)
	binary.Write(e.buf, binary.LittleEndian, uint16(len(data)))
	e.buf.Write(data)

	// Append CRC
	out := e.buf.Bytes()
	crc := crc32.ChecksumIEEE(out[2:]) // exclude magic
	out = append(out, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(out[len(out)-4:], crc)
	return out, nil
}

func (y *HandshakeFrame) EncodeHandshake(h HandshakeFrame) ([]byte, error) {
	buf := &bytes.Buffer{}
	writePreamble(buf, TypeHandshake)

	// Reserve length
	binary.Write(buf, binary.LittleEndian, uint32(0))
	binary.Write(buf, binary.LittleEndian, h.VersionMask)
	binary.Write(buf, binary.LittleEndian, h.MTU)
	binary.Write(buf, binary.LittleEndian, h.TimeoutMS)
	binary.Write(buf, binary.LittleEndian, uint16(len(h.AlgCodes)))
	buf.Write(h.AlgCodes)

	// Set length field
	out := buf.Bytes()
	total := uint32(len(out) + 4)
	binary.LittleEndian.PutUint32(out[3:], total)

	// CRC over entire frame minus magic
	crc := crc32.ChecksumIEEE(out[2:])
	out = append(out, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(out[len(out)-4:], crc)
	return out, nil
}
