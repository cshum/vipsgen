# vipsgen

vipsgen is a Go binding generator for [libvips](https://github.com/libvips/libvips) - a fast and efficient image processing library.

Existing Go libvips bindings rely on manually written code that is often incomplete, error-prone, and difficult to maintain as libvips evolves. vipsgen aims to solve this problem by using GObject introspection to automatically generate type-safe, efficient, and fully documented Go bindings that adapt to your specific libvips installation.

vipsgen provides a pre-generated library you can import directly (`github.com/cshum/vipsgen/vips`), but also allows you to generate bindings for your specific libvips installation.

- **Coverage**: Comprehensive bindings that cover most of the libvips operations, with allowing custom code
- **Type-Safe**: Generates proper Go types for libvips enums and structs
- **Idiomatic**: Creates Go style code that feels natural to use
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

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"github.com/cshum/vipsgen/vips"
)

func main() {
	// Fetch an image from URL
	resp, err := http.Get("https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png")
	if err != nil {
		log.Fatalf("Failed to fetch image: %v", err)
	}

	// Create source from io.ReadCloser
	source := vips.NewSource(resp.Body)
	defer source.Close() // source needs to remain available during image lifetime

	// Load image from source
	image, err := vips.NewImageFromSource(source, nil)
	if err != nil {
		log.Fatalf("Failed to load image: %v", err)
	}
	defer image.Close() // always close images to free memory

	// Resize the image
	if err := image.Resize(0.5, nil); err != nil {
		log.Fatalf("Failed to resize image: %v", err)
	}

	// Save the result as WebP
	buf, err := image.WebpsaveBuffer(nil)
	if err != nil {
		log.Fatalf("Failed to save image as WebP: %v", err)
	}

	// Write to file
	if err := os.WriteFile("resized-gopher.webp", buf, 0666); err != nil {
		log.Fatalf("Failed to write file: %v", err)
	}

	fmt.Println("Successfully resized image to resized-gopher.webp")
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

## How It Works

vipsgen uses C bindings and reflection to:

1. Introspect the libvips library at build time
2. Discover all available operations and their parameters
3. Generate Go code with proper type mappings
4. Create helper methods for common operations

The result is a complete, type-safe, and efficient binding to libvips that stays up-to-date with the underlying library.

## Command Line Options

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
