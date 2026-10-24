# Examples

Each directory is a small program showing one way to use the bindings. Run one from the
repository root:

```bash
go run ./examples/from_file
```

| example | shows | writes |
|---|---|---|
| [animated_gif](animated_gif/main.go) | applying an operation to every frame, loaded with `GifloadBuffer` | `dancing-banana-pixelated.gif` |
| [from_buffer](from_buffer/main.go) | loading from memory with `NewImageFromBuffer` | `gopher.jpg` |
| [from_file](from_file/main.go) | loading by path with `NewImageFromFile` | `dancing-banana-cropped.gif` |
| [go_image](go_image/main.go) | turning a Go `image.Image` into a libvips image | `gopher.jpg` |
| [io_reader_writer](io_reader_writer/main.go) | reading from an `io.Reader` through a `Source` | `resized-gopher.webp` |
| [option_presence](option_presence/main.go) | what `nil` and `vips.Ptr(0)` mean in an options struct | `presence.png`, in a temporary directory |
| [optional_outputs](optional_outputs/main.go) | reading values an operation reports back through options | `mosaic_result.jpg`, `smartcrop_result.jpg`, `flood_fill_result.jpg` |

Most examples download their input, so they need network access.
