# Examples

Each directory is a small program showing one way to use the bindings. Run one from the
repository root:

```bash
go run ./examples/from_file
```

| example | what it does |
|---|---|
| [animated_gif](animated_gif/main.go) | loads a GIF with `NewGifloadBuffer`, pixelates every frame, saves `dancing-banana-pixelated.gif` |
| [from_buffer](from_buffer/main.go) | loads a `[]byte` with `NewImageFromBuffer`, saves `gopher.jpg` |
| [from_file](from_file/main.go) | loads a path with `NewImageFromFile`, crops every page, flattens, saves `dancing-banana-cropped.gif` |
| [go_image](go_image/main.go) | turns a Go `image.Image` into a libvips image, saves `gopher.jpg` |
| [io_reader_writer](io_reader_writer/main.go) | reads an `io.Reader` through a `Source`, thumbnails, writes `resized-gopher.webp` through a target |
| [option_presence](option_presence/main.go) | writes `presence.png` to a temporary directory to show what `nil` and `vips.Ptr(0)` mean in an options struct |
| [optional_outputs](optional_outputs/main.go) | mosaics two images, reads back the optional outputs, saves `mosaic_result.jpg`, `smartcrop_result.jpg` and `flood_fill_result.jpg` |

Most examples download their input, so they need network access.
