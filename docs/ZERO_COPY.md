# Fractus Zero-Copy Integration Plan

This document outlines the recommended approach for integrating the experimental
zero-copy implementations from the `dev` branch into the main Fractus repository.

Goals
- Preserve the default, safe `fractus` package behaviour for existing users.
- Provide an explicit, opt-in zero-copy package that contains different
  encoding modes (TagWalk, HotVtable, FullMode codegen) and any codegen tools.
- Make zero-copy features discoverable, well-documented, and tested separately.

High-level approach
1. Create a subpackage `fractus/zc` to host zero-copy helpers, runtime modes,
   and codegen runtime types. This repository already contains a `dev` branch
   with multiple modes; move the implementations into this package.
2. Treat `FullMode` as a codegen-first target. Provide a small generator in
   `cmd/fractus-gen` that emits strongly-typed encoders/decoders for predefined
   schemas. Generated code lives in the user's package (or an examples folder).
3. Keep `TagWalk` and `HotVtable` available as runtime zero-copy modes in
   `fractus/zc` (these are simpler to make safe at runtime with alignment checks).

Packaging & safety
- Make zero-copy behaviour opt-in via runtime options (`zc.Options`) and
  strongly recommend build tags (e.g., `//go:build fractus_zero_copy`) for
  production deployments that enable aggressive optimizations.
- Add `docs/ZERO_COPY.md` (this file), a `docs/SAFE_USAGE.md` checklist, and
  examples demonstrating `SafeDecoder` usage.

Testing & CI
- Add CI jobs that run tests for both the safe codebase and the zero-copy
  package (with any build tags required).
- Add benchmark jobs comparing default Fractus, `fractus/zc` runtime modes,
  and generated FullMode code.

Next steps
1. Move zero-copy helper funcs (alignment checks, unsafe conversions, setUnsafeFixed)
   into `fractus/zc`.
2. Create `zc` public API wrappers and small examples in `examples/`.
3. Add a `cmd/fractus-gen` scaffold to generate FullMode code.
4. Update README and `docs/USAGE.md` with clear migration docs.
