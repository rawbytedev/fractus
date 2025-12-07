# Fractus Wire Format

This file documents the on-disk / on-the-wire binary layout produced by `Fractus`.

Overview
--------
Fractus encodes a struct into a compact, deterministic byte sequence. The layout is
simple and designed for fast encoding/decoding with low allocation overhead.

Top-level layout
----------------
All multi-field encodings follow this high-level layout:

1. VarInt: Field count N (number of exported fields present in the struct plan)
2. Body: Fixed-size fields and variable-length fields concatenated in declaration order.

Notes about fields
------------------
- Fixed-size primitives (bool, int8/int16/int32/int64, uint*, float32, float64)
  are written inline, in little-endian byte order, with no padding between
  fixed fields. The encoder reuses a small 8-byte scratch buffer to avoid
  allocating for these writes.

- Variable-length fields (strings, slices) are encoded in the body. Each
  variable element written to the body is prefixed by a VarInt length and
  then the payload bytes.

- For slices of fixed-size primitives, Fractus attempts a zero-copy write
  by appending the backing memory of the slice directly. This is only done
  when `SafeOptions.UnsafePrimitives` is enabled and the slice is properly
  aligned (or alignment checks are disabled). Otherwise the slice is encoded
  element-by-element.

Varint encoding
---------------
Fractus uses an LEB128-like unsigned varint for compact lengths and counters.
Each byte uses the low 7 bits for payload and the high bit as a continuation
marker.

Unsafe / zero-copy modes
------------------------
- `UnsafeStrings`: When enabled, decoded strings may alias the original input
  buffer. The caller must ensure the input byte slice remains valid (in
  scope and not modified) for the lifetime of any string values created from
  it. If this cannot be guaranteed, disable `UnsafeStrings` and Fractus will
  make safe copies.

- `UnsafePrimitives`: When enabled, Fractus may perform zero-copy conversions
  of `[]T` to `[]byte` for primitive types. This requires alignment of the
  slice memory for the element type and is platform-dependent; Fractus
  performs an alignment check by default and falls back to safe copying if
  alignment is not satisfied.

Compatibility and evolution
---------------------------
The current format is intentionally minimal and stable for a fixed struct
layout. There is no built-in versioning or schema evolution mechanism yet.
If you need forward/backward compatibility across different struct layouts
consider adding an explicit version field to your struct and handling
compatibility in your application.

Examples
--------
For a struct with fields: (Int32, Str, []int16)

- Encoder writes: VarInt(3) then writes fixed Int32 (4 bytes), then writes
  VarInt(len(Str)) + Str bytes, then VarInt(len(slice)) + slice elements
  (each 2 bytes little-endian).

This layout keeps decoding fast — fixed fields are read in order and variable
fields are parsed by reading their length prefixes.

For more examples, see the unit tests and benchmark files.
