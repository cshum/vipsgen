# Introspection metadata

`internal/introspection` asks libvips about every operation and argument: type, required
or optional, enum and flags members, default value, numeric bounds.

## What the tests check

Only what holds for every libvips version: no bound is infinite, every operation
survives JSON for the `-debug` dump, a flags default is non-zero and is one of its own
flags, a 64-bit argument has a range.

Values that come from libvips (defaults, bounds, mask bits) are not pinned in tests, so
a libvips upgrade shows up as a diff instead of a failure.

## Upgrading libvips

The matrix in `.github/workflows/ci.yml` is the only place a version is named, so an
upgrade is one matrix entry. Each package is uploaded as an artifact named after its
output dir (`vips`, `vips816`, `vips817`), so one `vips*` download restores them all.

- On a same-repo PR, CI regenerates and the `commit` job pushes the result. If it pushes
  nothing, the upgrade changed no generated code.
- Locally, run `go clean -cache` first: cgo remembers the old pkg-config paths, so
  builds fail with `ld: library 'vips' not found` pointing at the deleted install.
- In the diff, look for a default crossing zero, a bound becoming finite, a new flags
  bit (`keep` is 31 before 8.18, 63 after), or an argument appearing or disappearing.
