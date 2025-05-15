package introspection

// #include "introspection.h"
import "C"
import (
	"encoding/json"
	"fmt"
	"github.com/cshum/vipsgen/internal/generator"
	"log"
	"os"
	"strings"
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

	// First pass: gather arguments and detect if we need to add an 'n' parameter
	hasNParam := false
	hasArrayInput := false
	hasArrayOutput := false

	// Special cases for operations with known output array parameters
	isOutputArrayOp := opName == "getpoint" || opName == "getpoint_interpolate" || opName == "profile_load"

	for i := 0; i < int(nArgs); i++ {
		arg := argsSlice[i]
		name := C.GoString(arg.name)

		// Check if we already have an 'n' parameter
		if name == "n" {
			hasNParam = true
		}

		// Get type name and check if this is an array type
		cTypeNamePtr := C.get_type_name(arg.type_val)
		cTypeName := C.GoString(cTypeNamePtr)
		isInput := int(arg.is_input) != 0
		isOutput := int(arg.is_output) != 0

		// Determine C type to check for array types
		_, _, cType := v.mapGTypeToTypes(arg.type_val, cTypeName, isOutput)

		// Check if this is an array input parameter (double*, int*, etc.)
		if isInput && !isOutput && strings.HasSuffix(cType, "*") &&
			cType != "VipsImage*" && cType != "const char*" {
			hasArrayInput = true
		}

		// Check if this is an array output parameter (for getpoint, etc.)
		if !isInput && isOutput && (name == "out_array" || name == "vector") {
			hasArrayOutput = true
		}
	}

	// Second pass: create Go arguments and add 'n' parameter if needed
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

	// Special case: handle buffer operations
	if strings.Contains(opName, "_buffer") {
		if strings.HasSuffix(opName, "load_buffer") || strings.HasSuffix(opName, "thumbnail_buffer") {
			// INPUT buffer operations - add length parameter for input buffer
			hasBufParam := false
			hasLenParam := false

			for _, arg := range goArgs {
				if (arg.Name == "buf" || arg.Name == "buffer") && arg.CType == "void*" && arg.IsInput {
					hasBufParam = true
				}
				if arg.Name == "len" && arg.IsInput {
					hasLenParam = true
				}
			}

			// If we have an input buffer but no length parameter, add one
			if hasBufParam && !hasLenParam {
				lenParam := generator.Argument{
					Name:        "len",
					GoName:      "len",
					Type:        "gsize",
					GoType:      "int",
					CType:       "size_t",
					Description: "Size of buffer in bytes",
					Required:    true,
					IsInput:     true,
					IsOutput:    false,
					Flags:       19, // VIPS_ARGUMENT_REQUIRED | VIPS_ARGUMENT_INPUT
				}

				// Insert the length parameter right after the buffer parameter
				newArgs := make([]generator.Argument, 0, len(goArgs)+1)
				bufIndex := -1

				for i, arg := range goArgs {
					newArgs = append(newArgs, arg)
					if (arg.Name == "buf" || arg.Name == "buffer") && arg.CType == "void*" && arg.IsInput {
						bufIndex = i
					}
				}

				if bufIndex >= 0 {
					// Insert len parameter after buf parameter
					newArgs = append(newArgs[:bufIndex+1], append([]generator.Argument{lenParam}, newArgs[bufIndex+1:]...)...)
				} else {
					// Fallback: just append at the end
					newArgs = append(newArgs, lenParam)
				}

				goArgs = newArgs
			}
		} else if strings.HasSuffix(opName, "save_buffer") {
			// OUTPUT buffer operations - ensure buf and len are output params
			hasBufParam := false
			hasLenParam := false

			for i, arg := range goArgs {
				// Fix buf parameter if it exists
				if arg.Name == "buf" || arg.Name == "buffer" {
					hasBufParam = true
					// Ensure it's an output parameter with the right type
					goArgs[i].IsInput = false
					goArgs[i].IsOutput = true
					goArgs[i].CType = "void**"
				}

				// Check for len parameter
				if arg.Name == "len" {
					hasLenParam = true
					// Ensure it's an output parameter with the right type
					goArgs[i].IsInput = false
					goArgs[i].IsOutput = true
					goArgs[i].CType = "size_t*"
				}
			}

			// If we have a buf parameter but no len parameter, add one
			if hasBufParam && !hasLenParam {
				lenParam := generator.Argument{
					Name:        "len",
					GoName:      "len",
					Type:        "gsize",
					GoType:      "int",
					CType:       "size_t*",
					Description: "Size of output buffer in bytes",
					Required:    true,
					IsInput:     false,
					IsOutput:    true,
					Flags:       35, // VIPS_ARGUMENT_REQUIRED | VIPS_ARGUMENT_OUTPUT
				}

				// Add len parameter
				goArgs = append(goArgs, lenParam)
			}
		}
	}

	// Special case: Add the missing 'n' parameter if needed
	if !hasNParam {
		// Special cases for operations with output arrays
		if hasArrayOutput || isOutputArrayOp {
			// Add output 'n' parameter
			nParam := generator.Argument{
				Name:        "n",
				GoName:      "n",
				Type:        "gint",
				GoType:      "int",
				CType:       "int*",
				Description: "Length of output array",
				Required:    true,
				IsInput:     false,
				IsOutput:    true,
				Flags:       35, // VIPS_ARGUMENT_REQUIRED | VIPS_ARGUMENT_OUTPUT
			}
			goArgs = append(goArgs, nParam)
		} else if hasArrayInput {
			// Add input 'n' parameter for array operations like linear, remainder_const, etc.
			nParam := generator.Argument{
				Name:         "n",
				GoName:       "n",
				Type:         "gint",
				GoType:       "int",
				CType:        "int",
				Description:  "Array length",
				Required:     true, // Required for input arrays in most cases
				IsInput:      true,
				IsOutput:     false,
				Flags:        19, // VIPS_ARGUMENT_REQUIRED | VIPS_ARGUMENT_INPUT
				DefaultValue: 1,  // Default to 1 element
			}
			goArgs = append(goArgs, nParam)
		}
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
	// Special case for VipsImage which has a different pointer pattern
	if cTypeCheck(gtype, "VipsImage") {
		if isOutput {
			return "VipsImage", "*C.VipsImage", "VipsImage**"
		}
		return "VipsImage", "*C.VipsImage", "VipsImage*"
	}

	// Handle output array parameters (vector, out_array)
	if isOutput {
		if cTypeCheck(gtype, "VipsArrayDouble") {
			return "VipsArrayDouble", "[]float64", "double**"
		} else if cTypeCheck(gtype, "VipsArrayInt") {
			return "VipsArrayInt", "[]int", "int**"
		}
	}

	// Handle other common vips array types
	switch {
	case cTypeCheck(gtype, "VipsArrayInt"):
		return "VipsArrayInt", "[]int", addOutputPointer("int*", isOutput)
	case cTypeCheck(gtype, "VipsArrayDouble"):
		return "VipsArrayDouble", "[]float64", addOutputPointer("double*", isOutput)
	case cTypeCheck(gtype, "VipsArrayImage"):
		return "VipsArrayImage", "[]*C.VipsImage", "VipsImage**"
	case cTypeCheck(gtype, "VipsBlob"):
		return "VipsBlob", "[]byte", addOutputPointer("void*", isOutput)
	}

	// Map basic scalar types
	var baseMap = map[string]struct {
		baseType string
		goType   string
		cType    string
	}{
		"gboolean":   {"gboolean", "bool", "gboolean"},
		"gint":       {"gint", "int", "int"},
		"guint":      {"guint", "int", "unsigned int"},
		"gint64":     {"gint64", "int64", "gint64"},
		"guint64":    {"guint64", "uint64", "guint64"},
		"gdouble":    {"gdouble", "float64", "double"},
		"gfloat":     {"gfloat", "float32", "float"},
		"gchararray": {"gchararray", "string", "const char*"},
	}

	// Check for basic types
	for typeName, typeInfo := range baseMap {
		if cTypeCheck(gtype, typeName) {
			if isOutput {
				cType := typeInfo.cType
				// Special case for string
				if cType == "const char*" {
					cType = "char**"
				} else {
					cType = addAsterisk(cType)
				}
				return typeInfo.baseType, typeInfo.goType, cType
			}
			return typeInfo.baseType, typeInfo.goType, typeInfo.cType
		}
	}

	// Check for enum/flags
	if C.is_type_enum(gtype) != 0 || C.is_type_flags(gtype) != 0 {
		goEnumName := v.GetGoEnumName(typeName)
		if isOutput {
			return typeName, goEnumName, typeName + "*"
		}
		return typeName, goEnumName, typeName
	}

	// Default fallback
	if isOutput {
		return typeName, "interface{}", "void**"
	}
	return typeName, "interface{}", "void*"
}

// addAsterisk adds a * to the end of a type name if not already there
func addAsterisk(typeName string) string {
	if !strings.HasSuffix(typeName, "*") {
		return typeName + "*"
	}
	return typeName
}

// addOutputPointer adds an additional * for output parameters
func addOutputPointer(cType string, isOutput bool) string {
	if isOutput {
		return addAsterisk(cType)
	}
	return cType
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

// moveParamToEnd moves a parameter to the end of the argument list
func moveParamToEnd(args []generator.Argument, paramName string) {
	for i, arg := range args {
		if arg.Name == paramName {
			// Remove parameter from its current position
			param := args[i]
			copy(args[i:], args[i+1:])
			// Add it back at the end
			args[len(args)-1] = param
			break
		}
	}
}

// moveParamAfter moves a parameter to be after a specific parameter
func moveParamAfter(args []generator.Argument, paramToMove, afterParam string) {
	paramIndex := -1
	afterIndex := -1

	// Find the indices
	for i, arg := range args {
		if arg.Name == paramToMove {
			paramIndex = i
		}
		if arg.Name == afterParam {
			afterIndex = i
		}
	}

	// If both parameters found and they're not already in the right order
	if paramIndex != -1 && afterIndex != -1 && paramIndex != afterIndex+1 {
		// Save the parameter to move
		param := args[paramIndex]

		// Remove parameter from its current position
		if paramIndex < afterIndex {
			// Param is before the target position, adjust indices
			copy(args[paramIndex:afterIndex], args[paramIndex+1:afterIndex+1])
			args[afterIndex] = param
		} else {
			// Param is after the target position
			copy(args[afterIndex+2:paramIndex+1], args[afterIndex+1:paramIndex])
			args[afterIndex+1] = param
		}
	}
}
