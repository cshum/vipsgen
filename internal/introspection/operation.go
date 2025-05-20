package introspection

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
	HasThisImageInput  bool
	HasImageOutput     bool
	HasOneImageOutput  bool
	HasBufferInput     bool
	HasBufferOutput    bool
	HasArrayImageInput bool
	ImageTypeString    string
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
	IsInputN     bool
	IsOutput     bool
	IsOutputN    bool
	IsSource     bool
	IsImage      bool
	IsBuffer     bool
	IsArray      bool
	Flags        int
	IsEnum       bool
	EnumType     string
	NInputFrom   string
	DefaultValue interface{}
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
			GoName:             formatGoFunctionName(name),
			Description:        description,
			HasThisImageInput:  int(details.has_this_image_input) != 0,
			HasImageOutput:     int(details.has_image_output) != 0,
			HasOneImageOutput:  int(details.has_one_image_output) != 0,
			HasBufferInput:     int(details.has_buffer_input) != 0,
			HasBufferOutput:    int(details.has_buffer_output) != 0,
			HasArrayImageInput: int(details.has_array_image_input) != 0,
			ImageTypeString:    v.determineImageTypeStringFromOperation(name),
		}

		v.discoverEnumsFromOperation(name)

		// Get all arguments
		args, err := v.DiscoverOperationArguments(name)
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

		if strings.Contains(op.Name, "_target") ||
			strings.Contains(op.Name, "_mime") ||
			strings.Contains(op.Name, "fitsload_source") {
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

// DiscoverOperationArguments uses GObject introspection to extract all arguments for an operation
func (v *Introspection) DiscoverOperationArguments(opName string) ([]Argument, error) {
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
	var goArgs []Argument

	// Detect if we need to add an 'n' parameter
	hasArrayInput := -1
	hasArrayNOutput := -1

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
		isImage := int(arg.is_image) != 0
		isBuffer := int(arg.is_buffer) != 0
		isArray := int(arg.is_array) != 0
		isSource := cTypeCheck(arg.type_val, "VipsSource")

		// Create the Go argument structure
		goArg := Argument{
			Name:        formatIdentifier(name),
			GoName:      formatGoIdentifier(name),
			Description: description,
			IsRequired:  required,
			IsInput:     isInput,
			IsOutput:    isOutput,
			IsImage:     isImage,
			IsBuffer:    isBuffer,
			IsArray:     isArray,
			IsSource:    isSource,
			Flags:       int(arg.flags),
		}

		// Check if this is an enum or flags type
		isEnum := C.is_type_enum(arg.type_val) != 0
		isFlags := C.is_type_flags(arg.type_val) != 0

		// Set IsEnum if either enum or flags type
		goArg.IsEnum = isEnum || isFlags

		// Determine Go type and C type based on GType
		goArg.Type, goArg.GoType, goArg.CType = v.mapGTypeToTypes(arg.type_val, cTypeName, isOutput)

		// Determine a special case for affine matrix
		isAffineMatrix := goArg.Name == "matrix" && goArg.IsArray && goArg.IsRequired && goArg.IsInput

		// Extract default value if present
		if hasDefault {
			goArg.DefaultValue = v.extractDefaultValue(arg, goArg.GoType)
		}

		// If it's an enum or flags, get the proper type name
		if goArg.IsEnum {
			enumName := C.GoString(C.g_type_name(arg.type_val))
			goArg.EnumType = v.getGoEnumName(enumName)
			v.addEnumType(enumName, goArg.EnumType)
		}
		if isArray && isInput && required && !isAffineMatrix {
			hasArrayInput = i
		}
		if (isArray || (hasArrayInput >= 0 && isImage)) && isOutput && required {
			hasArrayNOutput = i
		}

		// Fix the vips_composite mode parameter - should be an array of BlendMode
		if opName == "composite" && name == "mode" && goArg.CType == "int*" && goArg.GoType == "[]int" {
			// Update to array of BlendMode
			goArg.GoType = "[]BlendMode"
			goArg.IsEnum = true
			goArg.EnumType = "BlendMode"
		}

		// special case: affine operation to use individual parameters
		if isAffineMatrix {
			aArg := Argument{
				Name:        "a",
				GoName:      "a",
				Type:        "gdouble",
				GoType:      "float64",
				CType:       "double",
				Description: "Coefficient a (horizontal scale)",
				IsRequired:  true,
				IsInput:     true,
				IsOutput:    false,
				Flags:       19, // VIPS_ARGUMENT_REQUIRED | VIPS_ARGUMENT_INPUT
			}
			bArg := Argument{
				Name:        "b",
				GoName:      "b",
				Type:        "gdouble",
				GoType:      "float64",
				CType:       "double",
				Description: "Coefficient b (horizontal shear)",
				IsRequired:  true,
				IsInput:     true,
				IsOutput:    false,
				Flags:       19,
			}
			cArg := Argument{
				Name:        "c",
				GoName:      "c",
				Type:        "gdouble",
				GoType:      "float64",
				CType:       "double",
				Description: "Coefficient c (vertical shear)",
				IsRequired:  true,
				IsInput:     true,
				IsOutput:    false,
				Flags:       19,
			}
			dArg := Argument{
				Name:        "d",
				GoName:      "d",
				Type:        "gdouble",
				GoType:      "float64",
				CType:       "double",
				Description: "Coefficient d (vertical scale)",
				IsRequired:  true,
				IsInput:     true,
				IsOutput:    false,
				Flags:       19,
			}
			goArgs = append(goArgs, aArg, bArg, cArg, dArg)
			continue
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
				if arg.IsBuffer && arg.IsInput {
					hasBufParam = true
				}
				if arg.Name == "len" && arg.IsInput {
					hasLenParam = true
				}
			}

			// If we have an input buffer but no length parameter, add one
			if hasBufParam && !hasLenParam {
				lenParam := Argument{
					Name:        "len",
					GoName:      "len",
					Type:        "gsize",
					GoType:      "int",
					CType:       "size_t",
					Description: "Size of buffer in bytes",
					IsRequired:  true,
					IsInput:     true,
					IsOutput:    false,
					Flags:       19, // VIPS_ARGUMENT_REQUIRED | VIPS_ARGUMENT_INPUT
				}

				// Insert the length parameter right after the buffer parameter
				newArgs := make([]Argument, 0, len(goArgs)+1)
				bufIndex := -1

				for i, arg := range goArgs {
					newArgs = append(newArgs, arg)
					if arg.IsBuffer && arg.IsInput {
						bufIndex = i
					}
				}

				if bufIndex >= 0 {
					// Insert len parameter after buf parameter
					newArgs = append(newArgs[:bufIndex+1], append([]Argument{lenParam}, newArgs[bufIndex+1:]...)...)
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
				if arg.IsBuffer && arg.IsOutput {
					hasBufParam = true
					goArgs[i].CType = "void**"
				}
				if arg.Name == "len" {
					hasLenParam = true
					goArgs[i].CType = "size_t*"
				}
			}
			// If we have a buf parameter but no len parameter, add one
			if hasBufParam && !hasLenParam {
				lenParam := Argument{
					Name:        "len",
					GoName:      "len",
					Type:        "gsize",
					GoType:      "int",
					CType:       "size_t*",
					Description: "Size of output buffer in bytes",
					IsRequired:  true,
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
	if hasArrayNOutput >= 0 || hasArrayInput >= 0 {
		i := hasArrayInput + 1
		if hasArrayNOutput >= 0 {
			i = hasArrayNOutput + 1
		}
		if i > len(goArgs) {
			i = len(goArgs)
		}
		var nFrom string
		if hasArrayInput >= 0 && hasArrayInput < len(goArgs) && goArgs[hasArrayInput].IsArray {
			nFrom = goArgs[hasArrayInput].Name
		}
		var nParam Argument
		if hasArrayNOutput >= 0 && !goArgs[hasArrayNOutput].IsImage {
			// output 'n' parameter for getpoint
			nParam = Argument{
				Name:        "n",
				GoName:      "n",
				Type:        "gint",
				GoType:      "int",
				CType:       "int*",
				Description: "Length of output array",
				IsRequired:  true,
				IsInput:     false,
				IsOutput:    true,
				IsOutputN:   true,
				Flags:       35, // VIPS_ARGUMENT_REQUIRED | VIPS_ARGUMENT_OUTPUT
			}
		} else {
			// input 'n' parameter for array operations like linear, remainder_const, etc.
			nParam = Argument{
				Name:        "n",
				GoName:      "n",
				Type:        "gint",
				GoType:      "int",
				CType:       "int",
				Description: "Array length",
				IsRequired:  true, // IsRequired for input arrays in most cases
				NInputFrom:  nFrom,
				IsInput:     true,
				IsInputN:    true,
				IsOutput:    false,
				Flags:       19, // VIPS_ARGUMENT_REQUIRED | VIPS_ARGUMENT_INPUT
			}
		}
		goArgs = append(goArgs[0:i], append([]Argument{nParam}, goArgs[i:]...)...)
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
	// Special case for VipsSource - map to VipsSourceCustom for proper compatibility
	if cTypeCheck(gtype, "VipsSource") {
		// For VipsSource, we want to use VipsSourceCustom in the bindings
		if isOutput {
			return "VipsSourceCustom", "*C.VipsSourceCustom", "VipsSourceCustom**"
		}
		return "VipsSourceCustom", "*C.VipsSourceCustom", "VipsSourceCustom*"
	}
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

	// Special case for VipsInterpolate
	if cTypeCheck(gtype, "VipsInterpolate") {
		if isOutput {
			return "VipsInterpolate", "*Interpolate", "VipsInterpolate**"
		}
		return "VipsInterpolate", "*Interpolate", "VipsInterpolate*"
	}

	// Special case for VipsBlob which needs special output handling
	if cTypeCheck(gtype, "VipsBlob") {
		if isOutput {
			return "VipsBlob", "[]byte", "VipsBlob**"
		}
		return "VipsBlob", "[]byte", "void*"
	}

	switch {
	case cTypeCheck(gtype, "VipsArrayInt"):
		return "VipsArrayInt", "[]int", addOutputPointer("int*", isOutput)
	case cTypeCheck(gtype, "VipsArrayDouble"):
		return "VipsArrayDouble", "[]float64", addOutputPointer("double*", isOutput)
	case cTypeCheck(gtype, "VipsArrayImage"):
		return "VipsArrayImage", "[]*C.VipsImage", "VipsImage**"
	}

	// Check if this is an object type (not just VipsImage and VipsInterpolate)
	if C.g_type_is_a(gtype, C.g_type_from_name(C.CString("GObject"))) != 0 {
		// Get the actual type name
		cTypeNamePtr := C.g_type_name(gtype)
		if cTypeNamePtr != nil {
			actualTypeName := C.GoString(cTypeNamePtr)

			// Log for debugging
			log.Printf("Found object type: %s", actualTypeName)

			if isOutput {
				return actualTypeName, "*C." + actualTypeName, actualTypeName + "**"
			}
			return actualTypeName, "*C." + actualTypeName, actualTypeName + "*"
		}
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
		goEnumName := v.getGoEnumName(typeName)
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

// checkOperationExists checks if a libvips operation exists
func (v *Introspection) checkOperationExists(name string) bool {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	// Try to create the operation - if it succeeds, the operation exists
	vop := C.vips_operation_new(cName)
	if vop == nil {
		return false
	}

	// Clean up and return true
	C.g_object_unref(C.gpointer(vop))
	return true
}
