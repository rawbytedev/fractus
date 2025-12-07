package zc

// Package zc (zero-copy) contains experimental, opt-in zero-copy
// encoders/decoders and codegen hooks for Fractus. The intent is to
// provide a clearly-separated package for implementations that rely on
// predefined schema + unsafe lifetime assumptions.

// NOTE: This package is a skeleton created to host zero-copy helpers.
// The actual implementations from the `dev` branch should be moved
// here in a follow-up patch. Keep the runtime-safe `fractus` API
// unchanged; zero-copy code must be opt-in and well-documented.

// Options contains runtime flags controlling zero-copy behaviour.
type Options struct {
    // UnsafeStrings allows converting []byte -> string without copy.
    UnsafeStrings bool

    // UnsafePrimitives allows aliasing primitive slices (e.g. []uint32)
    // into []byte without copying when alignment rules are satisfied.
    UnsafePrimitives bool

    // CheckAlignment enables runtime alignment checks before performing
    // zero-copy aliasing.
    CheckAlignment bool
}

// Encoder/Decoder entrypoints will be added here. For now this file
// contains the minimal types and documentation to help move code into
// the package in subsequent commits.
