package introspection

// #include "introspection.h"
import "C"
import (
	"fmt"
	"github.com/cshum/vipsgen/internal/generator"
	"unsafe"
)

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
		goArg.Type, goArg.GoType, goArg.CType = v.mapGTypeToTypes(arg.type_val, cTypeName)

		// Check if this is an enum type
		if C.is_type_enum(arg.type_val) != 0 {
			goArg.IsEnum = true
			goArg.EnumType = v.GetGoEnumName(goArg.Type)

			// Register the enum type
			v.AddEnumType(goArg.Type, goArg.EnumType)
		}

		goArgs = append(goArgs, goArg)
	}

	return goArgs, nil
}

// mapGTypeToTypes maps a GType to Go and C types
func (v *Introspection) mapGTypeToTypes(gtype C.GType, typeName string) (baseType, goType, cType string) {
	// First check known vips types (simplified for brevity)
	switch {
	case cTypeCheck(gtype, "VipsImage"):
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

// GetOperationOptionalArgs extracts only the optional arguments for an operation
func (v *Introspection) GetOperationOptionalArgs(opName string) ([]generator.Argument, error) {
	allArgs, err := v.GetOperationArguments(opName)
	if err != nil {
		return nil, err
	}

	// Filter for optional input arguments only
	var optionalArgs []generator.Argument
	for _, arg := range allArgs {
		if !arg.Required && arg.IsInput {
			optionalArgs = append(optionalArgs, arg)
		}
	}

	return optionalArgs, nil
}

// ExtractOptionalArgsFromIntrospection is a replacement for extractOptionalArgsFromDoc
// that uses direct GObject introspection instead of parsing docs
func (v *Introspection) ExtractOptionalArgsFromIntrospection(opName string) []generator.Argument {
	// Get optional arguments directly from introspection
	optionalArgs, err := v.GetOperationOptionalArgs(opName)
	if err != nil {
		fmt.Printf("Error extracting optional args for %s: %v\n", opName, err)
		return nil
	}

	return optionalArgs
}
