package main

import (
    "fmt"
    "github.com/rawbytedev/Fractus/zc"
)

// Example skeleton showing how a user would opt into the zero-copy package.
func main() {
    opts := zc.Options{UnsafeStrings: true, UnsafePrimitives: true, CheckAlignment: true}
    fmt.Printf("Created zero-copy options: %+v\n", opts)
    // TODO: call zc encoder/decoder once implemented and generated code is available.
}
