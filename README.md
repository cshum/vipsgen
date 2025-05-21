# vipsgen

[![Go Reference](https://pkg.go.dev/badge/github.com/cshum/vipsgen.svg)](https://pkg.go.dev/github.com/cshum/vipsgen)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/cshum/vipsgen)
[![CI](https://github.com/cshum/vipsgen/actions/workflows/ci.yml/badge.svg)](https://github.com/cshum/vipsgen/actions/workflows/ci.yml)

vipsgen is a Go binding generator for [libvips](https://github.com/libvips/libvips) - a fast and efficient image processing library.

Existing Go libvips bindings rely on manually written code that is often incomplete, error-prone, and difficult to maintain as libvips evolves. vipsgen aims to solve this problem by generating type-safe, robust, and fully documented Go bindings using GObject introspection.

vipsgen provides a pre-generated library you can import directly `github.com/cshum/vipsgen/vips`. Also allows code generation via `vipsgen` command that adapts to your specific libvips installation.

- **Coverage**: Comprehensive bindings for over 200 libvips operations
- **Type-Safe**: Generates proper Go types for libvips enums and structs
- **Idiomatic**: Creates clear Go style code that feels natural to use
- **Streaming**: Includes `VipsSource` bindings with `io.ReadCloser` integration for streaming

## Quick Start

Use homebrew to install vips and pkg-config:
```
brew install vips pkg-config

```

On MacOS, vipsgen may not compile without first setting an environment variable:

```bash
export CGO_CFLAGS_ALLOW="-Xpreprocessor"
```

Use the package directly:

```bash
go get -u github.com/cshum/vipsgen/vips
```

vipsgen provides rich options for fine-tuning image operations. Each operation can accept a nil value for default options, or customize optional arguments with specific option structs:

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	
	"github.com/cshum/vipsgen/vips"
)

func main() {
	// Fetch an image from http.Get
	resp, err := http.Get("https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png")
	if err != nil {
		log.Fatalf("Failed to fetch image: %v", err)
	}
	defer resp.Body.Close()

	// Create source from io.ReadCloser
	source := vips.NewSource(resp.Body)
	defer source.Close() // source needs to remain available during image lifetime

	// Shrink-on-load via creating image from thumbnail source with options
	image, err := vips.NewThumbnailSource(source, 800, &vips.ThumbnailSourceOptions{
		Height: 1000,
		FailOn: vips.FailOnError, // Fail on first error
	})
	if err != nil {
		log.Fatalf("Failed to load image: %v", err)
	}
	defer image.Close() // always close images to free memory

	// Add a yellow border using vips_embed
	border := 10
	if err := image.Embed(
		border, border,
		image.Width()+border*2,
		image.Height()+border*2,
		&vips.EmbedOptions{
			Extend:     vips.ExtendBackground, // extend with colour from the background property
			Background: []float64{255, 255, 0, 255}, // Yellow border
		},
	); err != nil {
		log.Fatalf("Failed to add border: %v", err)
	}

	fmt.Printf("Processed image: %dx%d\n", image.Width(), image.Height())

	// Save the result as WebP file with options
	err = image.Webpsave("resized-gopher.webp", &vips.WebpsaveOptions{
		Q:              85,    // Quality factor (0-100)
		Lossless:       false, // Use lossy compression
		Effort:         4,     // Compression effort (0-6)
		SmartSubsample: true,  // Better chroma subsampling
	})
	if err != nil {
		log.Fatalf("Failed to save image as WebP: %v", err)
	}
	fmt.Println("Successfully saved processed images")
}
```

## Code Generation

Code generation requires libvips to be built with GObject introspection support.

```bash
go install github.com/cshum/vipsgen/cmd/vipsgen@latest
```

Generate the bindings:

```bash
vipsgen -out ./vips
```

Use your custom-generated code:

```go
package main

import (
    "yourproject/vips"
)

```

### Command Line Options

```
Usage: vipsgen [options]

Options:
-out string            Output directory (default "./out")
-templates string      Template directory (uses embedded templates if not specified)
-extract               Extract embedded templates and exit
-extract-dir string    Directory to extract templates to (default "./templates")
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT
