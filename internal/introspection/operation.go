package introspection

// #include "introspection.h"
import "C"
import (
	"encoding/json"
	"fmt"
	"github.com/cshum/vipsgen/internal/generator"
	"log"
	"os"
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

		// Create the Go operation structure
		op := generator.Operation{
			Name:               name,
			GoName:             FormatGoFunctionName(name),
			Description:        C.GoString(cOp.description),
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
					if arg.Required {
						op.Arguments = append(op.Arguments, arg)
						op.RequiredInputs = append(op.RequiredInputs, arg)
					} else {
						op.OptionalInputs = append(op.OptionalInputs, arg)
					}
				} else if arg.IsOutput {
					if arg.Required {
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

		operations = append(operations, op)
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

// GetOperationArguments uses GObject introspection to extract all arguments for an operation
func (v *Introspection) GetOperationArguments(opName string) ([]generator.Argument, error) {
	cOpName := C.CString(opName)
	defer C.free(unsafe.Pointer(cOpName))

	var nArgs C.int
	argsPtr := C.get_operation_arguments(cOpName, &nArgs)
	if argsPtr == nil || nArgs == 0 {
		return nil, fmt.Errorf("operation %s not found or has no arguments", opName)
	}
	defer C.free_operation_arguments(argsPtr, nArgs)

	// Convert C array to Go slice
	argsSlice := (*[1 << 30]C.ArgInfo)(unsafe.Pointer(argsPtr))[:nArgs:nArgs]
	var goArgs []generator.Argument

	for i := 0; i < int(nArgs); i++ {
		arg := argsSlice[i]

		// Extract argument information
		name := C.GoString(arg.name)
		description := C.GoString(arg.blurb)

		// Get type name using our helper function
		cTypeNamePtr := C.get_type_name(arg.type_val)
		cTypeName := C.GoString(cTypeNamePtr)

		isInput := int(arg.is_input) != 0
		isOutput := int(arg.is_output) != 0
		required := int(arg.required) != 0
		hasDefault := int(arg.has_default) != 0

		// Create the Go argument structure
		goArg := generator.Argument{
			Name:        FormatIdentifier(name),
			GoName:      FormatGoIdentifier(name),
			Description: description,
			Required:    required,
			IsInput:     isInput,
			IsOutput:    isOutput,
			Flags:       int(arg.flags),
		}

		// Determine Go type and C type based on GType
		goArg.Type, goArg.GoType, goArg.CType = v.mapGTypeToTypes(arg.type_val, cTypeName, isOutput)

		// Extract default value if present
		if hasDefault {
			goArg.DefaultValue = v.extractDefaultValue(arg, goArg.GoType)
		}

		// Check if this is an enum type
		if C.is_type_enum(arg.type_val) != 0 {
			goArg.IsEnum = true
			goArg.EnumType = v.GetGoEnumName(goArg.Type)

			// Register the enum type
			v.AddEnumType(goArg.Type, goArg.EnumType)
		}

		// Fix the vips_composite mode parameter - should be an array of BlendMode
		if opName == "composite" && name == "mode" && goArg.CType == "int*" && goArg.GoType == "[]int" {
			// Update to array of BlendMode
			goArg.GoType = "[]BlendMode"
			goArg.IsEnum = true
			goArg.EnumType = "BlendMode"
		}

		goArgs = append(goArgs, goArg)
	}

	return goArgs, nil
}

// Helper function to extract default values based on type
func (v *Introspection) extractDefaultValue(arg C.ArgInfo, goType string) interface{} {
	// Check if there's a default value
	if int(arg.has_default) == 0 {
		return nil
	}

	// Extract based on the default type
	switch int(arg.default_type) {
	case 1: // bool
		return int(arg.bool_default) != 0
	case 2: // int
		return int(arg.int_default)
	case 3: // double
		return float64(arg.double_default)
	case 4: // string
		if arg.string_default != nil {
			return C.GoString(arg.string_default)
		}
		return ""
	default:
		return nil
	}
}

// mapGTypeToTypes maps a GType to Go and C types
func (v *Introspection) mapGTypeToTypes(gtype C.GType, typeName string, isOutput bool) (baseType, goType, cType string) {
	// First check known vips types (simplified for brevity)
	switch {
	case cTypeCheck(gtype, "VipsImage"):
		if isOutput {
			return "VipsImage", "*C.VipsImage", "VipsImage**"
		}
		return "VipsImage", "*C.VipsImage", "VipsImage*"
	case cTypeCheck(gtype, "VipsArrayInt"):
		return "VipsArrayInt", "[]int", "int*"
	case cTypeCheck(gtype, "VipsArrayDouble"):
		return "VipsArrayDouble", "[]float64", "double*"
	case cTypeCheck(gtype, "VipsArrayImage"):
		return "VipsArrayImage", "[]*C.VipsImage", "VipsImage**"
	case cTypeCheck(gtype, "VipsBlob"):
		return "VipsBlob", "[]byte", "void*"
	}

	// Check for known GLib types
	switch {
	case cTypeCheck(gtype, "gboolean"):
		return "gboolean", "bool", "gboolean"
	case cTypeCheck(gtype, "gint"):
		return "gint", "int", "int"
	case cTypeCheck(gtype, "guint"):
		return "guint", "int", "unsigned int"
	case cTypeCheck(gtype, "gint64"):
		return "gint64", "int64", "gint64"
	case cTypeCheck(gtype, "guint64"):
		return "guint64", "uint64", "guint64"
	case cTypeCheck(gtype, "gdouble"):
		return "gdouble", "float64", "double"
	case cTypeCheck(gtype, "gfloat"):
		return "gfloat", "float32", "float"
	case cTypeCheck(gtype, "gchararray"):
		return "gchararray", "string", "const char*"
	}

	// Check for enum/flags
	if C.is_type_enum(gtype) != 0 {
		goEnumName := v.GetGoEnumName(typeName)
		return typeName, goEnumName, typeName
	}
	if C.is_type_flags(gtype) != 0 {
		goEnumName := v.GetGoEnumName(typeName)
		return typeName, goEnumName, typeName
	}

	// Default fallback
	return typeName, "interface{}", "void*"
}

// Helper function to check type names
func cTypeCheck(gtype C.GType, name string) bool {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	cTypeNamePtr := C.get_type_name(gtype)
	if cTypeNamePtr == nil {
		return false
	}

	cTypeName := C.GoString(cTypeNamePtr)
	return cTypeName == name
}
