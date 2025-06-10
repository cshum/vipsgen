# vipsgen

[![Go Reference](https://pkg.go.dev/badge/github.com/cshum/vipsgen/vips.svg)](https://pkg.go.dev/github.com/cshum/vipsgen/vips)
[![CI](https://github.com/cshum/vipsgen/actions/workflows/ci.yml/badge.svg)](https://github.com/cshum/vipsgen/actions/workflows/ci.yml)

vipsgen is a Go binding generator for [libvips](https://github.com/libvips/libvips) - a fast and efficient image processing library.

libvips is generally 4-8x [faster](https://github.com/libvips/libvips/wiki/Speed-and-memory-use) than ImageMagick with low memory usage, thanks to its [demand-driven, horizontally threaded](https://github.com/libvips/libvips/wiki/Why-is-libvips-quick) architecture.

Existing Go libvips bindings rely on manually written code that is often incomplete, error-prone, and difficult to maintain as libvips evolves.
vipsgen solves this by generating type-safe, robust, and fully documented Go bindings using GObject introspection.

You can use vipsgen in two ways:

- **Import directly**: Use the pre-generated library `github.com/cshum/vipsgen/vips` for the default installation of libvips 8.17
- **Generate custom bindings**: Run the vipsgen command to create bindings for your specific libvips version and installation

### Features

- **Comprehensive**: Bindings for around [300 libvips operations](https://www.libvips.org/API/current/function-list.html)
- **Type-Safe**: Proper Go types for all libvips C enums and structs
- **Idiomatic**: Clean Go APIs that feel natural to use
- **Streaming**: `VipsSource` and `VipsTarget` integration with Go `io.Reader` and `io.Writer` for [streaming](https://www.libvips.org/2019/11/29/True-streaming-for-libvips.html)

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

All operations support optional configuration where applicable. Pass nil to use default settings, or provide an options struct for customization. These bindings maintain direct argument equivalence with the [libvips API](https://www.libvips.org/API/current/). See [examples](https://github.com/cshum/vipsgen/tree/main/examples) for common usage patterns.


```go
package main

import (
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
			Extend:     vips.ExtendBackground,       // extend with colour from the background property
			Background: []float64{255, 255, 0, 255}, // Yellow border
		},
	); err != nil {
		log.Fatalf("Failed to add border: %v", err)
	}

	log.Printf("Processed image: %dx%d\n", image.Width(), image.Height())

	// Save the result as WebP file with options
	err = image.Webpsave("resized-gopher.webp", &vips.WebpsaveOptions{
		Q:              85,   // Quality factor (0-100)
		Effort:         4,    // Compression effort (0-6)
		SmartSubsample: true, // Better chroma subsampling
	})
	if err != nil {
		log.Fatalf("Failed to save image as WebP: %v", err)
	}
	log.Println("Successfully saved processed images")
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
-debug                 Enable debug json output
```

### How Code Generation Works

The generation process involves multiple layers to provide a type-safe, idiomatic Go API:

1. **Introspection Analysis**: vipsgen uses GObject introspection to analyze the libvips API, extracting operation metadata, argument types, and enum definitions.

2. **Multi-Layer Generation**: To create type-safe, idiomatic Go APIs from libvips dynamic parameter system, vipsgen creates a layered approach that handles both required and optional parameters.

3. **Type-Safe Bindings**: The generated code is fully type-safe with proper Go types, structs, and enums based on centralized introspection data.

```
┌─────────────────────────────────────────────────────────────┐
│                    Go Method Layer                          │
│  • Methods on *Image struct                                 │
│  • Go enums and structs                                     │  
│  • Options structs for optional parameters                  │
│  • Type conversions (Go <-> C)                              │
└─────────────────────────────────────────────────────────────┘
                               │
┌─────────────────────────────────────────────────────────────┐
│                   Go Binding Layer                          │
│  • vipsgenAbc() - required parameters only                  │
│  • vipsgenAbcWithOptions() - with optional parameters       │
│  • C array handling and memory management                   │
│  • String conversions and cleanup                           │
│  • Error handling and resource management                   │
└─────────────────────────────────────────────────────────────┘
                               │
┌─────────────────────────────────────────────────────────────┐
│                     C Layer                                 │
│  • vipsgen_abc() - required args only                       │
│  • vipsgen_abc_with_options() - all parameters              │
│  • VipsOperation dynamic dispatch                           │
│  • Proper VipsArray creation and cleanup                    │
└─────────────────────────────────────────────────────────────┘
                               │
┌─────────────────────────────────────────────────────────────┐
│                    libvips                                  │
│  • vips_abc() - original variadic functions                 │
│  • VipsOperation object system                              │
│  • GObject introspection metadata                           │
└─────────────────────────────────────────────────────────────┘
```

**1. C Layer (vips.c/vips.h)**

**Problem**: libvips dynamic parameter system with variadic functions like `vips_resize(in, &out, scale, "kernel", kernel, ...)` does not translate well to type-safe, idiomatic Go APIs.

**Solution**: Generate two types of C wrapper functions:

```c
// Basic function - required arguments only, calls vips_resize directly
int vipsgen_resize(VipsImage* in, VipsImage** out, double scale) {
    return vips_resize(in, out, scale, NULL);
}

// With options - uses VipsOperation for optional parameters  
int vipsgen_resize_with_options(VipsImage* in, VipsImage** out, double scale, 
                               VipsKernel kernel, double gap, double vscale) {
    VipsOperation *operation = vips_operation_new("resize");
    if (!operation) return 1;
    if (
        vips_object_set(VIPS_OBJECT(operation), "in", in, NULL) ||
        vips_object_set(VIPS_OBJECT(operation), "scale", scale, NULL) ||
        vipsgen_set_int(operation, "kernel", kernel) ||
        vipsgen_set_double(operation, "gap", gap) ||
        vipsgen_set_double(operation, "vscale", vscale)
    ) {
        g_object_unref(operation);
        return 1;
    }
    int result = vipsgen_operation_execute(operation, "out", out, NULL);
    return result;
}
```

This layer handles VipsArray creation/cleanup, VipsOperation lifecycle management, type-specific setters.

**2. Go Binding Layer (vips.go)**

**Problem**: C arrays, string management, and complex type conversions.

**Solution**: Generate Go wrapper functions that handle CGO complexity:

```go
// vipsgenResize vips_resize resize an image
func vipsgenResize(in *C.VipsImage, scale float64) (*C.VipsImage, error) {
    var out *C.VipsImage
    if err := C.vipsgen_resize(in, &out, C.double(scale)); err != 0 {
        return nil, handleImageError(out)
    }
    return out, nil
}

// vipsgenResizeWithOptions vips_resize resize an image with optional arguments
func vipsgenResizeWithOptions(in *C.VipsImage, scale float64, kernel Kernel, 
                             gap float64, vscale float64) (*C.VipsImage, error) {
    var out *C.VipsImage
    if err := C.vipsgen_resize_with_options(in, &out, C.double(scale), 
                                           C.VipsKernel(kernel), C.double(gap), 
                                           C.double(vscale)); err != 0 {
        return nil, handleImageError(out)
    }
    return out, nil
}
```

This layer handles C array conversion, string conversion with cleanup, memory management, error handling, and type conversion between Go and C.

**3. Go Method Layer (image.go)**

**Problem**: Provide idiomatic Go API with proper encapsulation.

**Solution**: Generate methods on `*Image` struct that encapsulate the two-function approach with options pattern:

```go
// ResizeOptions optional arguments for vips_resize
type ResizeOptions struct {
    // Kernel Resampling kernel
    Kernel Kernel
    // Gap Reducing gap
    Gap float64
    // Vscale Vertical scale image by this factor
    Vscale float64
}

// DefaultResizeOptions creates default value for vips_resize optional arguments
func DefaultResizeOptions() *ResizeOptions {
    return &ResizeOptions{
        Kernel: Kernel(5),
        Gap: 2,
    }
}

// Resize vips_resize resize an image
func (r *Image) Resize(scale float64, options *ResizeOptions) error {
    if options != nil {
        // Use the WithOptions variant when options are provided
        out, err := vipsgenResizeWithOptions(r.image, scale, 
                                           options.Kernel, options.Gap, options.Vscale)
        if err != nil {
            return err
        }
        r.setImage(out)
        return nil
    }
    // Use the basic variant for required parameters only
    out, err := vipsgenResize(r.image, scale)
    if err != nil {
        return err
    }
    r.setImage(out)
    return nil
}
```

This layer provides idiomatic Go methods, options structs for optional parameters, Go type system integration.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Development Workflow

When contributing to vipsgen, **do not commit the generated code** in the `vips/` directory (except `vips/vips_test.go`). The development workflow is designed to keep generated code separate from source code. The repository uses GitHub Actions to automatically handle code generation when PRs are created.

## License

MIT
