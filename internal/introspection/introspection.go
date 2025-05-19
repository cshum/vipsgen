package introspection

// #cgo pkg-config: vips
// #include "introspection.h"
import "C"
import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"unsafe"
)

// Operation represents a libvips operation
type Operation struct {
	Name               string
	GoName             string
	Description        string
	Arguments          []Argument
	RequiredInputs     []Argument
	OptionalInputs     []Argument
	RequiredOutputs    []Argument
	OptionalOutputs    []Argument
	HasImageInput      bool
	HasImageOutput     bool
	HasOneImageOutput  bool
	HasBufferInput     bool
	HasBufferOutput    bool
	HasArrayImageInput bool
	ImageTypeString    string
	Category           string // arithmetic, conversion, etc
}

// Argument represents an argument to a libvips operation
type Argument struct {
	Name         string
	GoName       string
	Type         string
	GoType       string
	CType        string
	Description  string
	IsRequired   bool
	IsInput      bool
	IsNInput     bool
	IsOutput     bool
	IsImage      bool
	IsBuffer     bool
	IsArray      bool
	Flags        int
	IsEnum       bool
	EnumType     string
	NInputFrom   string
	DefaultValue interface{}
}

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

// DiscoverOperations uses GObject introspection to discover all available operations
func (v *Introspection) DiscoverOperations() []Operation {
	var nOps C.int
	opsPtr := C.get_all_operations(&nOps)
	if opsPtr == nil || nOps == 0 {
		return nil
	}
	defer C.free_operation_info(opsPtr, nOps)

	// Convert C array to Go slice
	opsSlice := (*[1 << 30]C.OperationInfo)(unsafe.Pointer(opsPtr))[:nOps:nOps]
	var operations []Operation

	seenOperations := make(map[string]bool)
	var excludedCount, duplicateCount int

	for i := 0; i < int(nOps); i++ {
		cOp := opsSlice[i]
		name := C.GoString(cOp.name)

		// Skip deprecated operations
		if (cOp.flags & C.VIPS_OPERATION_DEPRECATED) != 0 {
			continue
		}

		// Get detailed operation information
		opName := C.CString(name)
		details := C.get_operation_details(opName)
		C.free(unsafe.Pointer(opName))

		description := fmt.Sprintf("vips_%s ", name) + C.GoString(cOp.description)

		// Create the Go operation structure
		op := Operation{
			Name:               name,
			GoName:             FormatGoFunctionName(name),
			Description:        description,
			HasImageInput:      int(details.has_image_input) != 0,
			HasImageOutput:     int(details.has_image_output) != 0,
			HasOneImageOutput:  int(details.has_one_image_output) != 0,
			HasBufferInput:     int(details.has_buffer_input) != 0,
			HasBufferOutput:    int(details.has_buffer_output) != 0,
			HasArrayImageInput: int(details.has_array_image_input) != 0,
			Category:           C.GoString(details.category),
			ImageTypeString:    v.DetermineImageTypeStringFromOperation(name),
		}

		if details.category != nil {
			C.free(unsafe.Pointer(details.category))
		}

		v.DiscoverEnumsFromOperation(name)

		// Get all arguments
		args, err := v.GetOperationArguments(name)
		if err == nil {

			// Categorize arguments
			for _, arg := range args {
				if arg.IsInput {
					if arg.IsRequired {
						op.Arguments = append(op.Arguments, arg)
						op.RequiredInputs = append(op.RequiredInputs, arg)
					} else {
						op.OptionalInputs = append(op.OptionalInputs, arg)
					}
				} else if arg.IsOutput {
					if arg.IsRequired {
						op.Arguments = append(op.Arguments, arg)
						op.RequiredOutputs = append(op.RequiredOutputs, arg)
					} else {
						op.OptionalOutputs = append(op.OptionalOutputs, arg)
					}
				}
			}
		}

		if op.Name == "copy" || op.Name == "sequential" || op.Name == "linecache" || op.Name == "tilecache" {
			// operations that should not mutate the Image object
			op.HasOneImageOutput = false
		}
		if strings.Contains(op.Name, "_source") || strings.Contains(op.Name, "_target") ||
			strings.Contains(op.Name, "_mime") {
			fmt.Printf("Excluded operation: vips_%s \n", op.Name)
			excludedCount++
			continue
		}
		// Check for duplicate Go function names
		if seenOperations[op.GoName] {
			fmt.Printf("Skipping duplicated operation: vips_%s\n", op.Name)
			duplicateCount++
			continue
		}
		seenOperations[op.GoName] = true

		fmt.Printf("Discovered operation: vips_%s \n", op.Name)
		operations = append(operations, op)
	}
	fmt.Printf("Discovered Operations: %d (%d excluded, %d duplicates)\n",
		len(operations), excludedCount, duplicateCount)

	// Debug: Write operations object to a JSON file
	jsonData, err := json.MarshalIndent(operations, "", "  ")
	if err != nil {
		log.Printf("Warning: failed to marshal operations to JSON: %v", err)
	} else {
		err = os.WriteFile("debug_operations.json", jsonData, 0644)
		if err != nil {
			log.Printf("Warning: failed to write debug_operations.json: %v", err)
		} else {
			log.Println("Wrote introspected operations to debug_operations.json")
		}
	}

	return operations
}
