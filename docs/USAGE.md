# Usage and safety notes

This document gives practical examples for using `Fractus` and the `SafeDecoder`
wrapper, and highlights caveats when using unsafe zero-copy modes.

Basic encode/decode
-------------------
```go
package main

import (
    "fmt"
    "github.com/rawbytedev/fractus"
)

type Example struct {
    Name  string
    Scores []int16
}

func main() {
    f := fractus.NewFractus(fractus.SafeOptions{})
    v := Example{Name: "Alice", Scores: []int16{10,20,30}}
    data, err := f.Encode(v)
    if err != nil { panic(err) }

    var out Example
    if err := f.Decode(data, &out); err != nil { panic(err) }
    fmt.Println(out)
}
```

SafeDecoder example (keep payload alive)
---------------------------------------
When `UnsafeStrings` or `UnsafePrimitives` are enabled, decoded values may
alias the original input buffer. If you need the decoded values to remain
valid beyond the lifetime of the input byte slice variable, wrap decoding
with `SafeDecoder` which keeps a reference to the payload:

```go
f := fractus.NewFractus(fractus.SafeOptions{UnsafeStrings: true})
data, _ := f.Encode(MyStruct{S: "hello"})
sd := fractus.NewSafeDecoder(f)
var out MyStruct
sd.Decode(data, &out)
// sd.payload holds onto data ensuring `out.S` remains valid.
```

Unsafe-mode caveats and best practices
-------------------------------------
- `UnsafeStrings`: Decoded strings created via `unsafe.String` will point into
  the provided input slice. If that slice is modified or freed (goes out of
  scope and is GC'd), the string becomes invalid. Use `SafeDecoder` when you
  need the decoded values to outlive the original slice.

- `UnsafePrimitives`: For primitive `[]T` slices, Fractus can avoid copying
  element bytes by referencing the slice backing memory. This requires proper
  alignment and is enabled only when `SafeOptions.UnsafePrimitives` is set.
  If alignment checks fail, Fractus falls back to safe element-by-element
  encoding/decoding.

- Concurrency: `Fractus` caches per-type metadata safely, but individual
  `Fractus` instances reuse internal buffers (e.g. `scratch`) and are not
  safe to share for simultaneous `Encode`/`Decode` calls. Use separate
  instances per goroutine or add external synchronization.

When to prefer safe mode
------------------------
- If you cannot guarantee the lifetime of the input buffer → disable
  `UnsafeStrings` and `UnsafePrimitives`.
- If you need maximum performance and can ensure buffer lifetimes and
  memory alignment → enable unsafe modes and consider `SafeDecoder` where
  appropriate.
