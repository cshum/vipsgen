# Introspection metadata

`internal/introspection` reads libvips through GObject introspection and hands the
generator a model of every operation argument: type, required/optional, enum and flags
membership, `GParamSpec` default, and numeric bounds with their presence.

The tests assert only what holds for *any* libvips version: no bound is ever non-finite,
every operation marshals for the `-debug` dump, a flags default is non-zero and a member
of its own flags class, a 64-bit argument carries a range. Per-version values are not
pinned in tests — they belong to the libvips build, and pinning them turns a version
bump into a failure instead of a diff.

## Bumping the libvips version

The matrix in `.github/workflows/ci.yml` is the single source of truth. A version change
is one matrix entry: each generated package is uploaded as an artifact named after its
output directory (`vips`, `vips816`, `vips817`) and the commit jobs restore them with one
`vips*` pattern download, so no step names a version.

- **Same-repo PRs**: CI regenerates and the `commit` job pushes
  `build: update generated vips packages` when the bindings differ. That commit is the
  version change to review; an absent commit means the version changed nothing visible.
- **Locally, after upgrading libvips**: run `go clean -cache` first. cgo caches the
  pkg-config output, so builds and tests otherwise fail with
  `ld: library 'vips' not found` naming the *deleted* install path, while
  `pkg-config --modversion vips` already reports the new one.
- **In the diff**, watch for what flips a field between pointer and value: a default
  crossing zero, a bound becoming finite, a new flags bit (`VIPS_FOREIGN_KEEP_ALL` is 31
  before gainmap support and 63 from 8.18), an argument added or removed.
