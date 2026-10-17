# Upgrading libvips

`internal/introspection` reads libvips' own argument metadata, so the generated bindings
follow whichever libvips the matrix in `.github/workflows/ci.yml` builds. That matrix is
the only place a version is named: an upgrade is one matrix entry, because each package
is uploaded as an artifact named after its output dir (`vips`, `vips816`, `vips817`) and
one `vips*` download restores them all.

- On a same-repo PR, CI regenerates and the `commit` job pushes the result. Nothing
  pushed means the upgrade changed no generated code.
- Locally, run `go clean -cache` first: cgo remembers the old pkg-config paths, so builds
  fail with `ld: library 'vips' not found` pointing at the deleted install.
- In the diff, look for what changes an option's presence: a default crossing zero, a
  bound becoming finite, a new flags bit (`keep` is 31 before 8.18, 63 after), an
  argument appearing or disappearing.

What the tests do and deliberately do not assert is stated in `operation_test.go`, next
to the assertions themselves.
