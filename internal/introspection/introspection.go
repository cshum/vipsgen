package introspection

// #cgo pkg-config: vips
// #include "introspection.h"
import "C"
import (
	"fmt"
	"log"
	"strings"
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

// Introspection provides discovery and analysis of libvips operations
// through reflection of the C library's type system, extracting operation
// metadata, argument details, and supported enum types.
type Introspection struct {
	discoveredEnumTypes  map[string]string
	enumTypeNames        []enumTypeName
	functionInfo         []VipsFunctionInfo
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

	// Initialize map with known enum types
	discoveredTypes := make(map[string]string)
	for _, enum := range baseEnumTypeNames {
		discoveredTypes[enum.CName] = enum.GoName
	}

	return &Introspection{
		discoveredEnumTypes:  discoveredTypes,
		discoveredImageTypes: map[string]ImageTypeInfo{},
		enumTypeNames:        baseEnumTypeNames,
	}
}

// FilterOperations filters operations based on availability in the current libvips installation,
// excluded operations list, and deduplicates by Go function name
func (v *Introspection) FilterOperations(operations []Operation) []Operation {
	// Filter out excluded operations and deduplicate by Go function name
	seenFunctions := make(map[string]bool)
	var filteredOps []Operation
	var excludedCount, duplicateCount int

	for _, op := range operations {
		if strings.Contains(op.Name, "_source") || strings.Contains(op.Name, "_target") ||
			strings.Contains(op.Name, "_mime") {
			fmt.Printf("Excluding operation: %s \n", op.Name)
			excludedCount++
			continue
		}

		// Check if operation is explicitly excluded
		if ExcludedOperations[op.Name] {
			fmt.Printf("Excluding operation: %s (in ExcludedOperations list)\n", op.Name)
			excludedCount++
			continue
		}

		// Check if operation is excluded by config
		if config, ok := OperationConfigs[op.Name]; ok && config.SkipGen {
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

	fmt.Printf("Filtered operations: %d (%d excluded, %d duplicates)\n",
		len(filteredOps), excludedCount, duplicateCount)

	return filteredOps
}
