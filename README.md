# Fractus

![Test and Benchmark](https://github.com/rawbytedev/Fractus/actions/workflows/test-and-bench.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/rawbytedev/Fractus)](https://goreportcard.com/report/github.com/rawbytedev/Fractus)

Fractus is a lightweight, serialization library for Go.
It encodes and decodes structs into a compact binary format focused on
low allocations and high throughput.

Supported types:

- Fixed-size primitives (`int8`, `int16`, `int32`, `int64`, `uint*`, `float32`, `float64`, `bool`)
- Variable-size types (`string`, `[]byte`, slices of primitives/strings)
- Struct pointers (encoded by value)
- Optional unsafe zero-copy modes for strings and primitive slices (opt-in)

The goal is fast, deterministic encoding/decoding with a small allocation
footprint. See `docs/FORMAT.md` for the wire-format specification.

---

## Features

- **Struct encoding/decoding**: Works with exported fields of Go structs.
- **Slices and strings**: Handles variable-length data with varint length prefixes.
- **Unsafe modes**: `SafeOptions` toggles zero-copy for strings and primitive slices.
- **Fuzz & property-based tests**: Ensures round-trip correctness.

---

## Installation

```bash
go get github.com/rawbytedev/fractus
```

---

## Usage

### Encode / Decode

```go
package main

import (
    "fmt"
    "github.com/rawbytedev/fractus"
)

type Example struct {
    Name   string
    Age    int32
    Scores []float64
}

func main() {
    f := fractus.NewFractus(fractus.SafeOptions{})

    val := Example{Name: "Alice", Age: 30, Scores: []float64{95.5, 88.0}}
    data, err := f.Encode(val)
    if err != nil {
        panic(err)
    }

    var out Example
    if err := f.Decode(data, &out); err != nil {
        panic(err)
    }

    fmt.Printf("Decoded: %+v\n", out)
}
```

---

## Benchmarks

Fractus aims to minimize allocations and improve throughput:

```bash
go test -bench=. -benchmem
```

With buffer reuse and optional unsafe modes, allocations can be reduced
significantly for hot-path encoders and decoders.

---

## Testing

Fractus includes fuzz and property-based tests:

```bash
go test ./...
```

- `FuzzEncodeDecode` ensures correctness across mixed types.
- `quick.Check` encoding/decoding for random structs.
- Error cases are tested (non-structs, unexported fields, wrong pointer types).

---

## Important

- **UnsafeStrings**: When enabled, decoded strings may alias the original buffer.
    Ensure the buffer outlives the string usage, or disable this option for safe copies.
- **Unexported fields**: Skipped during encoding.
- **Unsupported types**: Maps, interfaces, complex numbers, and nested slices
    (except `[]byte`) are not supported.

Benchmarks and format
---------------------
- Benchmarks are available in `fractus_bench_test.go` (run with `go test -bench=. -benchmem`).
- Wire-format is documented in `docs/FORMAT.md`.

---

## License

MIT License. See [LICENSE](LICENSE) for details.
