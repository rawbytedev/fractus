package compactwire

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
)

// DecodeDataFrame parses a Data Frame and returns payload and offsets.
func (d *DataFrame) DecodeDataFrame(data []byte) ([]byte, []uint32, byte, error) {
	d.rdr = bytes.NewReader(data)
	// 1) preamble
	t, err := readPreamble(d.rdr)
	if err != nil || t != TypeData {
		return nil, nil, 0, errors.New("not a data frame")
	}

	// 2) total length + flags
	var length uint32
	binary.Read(d.rdr, binary.LittleEndian, &length)
	flags, _ := d.rdr.ReadByte()

	// optional: sanity‐check length
	// // length covers from byte[3] through CRC
	if int(length) != len(data) { // -2 for the 2-byte magic
		return nil, nil, 0, errors.New("length mismatch")
	}
	// 3) offset table
	var offsets []uint32
	if flags&FlagHasOffsetTable != 0 {
		// read count
		var cnt uint16
		binary.Read(d.rdr, binary.LittleEndian, &cnt)
		offsets = make([]uint32, cnt)
		for i := 0; i < int(cnt); i++ {
			binary.Read(d.rdr, binary.LittleEndian, &offsets[i])
		}
	}

	// 4) payload = everything up to the final 4-byte CRC
	// current rdr offset = payloadStart
	payloadStart := len(data) - d.rdr.Len()
	payloadEnd := len(data) - 4
	payload := data[payloadStart:payloadEnd]

	// 5) CRC check
	want := binary.LittleEndian.Uint32(data[len(data)-4:])
	if crc32.ChecksumIEEE(data[4:payloadEnd]) != want {
		return nil, nil, 0, errors.New("crc mismatch")
	}

	return payload, offsets, flags, nil
}

// DecodeErrorFrame parses an Error Frame and returns code and data.
func (e *ErrorFrame) DecodeErrorFrame(data []byte) (byte, []byte, error) {
	e.rdr = bytes.NewReader(data)
	t, err := readPreamble(e.rdr)
	if err != nil || t != TypeError {
		return 0, nil, errors.New("not an error frame")
	}

	var tlv uint32
	binary.Read(e.rdr, binary.LittleEndian, &tlv)
	code, _ := e.rdr.ReadByte()
	var dataLen uint16
	binary.Read(e.rdr, binary.LittleEndian, &dataLen)

	custom := make([]byte, dataLen)
	e.rdr.Read(custom)

	// Verify CRC over entire frame minus magic
	body := data[2 : len(data)-4]
	want := binary.LittleEndian.Uint32(data[len(data)-4:])
	if crc32.ChecksumIEEE(body) != want {
		return 0, nil, errors.New("crc mismatch")
	}
	return code, custom, nil
}

func (y *HandshakeFrame) DecodeHandshake(data []byte) (HandshakeFrame, error) {
	var h HandshakeFrame
	buf := bytes.NewReader(data)
	t, err := readPreamble(buf)
	if err != nil || t != TypeHandshake {
		return h, errors.New("not handshake")
	}

	var length uint32
	binary.Read(buf, binary.LittleEndian, &length)
	binary.Read(buf, binary.LittleEndian, &h.VersionMask)
	binary.Read(buf, binary.LittleEndian, &h.MTU)
	binary.Read(buf, binary.LittleEndian, &h.TimeoutMS)
	var algoLen uint16
	binary.Read(buf, binary.LittleEndian, &algoLen)
	// Remaining bytes minus CRC are alg codes
	algoC := make([]byte, algoLen)
	io.ReadFull(buf, algoC)
	h.AlgCodes = algoC

	// CRC validation
	body := data[2 : len(data)-4]
	want := binary.LittleEndian.Uint32(data[len(data)-4:])
	if crc32.ChecksumIEEE(body) != want {
		return h, errors.New("crc mismatch")
	}
	return h, nil
}
