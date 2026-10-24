# Migrating from v1 to v2

v2 fixes one thing: an option set to a zero value is no longer indistinguishable from an option
you never touched. Everything else is small.

## The bug worth upgrading for

In v1, vipsgen decided whether to pass an option by looking at its value. A zero meant "not
supplied", so it was dropped, and libvips applied its default instead.

```go
// v1: you ask for uncompressed PNG
&vips.PngsaveOptions{Compression: 0}

// v1 passed nothing. libvips used its default, 6.
```

That is a silent wrong answer. The output is a valid PNG, so nothing fails; it is just not the
file you asked for. It bit `effort`, `compression` and `intent` before v2, and each was patched
by hand. A growing list of special cases is not a design.

## How v2 reads an options struct

```go
&vips.PngsaveOptions{}                          // not supplied: libvips default, 6
&vips.PngsaveOptions{Compression: vips.Ptr(0)}  // supplied: store uncompressed
&vips.PngsaveOptions{Compression: vips.Ptr(6)}  // supplied: 6, same bytes as the default
```

`nil` means "not supplied". `vips.Ptr(v)` means "supplied", zero included. The type tells you
which fields work this way: `*int` against `int`, `*string` against `string`.

Why not make every field a pointer? Because most fields do not need it. A field is only a
pointer when libvips accepts its zero **and** that zero differs from the option's default.
Everywhere else a plain value is already unambiguous, and keeps the structs readable.

## Which fields changed, and why you can trust it

The classification comes from GParamSpec: the accepted range, the enum or flag members, and the
default. vipsgen does not keep a list of field names, so a new libvips option is handled the
same way as the old ones.

The numbers from the shipped package: 220 options structs, 1,143 optional fields, and 208 of
them are pointers. Most of your options code does not move.

Some do, though, because a zero is meaningful there: `Compression`, `Kernel`, `Id`.

| v1 | v2 |
|----|----|
| `&vips.PngsaveOptions{Compression: 0}` — the 0 was dropped | `&vips.PngsaveOptions{Compression: vips.Ptr(0)}` — stores uncompressed |
| `&vips.ResizeOptions{Kernel: vips.KernelLanczos3}` | `&vips.ResizeOptions{Kernel: vips.Ptr(vips.KernelLanczos3)}` |
| `&vips.ThumbnailOptions{Height: 1000}` | same: `Height` has no zero/default conflict |

Strings follow the same rule when their default is not empty. `dzsave`'s `id` defaults to
`https://example.com/iiif` and is written into IIIF tile references, so an empty string is a
value you might want, and v1 could not pass it.

```go
&vips.DzsaveOptions{}                 // id defaults to "https://example.com/iiif"
&vips.DzsaveOptions{Id: vips.Ptr("")} // no prefix; tiles are referenced relatively
```

Spelling a pointer where v1 had a value is a compile error, so the compiler lists the work for
you. The one silent change is `vips.KeepNone` below.

## What introspection cannot tell, and how those were settled

GParamSpec gives a range and a default, not what libvips does with the value. `gifload.n` and
`webpload.n` have the same range and the same default, and only the first rejects a zero
(`bad page number`). No amount of introspection separates them.

So each numeric pointer argument was called twice: once with an explicit zero, once without.
Every saver accepted the zero (`pngsave.compression = 0` really does store uncompressed), which
keeps the rule intact for savers. The arguments that rejected it stay plain values:

    gifload.n    webpload.n   tiffload.n   tiffload.subifd   webpload.scale
    svgload.scale   heifload.n   pdfload.n   magickload.n   jxlload.n
    csvload.lines   similarity.scale   dzsave.suffix

The list is keyed by operation and argument, so a same-named argument on a different operation
is unaffected.

## `vips.KeepNone` moved

`vips.KeepNone` is `-1` in v1 and `0` in v2, matching `VIPS_FOREIGN_KEEP_NONE` in C. Code that
hardcodes `-1` still compiles and now means something else, so check for it.

## The `pointer` package is gone

v1, and v2 up to 2.0.5, carried `pointer.Save/Restore/Unref/Clear` to pass a Go object through C
as an index. The generated code uses `cgo.Handle`, which does the same job in the runtime, so the
package is gone. Nothing in this module imported it and pkg.go.dev recorded no importers. If you
called it directly, `cgo.Handle` replaces it.

## Reading the argument

- [Discussion #105](https://github.com/cshum/vipsgen/discussions/105) — the problem, written
  before the fix existed.
- [PR #109](https://github.com/cshum/vipsgen/pull/109) — the implementation, with the GParamSpec
  rule, the measurements and the cases that did not fit.
