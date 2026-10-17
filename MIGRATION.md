# Migrating from v1 to v2

v2 changes the import path, and how an optional argument says "I supplied this".

## Module path

```go
import "github.com/cshum/vipsgen/v2/vips"
```

v1 stays consumable at its existing import paths, so v1 code keeps building while you move.

## When a field is a pointer

A field is a pointer when libvips accepts its zero value **and** that zero differs from the
option's default. Then the two cases are distinguishable: `nil` means "not supplied", so
libvips uses its own default, and `Ptr(v)` means "supplied", a zero included. Every other
field stays a plain value, because there the zero is either not a value libvips accepts or
is what it would have used anyway.

```go
// examples/option_presence runs this and prints the resulting sizes
&vips.PngsaveOptions{}                          // libvips default, compression 6
&vips.PngsaveOptions{Compression: vips.Ptr(0)}  // store uncompressed
&vips.PngsaveOptions{Compression: vips.Ptr(6)}  // same output as the default
```

The same applies to strings, where the default is what decides: `dzsave`'s `id` defaults to
`https://example.com/iiif` and is the prefix written into IIIF tile references, so clearing
it is a real request - and one v1 could not express, because it dropped the empty string and
applied the default instead.

```go
&vips.DzsaveOptions{}                 // id defaults to "https://example.com/iiif"
&vips.DzsaveOptions{Id: vips.Ptr("")} // no URL prefix, so tiles are referenced relatively
```

| v1 | v2 |
|----|----|
| `&vips.PngsaveOptions{}` — compression skipped | unchanged: libvips default |
| `&vips.PngsaveOptions{Compression: 0}` — 0 silently dropped | `&vips.PngsaveOptions{Compression: vips.Ptr(0)}` — stores uncompressed |
| `&vips.ResizeOptions{Kernel: vips.KernelLanczos3}` | `&vips.ResizeOptions{Kernel: vips.Ptr(vips.KernelLanczos3)}` |
| `&vips.ThumbnailOptions{Height: 1000}` | unchanged — `Height` has no ambiguity to resolve |
| `vips.KeepNone` (-1, remapped) | `vips.KeepNone` (0, matching C `VIPS_FOREIGN_KEEP_NONE`) |

## The one exception

The rule above reads GParamSpec metadata, which cannot tell a zero that libvips accepts
from a zero it makes sense to pass. `gifload.n` and `webpload.n` have the same range
(-1 to 100000) and the same default (1), and only the first rejects a zero
(`gifload: bad page number`). Measured, those arguments are plain values anyway:

    gifload.n   tiffload.n   tiffload.subifd   webpload.scale   svgload.scale
    heifload.n  pdfload.n   magickload.n   similarity.scale   dzsave.suffix

Every saver argument was measured too and accepts an explicit zero -
`pngsave.compression = 0` stores uncompressed - which is why the rule stays in place for
them. The list keys on operation and argument, not on the argument name.


## What is still there

`Default<Op>Options` survives, but its fields are pointers now, so reading one takes a deref
where v1 returned a value. It fills them from the defaults of the libvips it was generated
against, which makes it a description rather than something to build payloads with: omitting
a field takes whatever the installed libvips uses.

Generated field comments now state the accepted range and the default, so neither has to be
looked up in libvips' documentation.

## Why the change

v1 decided an option was "not supplied" by looking at its value, so an explicit `0` was
dropped and libvips applied its own default instead. The v2 rule is derived from GParamSpec
metadata - the accepted range for numbers, the member list for enums and flags, the default
for both - so a new libvips option is classified without anyone maintaining a list.
201 of 1202 optional arguments are pointers.

Two limits are deliberate. Array options keep v1 behaviour, where an empty slice means
"not supplied". And operation-specific semantics that GParamSpec does not describe are not
guessed at; where libvips treats a value differently from a default in a way introspection
cannot see, the binding cannot know - those cases are measured, and kept in one list.
