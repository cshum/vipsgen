package introspection

// #cgo pkg-config: vips
// #include "introspection.h"
import "C"
import (
	"log"
)

// Introspection provides discovery and analysis of libvips operations
// through reflection of the C library's type system, extracting operation
// metadata, argument details, and supported enum types.
type Introspection struct {
	discoveredEnumTypes  map[string]string
	enumTypeNames        []enumTypeName
	discoveredImageTypes map[string]ImageTypeInfo
}

// NewIntrospection creates a new Introspection instance for analyzing libvips
// operations, initializing the libvips library in the process.
func NewIntrospection() *Introspection {
	// Initialize libvips
	if C.vips_init(C.CString("vipsgen")) != 0 {
		log.Fatal("Failed to initialize libvips")
	}
	defer C.vips_shutdown()

	return &Introspection{
		discoveredEnumTypes:  make(map[string]string),
		discoveredImageTypes: map[string]ImageTypeInfo{},
	}
}
