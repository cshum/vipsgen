# Migrating from v1 to v2

v2 changes the import path and how an optional argument says "I supplied this".

## Module path

```go
import "github.com/cshum/vipsgen/v2/vips"
```

v1 stays consumable at its existing import paths, so v1 code keeps building while you move.

## Option presence

A field is a **pointer** when libvips accepts its zero value and that zero differs from the option's default. `nil` means "not supplied" — the operation uses its own default. `Ptr` means "supplied", including a zero value. Every other field stays a plain value.

| v1 | v2 |
|----|----|
| `&vips.PngsaveOptions{}` — compression skipped | unchanged: libvips default |
| `&vips.PngsaveOptions{Compression: 0}` — 0 silently dropped | `&vips.PngsaveOptions{Compression: vips.Ptr(0)}` — stores uncompressed |
| `&vips.ResizeOptions{Kernel: vips.KernelLanczos3}` | `&vips.ResizeOptions{Kernel: vips.Ptr(vips.KernelLanczos3)}` |
| `&vips.ThumbnailOptions{Height: 1000}` | unchanged — `Height` has no ambiguity to resolve |
| `vips.KeepNone` (-1, remapped) | `vips.KeepNone` (0, matching C `VIPS_FOREIGN_KEEP_NONE`) |

`LoadOptions` fields are pointers as well, and `OptionString` tests nil rather than zero, so `Ptr(false)` is expressible.

## Removed

The generated `Default<Op>Options` constructors, around 220 of them. With pointer fields the zero value of an options struct already means "use the defaults":

```go
// v1
options := vips.DefaultSmartcropOptions()

// v2
options := &vips.SmartcropOptions{}
```

`DefaultLoadOptions` stays: it carries a real default, `FailOnError: true`.

## Why the change

v1 decided an option was "not supplied" by looking at its value, so an explicit `0` was dropped and libvips applied its own default instead. The v2 rule is derived from GParamSpec metadata — the accepted range for numbers, the member list for enums and flags, the default for both — so a new libvips option is classified without anyone maintaining a list of exceptions. 216 of 1202 optional arguments became pointers.

Two limits are deliberate. Array options keep v1 behaviour, where an empty slice means "not supplied". And operation-specific semantics that GParamSpec does not describe are not guessed at; where libvips treats a value differently from a default in a way introspection cannot see, the binding cannot know.
