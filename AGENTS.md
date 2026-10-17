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

## Tests

- CI runs three libvips versions; each leg regenerates its own package before testing.
- Tests under `internal/` are version-independent by design. One that imports `v2/vips`
  drags the 8.18-generated package into the 8.16 and 8.17 legs, which fails on headers the
  older libvips does not have. Pins on version-specific behaviour belong inside `vips/`.
- The generated tests count open images, so close what you open.

## Docs

- README and MIGRATION use plain words and state behaviour, not history. Generated field
  comments carry the range and the default, which is where a reader looks first.
- Numbers in docs come from the artefact — the classification table, the probe output — not
  from arithmetic in prose, and not from a previous draft.
