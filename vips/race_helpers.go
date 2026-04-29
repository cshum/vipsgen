//go:build race

// Helpers compiled only under -race. Kept in a non-_test.go file because
// Go disallows cgo statements inside _test.go files when the package
// already uses cgo (vips package builds with cgo via util.go etc.).

package vips

// #cgo pkg-config: vips
// #include "util.h"
import "C"

// raceTestSetConcurrency sets libvips' worker thread count and returns the
// previous value. Used by source_close_race_test.go to pin libvips to a
// single worker thread for deterministic race-window reproduction.
func raceTestSetConcurrency(n int) int {
	prev := int(C.vips_concurrency_get())
	C.vips_concurrency_set(C.int(n))
	return prev
}
