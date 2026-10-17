# v2 Options API Design

## Context

vipsgen currently generates options as ordinary Go value fields and decides whether to pass an option by inspecting its value. Zero-like values are therefore often treated as omitted:

- `0` for integers and enums
- `0.0` for floating-point values
- `false` for booleans
- `""` for strings
- empty collections in some cases

This is incorrect whenever libvips distinguishes omission from an explicitly supplied zero value. For example, `VIPS_INTENT_PERCEPTUAL` has the value `0`, while omitting `intent` allows an operation such as thumbnailing to use its own default.

The issue affects more than enums. The same ambiguity exists across integer, float, boolean, and other optional scalar inputs.

Measured on libvips 8.18.7 (golden: `internal/introspection/testdata/metadata-8.18.7.txt`): 327 operations are discovered, 219 of them have optional inputs, 1202 optional inputs in total. 216 of those classify as pointer fields under the rule below (121 numeric, 48 flags, 35 enum, 12 string) across 120 operations; 16 are handled by the manual allowlist today and 48 `keep` fields have the `-1` workaround, so 152 currently fall back to the libvips default without saying so. The dump behind those numbers, the classifier and the re-runnable probes are in `~/Developer/vipsgen-v2/`.

The current selective approach, such as maintaining a list of integer options that are allowed to pass zero, cannot reliably cover current or future libvips operations. The v2 API should make presence explicit where Go zero values are ambiguous, while keeping idiomatic representations for strings, slices, images, and objects.

## Goals

The v2 options API should:

1. Preserve the distinction between omitted and explicitly supplied values.
2. Work consistently for optional scalar types whose zero values may be meaningful.
3. Avoid manually maintained lists of zero-sensitive fields.
4. Keep omission semantics controlled by the Go API rather than guessed in C.
5. Remain understandable in generated code and ordinary Go usage.
6. Use metadata to support repeatable classification without pretending it contains operation-specific semantics.
7. Permit breaking changes since v2 is the compatibility boundary.

## Proposed contract

Shown on `pngsave`, the rule produces a mix of pointer and value fields:

```go
type PngsaveOptions struct {
    Compression *int     // default 6, range [0,9]
    Q           *int     // default 100, range [0,100]
    Dither      *float64 // default 1, range [0,1]
    Keep        *Keep    // flags, default all, 0 is a member

    Bitdepth  int    // default 8, range [1,16], 0 is out of range
    Interlace bool   // default false
    Profile   string // default empty
}
```

For pointer-valued fields:

> `nil` means omitted. A non-`nil` pointer means explicitly supplied, regardless of its value.

Examples:

```go
// Let libvips choose all defaults.
opts := PngsaveOptions{}

// Explicitly request compression level 0 and quality 0.
opts := PngsaveOptions{
    Compression: vips.Ptr(0),
    Q:           vips.Ptr(0),
}

// Explicit zero-like values are preserved.
opts := PngsaveOptions{
    Dither: vips.Ptr(0.0),
    Keep:   vips.Ptr(vips.KeepNone), // flags: 0 is a member, default is all
}
```

Options whose zero value cannot mean anything to libvips stay ordinary values: `Bitdepth` has range `[1,16]`, so a `0` would be refused, and omitting `Interlace` or `Profile` already means `false` and `""`.

The pointer decision is mixed rather than universal, but it is one rule:

> A field needs a pointer when its type's zero value is a value libvips would accept **and** it differs from the default libvips records.

- Integers and floating-point values: pointer when `0` is within the allowed range and the default is non-zero.
- Enums and flags: pointer when `0` is a member of the enum or the flags class and the default is not `0`.
- Strings: pointer when the default is non-empty, so that an empty string can be a distinct value.
- Booleans: pointer only when the default is `true`; none of the 337 optional booleans in libvips 8.18.2 has that default, so booleans stay ordinary values.
- Optional slices remain slices; see the collection policy below.
- Images and object-like values retain their natural pointer semantics.

Everything else stays an ordinary Go value, because for those fields omission and an explicitly supplied zero value are the same request.

Flags are not a separate policy: `0` is looked up in the flags class exactly as it is in an enum. That means `keep` (default all, `VIPS_FOREIGN_KEEP_NONE` is `0`) needs a pointer, while `pngsave.filter`'s flags class has no zero member and stays a value. Because the Go `Keep` constants must then be C-faithful, this also removes the `KeepNone = -1` remap and the `vipsgen_set_keep` special case.

The classification is total: since flags, 64-bit and bound presence are all in the metadata, no optional input is left unclassified and there is no "conservative" fallback.

The package should provide a generic helper:

```go
func Ptr[T any](value T) *T {
    return &value
}
```

For callers who prefer local variables, ordinary Go pointers remain usable:

```go
intent := vips.IntentPerceptual
opts := ThumbnailOptions{Intent: &intent}
```

## Default behavior

The zero value of every generated options struct should mean that no optional values were supplied:

```go
opts := ThumbnailOptions{}
```

Libvips should remain responsible for operation defaults. Generated Go code should not eagerly populate fields with defaults, because doing so changes omission into explicit configuration and can duplicate version-dependent libvips behavior.

If `Default<Operation>Options` is retained, it should return an options value with fields left `nil`, or it should be removed in favor of the options struct zero value. Hand-written facades are the exception: `DefaultLoadOptions` sets `FailOnError: true` as a vipsgen policy, not a libvips default, and must keep doing so. Documentation should describe libvips defaults without encoding them as present values.

## Generated wrapper behavior

Presence should be checked in the generated Go wrapper for pointer-valued fields. A conversion local is needed either way, because Go `int` is 64-bit while `C.int`, `gboolean` and the enum types are not:

```go
var cIntent C.VipsIntent
intentSet := 0
if options.Intent != nil {
    cIntent = C.VipsIntent(*options.Intent)
    intentSet = 1
}
```

The generated C wrapper receives presence as an explicit flag alongside the value. This is preferred over passing a pointer for the value: the Go side needs the local above regardless, and a flag also covers arrays and strings, which a value pointer cannot. The C parameter list grows by one `int <arg>_set` per pointer field; the setters themselves become exact:

```c
if (intent_set)
    vips_object_set(VIPS_OBJECT(operation), "intent", intent, NULL);
```

C setter helpers should not skip values based on whether they are zero-like. They should set the value they receive whenever the Go wrapper reports presence. Omission is already known by the Go wrapper from pointer or slice presence, or by the existing value policy for non-pointer fields.

This removes the need for:

- `shouldIntAllowZero`
- manually maintained zero-sensitive allowlists
- type-specific zero-value heuristics
- inferring user intent from values in C

## Collections and pointer-like values

Optional arrays are represented by ordinary Go slices, not pointers to slices:

- `nil` means omit the optional parameter.
- A non-`nil` empty slice means explicitly supply an empty array.
- A non-empty slice means explicitly supply those values.

The generated setter must check `slice != nil`, not `len(slice) > 0`. Two changes follow from that, and neither is in place today: the Go-side array conversion returns `nil` for a zero-length slice, so `nil` and an empty slice are indistinguishable before the C call, and the C wrapper only builds a `VipsArray*` when the length is positive. Array options therefore need presence carried across the Go/C boundary as well. The metadata only needs to identify the array kind and element type so the generator can construct the appropriate libvips boxed array. vipsgen does not need to infer operation-specific meaning for an empty array; libvips validates the supplied value.

Optional object and image values retain natural pointer semantics where `nil` means omission.

## Introspection metadata

Metadata supports generation and conservative classification, but it does not expose every semantic distinction in an operation. The model should record:

- the underlying scalar type, including `int64` and `uint64`
- whether a numeric minimum and maximum are bounded
- finite bounds when present
- the default value and whether a default is available
- enum members and whether a type is a flags bitmask, including whether `0` is a member
- the default of a flags option, which is currently not read at all
- array and object-like categories needed by the generator

Unbounded numeric ranges must not be serialized as JSON `-Inf` or `+Inf`; bound presence should be represented structurally. Ten arguments declare an unbounded range: five optional inputs (`clamp.min`, `clamp.max`, `find_trim.threshold`, `stdif.m0`, `stdif.s0`) and five outputs (`avg.out`, `deviate.out`, `hist_entropy.out`, `max.out`, `min.out`). A single infinity aborted the whole `-debug` dump, so it silently stopped writing `debug_operations.json`; bounds are now reported as finite-or-absent through `HasMinimum`/`HasMaximum`. Metadata must not be used to claim that an empty string, empty array, or other value has operation-specific semantics when libvips does not expose that fact.

## Alternatives considered

### Pointer-everything fields

Making every optional field a pointer would preserve presence, but would make strings and slices less idiomatic and would add pointer noise where a value representation is sufficient. The v2 policy therefore uses pointers selectively.

### Private presence bits

```go
type Options struct {
    Width    int
    widthSet bool
}
```

This avoids pointer fields, but presence is hidden and can become inconsistent with direct struct literals, copying, unmarshalling, and generated code. Setters can update the hidden bit, but callers must remember that direct assignment does not mark the field present.

### Generic `Option[T]` fields

```go
type Options struct {
    Width Option[int]
}
```

This can represent presence explicitly, but it introduces a custom abstraction for every field and makes generated code noisier:

```go
Width: Some(0)
```

It is worth reconsidering only if vipsgen needs a true three-state model such as unset, explicitly nil, and set to a value. For the current problem, pointers are simpler and align with common Go configuration APIs.

### Functional options

```go
Thumbnail(image, WithWidth(0), WithIntent(IntentPerceptual))
```

Functional options are useful for constructors with a small, stable configuration surface. vipsgen has a large generated option surface that callers may want to inspect, copy, compose, and pass through. Functional options would make generated APIs harder to discover and make the final configuration less visible. They could be added later as convenience helpers, but should not be the underlying representation.

## Compatibility and migration

This is intentionally a v2 source-breaking change:

1. Regenerate every options struct using the metadata-informed mixed representation.
2. Update callers to use `vips.Ptr(value)` or local variables and pointers.
3. Remove all zero-value exceptions and special-case allowlists.
4. Regenerate the checked-in bindings for supported libvips versions.
5. Update examples and documentation.

A temporary compatibility layer could provide helpers for common values, but v1 options structs should not be silently translated because doing so risks preserving the old omission bug.

## Implementation plan

1. Complete the metadata first: flags defaults, enum/flags membership of `0`, bound presence instead of infinities, 64-bit ranges, and an `IsFlags` flag on `Argument`. Bound presence, flags defaults, 64-bit ranges and `IsFlags` are in place (PRs #107 and #108), with a per-version metadata golden and `go test ./internal/...` now running in CI.
2. Update Go option struct generation.
3. Add the generic `Ptr[T]` helper to the generated or hand-written support package.
4. Update Go wrapper generation to test pointers and pass presence flags.
5. Update image methods and operation call generation to use the new option fields.
6. Update C wrapper templates so setters are exact and presence is explicit.
7. Delete `shouldIntAllowZero` and related zero-skipping exceptions.
8. Add focused generator snapshots for representative numeric, enum, flag, bool, string, slice, and pointer-like fields.
9. Add behavior tests covering omitted values and explicit zero-like values, plus a classification test driven by the committed introspection dump ("a field pointerified here, a value field there") so the rule is reviewable per libvips version.
10. Run the generator test and build gates:

```bash
go test ./internal/generator
go build -o /dev/null ./cmd/vipsgen
```

Generated packages should also be compiled when wrapper declarations or call signatures change.

## Acceptance criteria

The v2 implementation is correct when all of the following hold:

- An omitted option causes libvips to apply its operation default.
- An explicitly supplied `0` reaches libvips as `0`, wherever the option's range admits `0`.
- An explicitly supplied zero-valued enum or flags value reaches libvips as that value.
- An explicitly supplied `false` reaches libvips as `false` for any boolean the rule makes a pointer (on libvips 8.18.2 the rule selects none).
- An explicitly supplied `0.0` reaches libvips as `0.0`.
- An explicitly supplied empty string is distinguishable from omission whenever the option's default is non-empty.
- A non-`nil` empty slice reaches libvips as an empty array, which requires the Go conversion to keep `nil` and an empty slice distinct and the C wrapper to build a zero-length `VipsArray*`.
- A field whose zero value is out of range, or equals the operation default, keeps an ordinary value type.
- No generated code depends on a manually maintained list of fields that allow zero.
- Pointer presence, string values, and slice presence follow their respective policies across all generated operations.
