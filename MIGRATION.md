# Migrating from v1 to v2

v2 changes the import path, how an optional argument is passed, and the value of
`vips.KeepNone`.

The `vips.Ptr` changes are compile errors in v1 spelling, so the compiler finds them for you.
The `vips.KeepNone` change compiles and is silent, so it has its own section.

## Module path

```go
import "github.com/cshum/vipsgen/v2/vips"
```

v1 keeps its import paths, so v1 code keeps building while you move.

## When a field is a pointer

A field becomes a pointer when libvips accepts its zero value and that zero differs from the
option's default. Only then can `nil` mean "not supplied" while `vips.Ptr(v)` passes `v`,
including zero.

Every other field stays a plain value: its zero is either not accepted by libvips, or is the
value libvips would use anyway, so there is nothing to tell apart.

Most options keep their v1 spelling.

```go
// examples/option_presence runs this and prints the resulting sizes
&vips.PngsaveOptions{}                          // libvips default, compression 6
&vips.PngsaveOptions{Compression: vips.Ptr(0)}  // store uncompressed
&vips.PngsaveOptions{Compression: vips.Ptr(6)}  // same output as the default
```

Strings follow the same rule when the default is not empty. `dzsave`'s `id` defaults to
`https://example.com/iiif` and is written into IIIF tile references, so an empty string is a
value a caller may want. v1 could not pass it: the empty string was dropped and the default
applied instead.

```go
&vips.DzsaveOptions{}                 // id defaults to "https://example.com/iiif"
&vips.DzsaveOptions{Id: vips.Ptr("")} // no prefix; tiles are referenced relatively
```

| v1 | v2 |
|----|----|
| `&vips.PngsaveOptions{}` — compression not set | unchanged: libvips default |
| `&vips.PngsaveOptions{Compression: 0}` — the 0 was dropped | `&vips.PngsaveOptions{Compression: vips.Ptr(0)}` — stores uncompressed |
| `&vips.ResizeOptions{Kernel: vips.KernelLanczos3}` | `&vips.ResizeOptions{Kernel: vips.Ptr(vips.KernelLanczos3)}` |
| `&vips.ThumbnailOptions{Height: 1000}` | unchanged: `Height` has no zero/default conflict |

## vips.KeepNone moved

`vips.KeepNone` is -1 in v1 and 0 in v2, matching `VIPS_FOREIGN_KEEP_NONE` in C. Code that
hardcodes -1 still compiles and now means something else, so check for it.

## The exception

The rule reads GParamSpec, which gives a range and a default but not what libvips does with a
value. `gifload.n` and `webpload.n` have the range -1 to 100000 and the default 1, and both
reject 0 with `bad page number`. No range or default shows that, so both stay plain values.

Measured and kept as plain values:

    gifload.n    webpload.n   tiffload.n   tiffload.subifd   webpload.scale
    svgload.scale   heifload.n   pdfload.n   magickload.n   csvload.lines
    similarity.scale   dzsave.suffix

Every saver argument was measured too, and all accept an explicit zero
(`pngsave.compression = 0` stores uncompressed), so the rule stays for them. The list is keyed
by operation and argument, so a same-named argument elsewhere is unaffected.

## Why the change

v1 decided an option was "not supplied" by looking at the value. An explicit `0` looked exactly
like an unset field, so it was dropped and libvips used its own default.

v2 takes the answer from GParamSpec: the accepted range for numbers, the member list for enums
and flags, the default for both. A new libvips option is classified without anyone maintaining
a list.

The limit is unchanged: GParamSpec does not describe what an operation does with a value.
Where libvips treats one value differently from its default in a way introspection cannot see,
the binding cannot know. Those cases are measured and kept in the list above.
