# Introspection metadata

`internal/introspection` reads libvips through GObject introspection and hands the
generator a model of every operation argument: type, required/optional, enum and flags
membership, `GParamSpec` default, and numeric bounds together with whether each bound
is declared at all.

The tests assert only what is true of *any* libvips version: a bound is never
non-finite, every operation marshals for the `-debug` dump, a flags default is non-zero
and a member of its own flags class, a plain enum is not mistaken for one, and a 64-bit
argument carries a range. Which arguments have an unbounded range, the value of a
default, the bits in a mask — none of that is pinned in tests, because it belongs to
the libvips build and pinning it turns a version bump into a failure instead of a
diff. When the v2 options rule is implemented, its classifier and a version-keyed
metadata golden go here.

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
