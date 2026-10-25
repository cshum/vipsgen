# Migrating from v1 to v2

v2 changes how an optional argument is passed, and the value of `vips.KeepNone`.

The reason is a problem in the original options API design: generated options use ordinary Go
values, and vipsgen decides whether to pass an option by looking at its value. That becomes a
problem when libvips accepts `0` as a valid explicit option, because vipsgen may treat the value
as "not provided" and skip it. libvips then applies its default instead, which can produce
behaviour different from what was requested.

This has already appeared in several focused fixes:

- [Allow `effort=0` for HEIF and WebP](https://github.com/cshum/vipsgen/pull/95)
- [Allow `compression=0` for PNG](https://github.com/cshum/vipsgen/pull/101)
- [Allow `intent=0` for ICC and thumbnail operations](https://github.com/cshum/vipsgen/pull/103)

Those fixes solve the individual cases, but they also show that the problem was overlooked
during the initial design of the options API. Maintaining a growing list of special cases is
unlikely to be reliable as libvips adds more operations and options.

It also creates an unexpected inconsistency for users of the generated options structs. Because
Go value fields use their zero value when they are not explicitly set, some options may be
passed to libvips even when their value is `0`. That is surprising in normal Go usage, where the
zero value of an options struct is generally expected to mean "use the default" or "do not
provide this option".

## The idea

v2 preserves the distinction between

    not supplied

and

    explicitly supplied with a zero value

Go pointers represent that distinction directly: a `nil` pointer means the option was not
supplied, while a non-`nil` pointer means it was explicitly supplied. The value can still be
`0`, `false`, or another zero value.

```go
&vips.PngsaveOptions{}                          // not supplied: libvips default, 6
&vips.PngsaveOptions{Compression: vips.Ptr(0)}  // explicitly supplied: store uncompressed
&vips.PngsaveOptions{Compression: vips.Ptr(6)}  // explicitly supplied: 6, same bytes as the default
```

`vips.Ptr` is a small helper:

```go
func Ptr[T any](value T) *T { return &value }
```

The policy is to use pointers only where they are needed to distinguish an omitted option from
an explicitly supplied zero value. For the other options, the normal Go value type is cleaner,
more idiomatic and easier to use, so those fields remain ordinary values. In practice:

- optional booleans use pointers when `false` needs to be distinguished from omission;
- selected optional numeric and enum values use pointers when zero is meaningful and differs
  from the operation's default.

## Which fields changed

A field becomes a pointer when libvips accepts its zero **and** that zero differs from the
option's default. The classification comes from GParamSpec: the accepted range, the enum or flag
members, and the default.

From the shipped package: 220 options structs, 1,143 optional fields, and 208 of them are
pointers, since a zero is meaningful in those fields — for example `Compression`, `Kernel`
and `Id`.

| v1 | v2 |
|----|----|
| `&vips.PngsaveOptions{Compression: 0}` — the 0 was dropped | `&vips.PngsaveOptions{Compression: vips.Ptr(0)}` — stores uncompressed |
| `&vips.ResizeOptions{Kernel: vips.KernelLanczos3}` | `&vips.ResizeOptions{Kernel: vips.Ptr(vips.KernelLanczos3)}` |
| `&vips.ThumbnailOptions{Height: 1000}` | same: `Height` has no zero/default conflict |

Strings follow the same rule when their default is not empty. `dzsave`'s `id` defaults to
`https://example.com/iiif` and is written into IIIF tile references, so an empty string is a
value a caller may want, and v1 could not pass it.

```go
&vips.DzsaveOptions{}                 // id defaults to "https://example.com/iiif"
&vips.DzsaveOptions{Id: vips.Ptr("")} // no prefix; tiles are referenced relatively
```

Migration for those pointer field is straightforward, because compile errors identify them —
except for `vips.KeepNone`.

## `vips.KeepNone` moved

`vips.KeepNone` is `-1` in v1 and `0` in v2, matching `VIPS_FOREIGN_KEEP_NONE` in C. Code that
hardcodes `-1` still compiles and now means something else, so check for it.

## What introspection cannot tell

GParamSpec tells you the range and default of a field, but not what libvips does with the value.
For instance, `gifload.n` and `webpload.n` have the range -1–100000 and default 1, so by the rule
above they need a pointer. But `n` is the number of pages to load for multi-page or animated
images, where `n=-1` means unlimited and `n=0` is invalid.

That can be verified by loading a multi-page image with `n=0`, which fails with a
(`bad page number`) error. It means `n=0` do not exists and they should stay plain values:

    gifload.n    webpload.n   tiffload.n   tiffload.subifd   webpload.scale
    svgload.scale   heifload.n   pdfload.n   magickload.n   jxlload.n
    csvload.lines   similarity.scale   dzsave.suffix

## Further reading

- [Discussion #105](https://github.com/cshum/vipsgen/discussions/105) — the proposal, written
  before the change.
- [PR #109](https://github.com/cshum/vipsgen/pull/109) — the implementation, with the GParamSpec
  rule, the measurements and the cases that did not fit.
