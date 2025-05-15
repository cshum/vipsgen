package introspection

// #cgo pkg-config: vips
// #include "introspection.h"
import "C"
import (
	"fmt"
	"github.com/cshum/vipsgen/internal/generator"
	"log"
	"sort"
	"strings"
	"unsafe"
)

type enumTypeName struct {
	CName  string
	GoName string
}

// VipsFunctionInfo holds information needed to generate a wrapper function
type VipsFunctionInfo struct {
	Name           string
	CIdentifier    string
	ReturnType     string
	Category       string
	HasOutParam    bool
	OutParamIndex  int
	HasVarArgs     bool
	Description    string
	OriginalDoc    string
	Params         []VipsParamInfo
	RequiredParams []VipsParamInfo // Non-optional params
	OptionalParams []VipsParamInfo // Optional params that can be passed as named args
}

// VipsParamInfo represents a parameter for a vips function
type VipsParamInfo struct {
	Name       string
	CType      string
	IsOutput   bool
	IsOptional bool
	IsArray    bool
	ArrayType  string
	IsVarArgs  bool
}

// DebugInfo stores debug information during parsing
type DebugInfo struct {
	ProcessedFunctions         int
	FoundFunctionNames         []string
	MissingCIdentifierIncluded int
}

// Introspection provides discovery and analysis of libvips operations
// through reflection of the C library's type system, extracting operation
// metadata, argument details, and supported enum types.
type Introspection struct {
	discoveredEnumTypes map[string]string
	enumTypeNames       []enumTypeName
	// Parsed function info
	functionInfo []VipsFunctionInfo
	// Debug info from parsing
	debugInfo            *DebugInfo
	discoveredImageTypes map[string]generator.ImageTypeInfo
}

// NewIntrospection creates a new Introspection instance for analyzing libvips
// operations, initializing the libvips library in the process.
func NewIntrospection() *Introspection {
	// Initialize libvips
	if C.vips_init(C.CString("vipsgen")) != 0 {
		log.Fatal("Failed to initialize libvips")
	}
	defer C.vips_shutdown()

	// Initialize map with known enum types
	discoveredTypes := make(map[string]string)
	for _, enum := range baseEnumTypeNames {
		discoveredTypes[enum.CName] = enum.GoName
	}

	return &Introspection{
		discoveredEnumTypes:  discoveredTypes,
		discoveredImageTypes: map[string]generator.ImageTypeInfo{},
		enumTypeNames:        baseEnumTypeNames,
	}
}

// GetAllOperationNames retrieves names of all available operations from libvips
func (v *Introspection) GetAllOperationNames() []string {
	var count C.int
	cNames := C.get_all_operation_names(&count)
	defer C.free_operation_names(cNames, count)

	names := make([]string, int(count))

	// Convert C array to Go slice
	cNamesSlice := (*[1 << 30]*C.char)(unsafe.Pointer(cNames))[:count:count]
	for i, cName := range cNamesSlice {
		names[i] = C.GoString(cName)
	}

	sort.Strings(names)
	return names
}

// FilterOperations filters operations based on availability in the current libvips installation,
// excluded operations list, and deduplicates by Go function name
func (v *Introspection) FilterOperations(operations []generator.Operation) []generator.Operation {
	// Filter out excluded operations and deduplicate by Go function name
	seenFunctions := make(map[string]bool)
	var filteredOps []generator.Operation
	var notAvailableCount, excludedCount, duplicateCount int

	for _, op := range operations {
		// Check if operation can be instantiated in current libvips
		if !v.checkOperationExists(op.Name) {
			notAvailableCount++
			continue
		}
		if strings.Contains(op.Name, "_source") || strings.Contains(op.Name, "_target") ||
			strings.Contains(op.Name, "_mime") {
			fmt.Printf("Excluding operation: %s \n", op.Name)
			excludedCount++
			continue
		}

		// Check if operation is explicitly excluded
		if generator.ExcludedOperations[op.Name] {
			fmt.Printf("Excluding operation: %s (in ExcludedOperations list)\n", op.Name)
			excludedCount++
			continue
		}

		// Check if operation is excluded by config
		if config, ok := generator.OperationConfigs[op.Name]; ok && config.SkipGen {
			fmt.Printf("Skipping operation (configured in OperationConfigs): %s\n", op.Name)
			excludedCount++
			continue
		}

		// Check for duplicate Go function names
		if seenFunctions[op.GoName] {
			fmt.Printf("Skipping duplicate function: %s (from operation: %s)\n", op.GoName, op.Name)
			duplicateCount++
			continue
		}
		seenFunctions[op.GoName] = true

		filteredOps = append(filteredOps, op)
	}

	fmt.Printf("Filtered operations: %d excluded, %d duplicates, %d remaining\n",
		excludedCount, duplicateCount, len(filteredOps))

	return filteredOps
}

// UpdateImageInputOutputFlags examines operation arguments and sets proper flags
func (v *Introspection) UpdateImageInputOutputFlags(op *generator.Operation) {
	op.HasImageInput = false
	op.HasOneImageOutput = false
	op.HasArrayImageInput = false
	var imageOutputCount int

	// Check each argument to see if this operation takes/returns an image
	for _, arg := range op.Arguments {
		// Check for any input parameter with VipsImage* type
		if (arg.Type == "VipsImage" || arg.CType == "VipsImage*") && !arg.IsOutput {
			op.HasImageInput = true
		}

		// Check for "out" parameter with VipsImage* type
		if arg.Type == "VipsImage" && arg.CType == "VipsImage**" && arg.IsOutput {
			op.HasImageOutput = true
			imageOutputCount++
		}

		// Check for array image inputs
		if strings.HasPrefix(arg.GoType, "[]*C.VipsImage") ||
			(strings.Contains(arg.CType, "VipsImage**") && !arg.IsOutput) {
			op.HasArrayImageInput = true
		}

		if arg.CType == "void**" && arg.Name == "buf" {
			op.HasBufferOutput = true
		}
		if arg.CType == "void*" && arg.Name == "buf" {
			op.HasBufferInput = true
		}
	}
	if imageOutputCount == 1 {
		op.HasOneImageOutput = true
	}
	if op.Name == "copy" || op.Name == "sequential" || op.Name == "linecache" || op.Name == "tilecache" {
		// operations that should not mutate the Image object
		op.HasOneImageOutput = false
	}
}
