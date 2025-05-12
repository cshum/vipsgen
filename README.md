# vipsgen

A Go code generator for libvips bindings—solving the maintenance and compatibility challenges of manual bindings.

## Overview

vipsgen automatically generates Go bindings for [libvips](https://github.com/libvips/libvips), a fast image processing library. Unlike existing Go libraries that rely on manual CGO bindings, vipsgen takes a true Go approach by using code generation to create bindings that automatically adapt to libvips changes.

## Why vipsgen?

The traditional approach to Go bindings for C libraries like libvips has significant drawbacks:

- **Manual maintenance burden**: Existing libraries require manual updates when libvips adds or changes operations
- **Build inconsistencies**: Different build environments and libvips versions can cause unexpected behavior
- **Incomplete coverage**: Manual bindings typically cover only a subset of libvips functionality
- **Incompatibility issues**: Breaking changes in libvips often lead to broken Go bindings

vipsgen solves these problems by:

- **Introspecting libvips at build time**: Dynamically discovers available operations
- **Generating type-safe bindings**: Creates Go code that matches the exact API of your libvips installation
- **Automating updates**: When libvips changes, regenerate your bindings with a single command
- **Following Go idioms**: Uses code generation, which is the Go way of solving interface challenges

## Features

- **Full Coverage**: Automatically generates bindings for all available libvips operations
- **Type-Safe**: Generates proper Go types for libvips enums and structs
- **Idiomatic**: Creates clean, Go-style APIs that feel natural to use
- **Embedded Templates**: Includes templates in the binary for easy distribution
- **Customizable**: Override or exclude specific operations when needed
- **Zero Dependencies**: Just needs libvips and Go

## Installation

```bash
go install github.com/yourusername/vipsgen/cmd/vipsgen@latest
```

## Requirements

- Go 1.16+
- libvips 8.10+
- pkg-config

## Quick Start

1. Generate the bindings:

```bash
# Using embedded templates (simplest)
vipsgen -out ./vips

# Or with custom templates
vipsgen -templates ./my-templates -out ./vips
```

2. Use the generated code in your project:

```go
package main

import (
	"fmt"
	"github.com/yourusername/vips"
)

func main() {
	// Initialize vips
	vips.Initialize()
	defer vips.Shutdown()

	// Load an image
	image, err := vips.NewImageFromFile("input.jpg")
	if err != nil {
		panic(err)
	}
	defer image.Close()

	// Resize the image
	if err := image.Resize(0.5, nil); err != nil {
		panic(err)
	}

	// Save the result
	if err := image.WriteToFile("output.jpg", vips.NewJpegSaveParams()); err != nil {
		panic(err)
	}
}
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
