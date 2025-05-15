package introspection

// #include "introspection.h"
import "C"
import (
	"encoding/json"
	"github.com/cshum/vipsgen/internal/generator"
	"log"
	"os"
	"strings"
	"unicode"
	"unsafe"
)

// DiscoverOperations uses GObject introspection to discover all available operations
func (v *Introspection) DiscoverOperations() []generator.Operation {
	var nOps C.int
	opsPtr := C.get_all_operations(&nOps)
	if opsPtr == nil || nOps == 0 {
		return nil
	}
	defer C.free_operation_info(opsPtr, nOps)

	// Convert C array to Go slice
	opsSlice := (*[1 << 30]C.OperationInfo)(unsafe.Pointer(opsPtr))[:nOps:nOps]
	var operations []generator.Operation

	for i := 0; i < int(nOps); i++ {
		op := opsSlice[i]
		name := C.GoString(op.name)

		// Skip deprecated operations
		if (op.flags & C.VIPS_OPERATION_DEPRECATED) != 0 {
			continue
		}

		// Get detailed operation information
		opName := C.CString(name)
		details := C.get_operation_details(opName)
		C.free(unsafe.Pointer(opName))

		// Create the Go operation structure
		goOp := generator.Operation{
			Name:               name,
			GoName:             FormatGoFunctionName(name),
			Description:        C.GoString(op.description),
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

		// Get all arguments
		args, err := v.GetOperationArguments(name)
		if err == nil {

			// Categorize arguments
			for _, arg := range args {
				if arg.IsInput {
					if arg.Required {
						goOp.Arguments = append(goOp.Arguments, arg)
						goOp.RequiredInputs = append(goOp.RequiredInputs, arg)
					} else {
						goOp.OptionalInputs = append(goOp.OptionalInputs, arg)
					}
				} else if arg.IsOutput {
					goOp.Outputs = append(goOp.Outputs, arg)
					goOp.Arguments = append(goOp.Arguments, arg)
				}
			}
		}

		v.FixOperationTypes(&goOp)

		// Update image input/output flags
		v.UpdateImageInputOutputFlags(&goOp)

		operations = append(operations, goOp)
	}

	// Debug: Write the parsed GIR to a JSON file
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

func (v *Introspection) isEnumType(cType string) bool {
	return v.discoveredEnumTypes[strings.ToLower(cType)] != ""
}

// determineFlags calculates the flags for an argument
func determineFlags(isOutput bool, isRequired bool) int {
	if isOutput && isRequired {
		return 35 // VIPS_ARGUMENT_REQUIRED | VIPS_ARGUMENT_OUTPUT
	} else if isOutput && !isRequired {
		return 33 // VIPS_ARGUMENT_OPTIONAL | VIPS_ARGUMENT_OUTPUT
	} else if !isOutput && isRequired {
		return 19 // VIPS_ARGUMENT_REQUIRED | VIPS_ARGUMENT_INPUT
	} else {
		return 17 // VIPS_ARGUMENT_OPTIONAL | VIPS_ARGUMENT_INPUT
	}
}

// FormatGoIdentifier formats a name to a go identifier
func FormatGoIdentifier(name string) string {
	s := generator.SnakeToCamel(FormatIdentifier(name))

	// first letter lower case
	if len(s) == 0 {
		return s
	}

	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// FormatIdentifier formats a name to an identifier
func FormatIdentifier(name string) string {
	// Handle Go keywords
	switch name {
	case "type", "func", "map", "range", "select", "case", "default":
		return name + "_"
	}

	// Handle special cases
	name = strings.Replace(name, "-", "_", -1)

	return name
}

// FormatEnumValueName converts a C enum name to a Go name
func FormatEnumValueName(typeName, valueName string) string {
	// Convert to CamelCase
	camelValue := generator.SnakeToCamel(strings.ToLower(valueName))

	// Check if the value already contains "Vips" + typeName or "VipsForeign" + typeName
	lowerCamelValue := strings.ToLower(camelValue)
	lowerTypeName := strings.ToLower(typeName)

	if strings.HasPrefix(lowerCamelValue, "vips"+lowerTypeName) ||
		strings.HasPrefix(lowerCamelValue, "vipsforeign"+lowerTypeName) {
		return GetGoEnumName(camelValue)
	}

	// Otherwise, prepend the type name
	return typeName + GetGoEnumName(camelValue)
}

// GetGoEnumName converts a C enum type name to a Go type name
func GetGoEnumName(cName string) string {
	// Strip "Vips" prefix if present
	if strings.HasPrefix(cName, "Vips") {
		cName = cName[4:]
	}

	// Also strip "Foreign" prefix if present in both the original name
	// and after removing "Vips" prefix
	if strings.HasPrefix(cName, "Foreign") {
		cName = cName[7:]
	}

	return cName
}

// FormatGoFunctionName formats an operation name to a Go function name
func FormatGoFunctionName(name string) string {
	// Convert operation names to match existing Go function style
	// e.g., "extract_area" -> "ExtractArea"
	parts := strings.Split(name, "_")

	// Convert each part to title case
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[0:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}
