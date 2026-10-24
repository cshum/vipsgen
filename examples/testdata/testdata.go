// Package testdata locates the sample images the examples read.
package testdata

import (
	"path/filepath"
	"runtime"
)

// Path returns the full path of a file in this directory, taken from this file's own location
// so an example finds its input whatever directory it is run from.
func Path(name string) string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return name
	}
	return filepath.Join(filepath.Dir(file), name)
}
