# Working on vipsgen

Notes for agents and contributors, short on purpose. The README is for users; this is for
anyone changing the generator.

## Regenerating

- `vips/`, `vips817/` and `vips816/` are produced by CI from the matrix in
  `.github/workflows/ci.yml`. Do not regenerate them locally: the libvips you have is not
  the one they were generated from, so a local run produces spurious diffs. Change
  `internal/` or the templates, push, and CI commits the result.
- `vipsgen -debug` writes into the repo, including regenerated packages. If you only wanted
  the JSON dump, restore the tree afterwards.
- Commit explicit paths. `git add -A` sweeps generated trees and scratch programs into the
  branch.

## The options rule

- A field is a pointer when libvips accepts its zero value **and** that zero differs from the
  option's default. `nil` means "not supplied". The classification is derived from
  GParamSpec, so it tracks the libvips you build against — the three packages are not
  interchangeable.
- `internal/generator/presence.go` holds the rule and one measured exception list. The list
  cannot be derived: `gifload.n` and `webpload.n` have the same range and the same default,
  and only the first rejects a zero.
- `vips/zero_pins_test.go` pins the classification; `internal/generator/presence_test.go`
  pins the policy, including that the policy reaches `OptionalInputs` — that slice holds
  copies, so flagging `Arguments` alone silently does nothing.

## CI

- Three legs, one per libvips version, each regenerating its own package before it tests.
- Examples are built only on the `vips` (8.18) leg, so a compile error an older libvips
  would hit does not show there — and one the newer one hits shows nowhere else.
- When a change alters generated output, the last CI job commits it. That job's own run can
  park at `action_required`, which is neither a pass nor a failure: the parent run already
  built the content it committed.
- Same libvips in, same bytes out — regeneration is expected to be byte-identical, and a
  change that alters the output should be explainable operation by operation.

## Measuring a zero exception

The list in `internal/generator/presence.go` is measured, so re-measure before adding to it.
Call the operation twice — once with an explicit zero, once without — and compare outcomes;
an error in the first and not the second is attributable to the zero rather than to the
input. Two traps:

- The `vips` CLI swallows an empty string, so an empty `--id` proves nothing. Measure
  strings through the Go API, where `vips.Ptr("")` reaches libvips.
- A single-page input can give the wrong verdict for `n`, which is about pages: a documented
  rejection only reproduces with a multi-page file. That is why `heifload.n` and `pdfload.n`
  were missed on the first pass.

## The debug dump

`vipsgen -debug` writes `debug_operations.json`: operations with `RequiredInputs`,
`OptionalInputs` and their types. Note that its `GoName` is lowercase (`gifload`, `n`) while
the generated code is not (`GifloadOptions`, `N`), which is easy to get wrong in a script.

## Tests

- Tests under `internal/` are version-independent by design. One that imports `v2/vips`
  drags the 8.18-generated package into the 8.16 and 8.17 legs, which fails on headers the
  older libvips does not have. Pins on version-specific behaviour belong inside `vips/`.
- The generated tests count open images, so close what you open.

## Docs

- README and MIGRATION use plain words and state behaviour, not history. Generated field
  comments carry the range and the default, which is where a reader looks first.
- Numbers in docs come from the artefact — the classification table, the probe output — not
  from arithmetic in prose, and not from a previous draft.

## Releasing

Tags are manual and they are the release: the module path carries the major version, so
`v2.0.0` goes on this line and v1 keeps working at its existing paths.

## Elsewhere

README for users, MIGRATION for v1 to v2, `internal/introspection/README.md` for the
checklist to run when bumping libvips.
