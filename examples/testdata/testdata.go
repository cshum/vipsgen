// Package testdata locates the sample images the examples read.
package testdata

import (
	"path/filepath"
	"runtime"
)

// Path returns the full path of a file in this directory.
//
// The path is taken from this file's own location at compile time, so an example finds its
// input whatever directory it is run from - `go run ./examples/from_file` and
// `cd examples/from_file && go run .` both work. It assumes the source tree is present, which
// it is for an example run from a checkout.
func Path(name string) string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return name
	}
	return filepath.Join(filepath.Dir(file), name)
}
