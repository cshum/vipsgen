# Introspection metadata

`internal/introspection` reads libvips through GObject introspection and hands the
generator a model of every operation argument: type, required/optional, enum and flags
membership, `GParamSpec` default, and numeric bounds together with whether each bound
is declared at all.

## What the tests guarantee

`operation_test.go` asserts only what is true of *any* libvips version:

- no non-finite bound reaches the metadata — a single `±Inf` used to abort the whole
  `-debug` dump, which is why bounds are reported as finite-or-absent — and every
  operation still marshals for that dump;
- a flags argument reports a non-zero default that is itself a member of the flags
  class, and `IsFlags` distinguishes a flags class from an enum;
- a 64-bit argument carries a range;
- the argument classes the option-presence rule needs keep their pspec defaults.

Per-version facts — which arguments have an unbounded range, the value of a default,
the bits in a mask — are deliberately *not* pinned in tests: they belong to the libvips
build, and pinning them turns a version bump into a test failure instead of a
reviewable diff. When the v2 options rule is implemented, its classifier and a
version-keyed metadata golden will live here.

## Bumping the libvips version

The matrix in `.github/workflows/ci.yml` is the single source of truth. A version
change is one matrix entry: no step names a version, because each generated package is
uploaded as an artifact named after its output directory (`vips`, `vips816`,
`vips817`) and the commit jobs restore them with one `vips*` pattern download.

1. Change the matrix entry (`vips_version` / `output_dir`) in a PR.
2. CI builds that libvips from source, runs `go test ./internal/...`, regenerates the
   package, and — for a same-repo PR — the `commit` job pushes
   `build: update generated vips packages` when the regenerated bindings differ. Treat
   that commit as the version change, not as noise.
3. If you also upgrade libvips locally, **clear the Go build cache**: cgo caches the
   pkg-config output, so builds and tests otherwise fail with
   `ld: library 'vips' not found` naming the *deleted* install path, while
   `pkg-config --modversion vips` already reports the new one.

   ```bash
   go clean -cache
   pkg-config --modversion vips   # must report the version you just installed
   ```

4. Read the regenerated bindings for what the presence rule will care about: a default
   moving across zero, a bound becoming finite, a new flags bit (`VIPS_FOREIGN_KEEP_ALL`
   is 31 before gainmap support and 63 from 8.18), an argument added or removed. Those
   are the changes that flip a field between pointer and value, which is why the rule
   has to be derived at generation time rather than pinned in vipsgen.
