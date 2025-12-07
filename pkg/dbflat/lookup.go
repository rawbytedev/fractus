package dbflat

func GetFixedWidthMap(u bool, fmap map[uint16]int) map[uint16]int {
	if u {
		return fmap
	}
	return nil
}

// FixedHotfields Lookup table
func HotFieldWidth(c uint16) int {
	f := map[uint16]int{
		1: 4,
		2: 4,
		4: 8,
	}
	return f[c]
}

// fixedWidth maps compFlags→byte length for primitives
func fixedWidth(c uint16) int {
	switch {
	case c <= 15: // 9-15
		return 1 // bool
	case c <= 31: // 16-31
		return 1 // int8
	case c <= 63: // 32-63
		return 1 // uint8
	case c <= 127: // 64-127
		return 2 // int16
	case c <= 191: // 128-191
		return 2 // uint16
	case c <= 255: // 192-255
		return 4 // int32
	case c <= 319: // 256-319
		return 4 // uint32
	case c <= 383: // 320-383
		return 8 // int64
	case c <= 447: // 384-447
		return 8 // uint64
	case c <= 511: // 448-511
		return 4 // float32
	case c <= 575: // 512-575
		return 8 // float64
	default:
		return -1 // variable length or unknown
	}
}
