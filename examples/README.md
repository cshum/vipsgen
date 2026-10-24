# Examples

Each directory is a small program showing one way to use the bindings. Run one from the
repository root:

```bash
go run ./examples/from_file
```

| example | shows |
|---|---|
| [animated_gif](animated_gif/main.go) | applying an operation to every frame of an animated image |
| [from_buffer](from_buffer/main.go) | loading from memory with `NewImageFromBuffer` |
| [from_file](from_file/main.go) | loading every frame of an animated GIF with `N: -1` |
| [go_image](go_image/main.go) | turning a Go `image.Image` into a libvips image |
| [io_reader_writer](io_reader_writer/main.go) | reading from an `io.Reader` through a `Source` |
| [option_presence](option_presence/main.go) | what `nil` and `vips.Ptr(0)` mean in an options struct |
| [optional_outputs](optional_outputs/main.go) | reading values an operation reports back through options |

Most examples download their input, so they need network access.
