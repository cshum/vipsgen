package introspection

import (
	"github.com/cshum/vipsgen"
	"strings"
)

func (v *Introspection) FixCType(cType string, paramName string, functionName string, isOutput bool) string {
	if paramName == "len" && strings.Contains(functionName, "save_buffer") {
		return "size_t*"
	}
	// Handle basic types first
	if cType == "utf8*" {
		return "const char*"
	}
	if cType == "Source*" {
		return "VipsSource*"
	}
	if cType == "Target*" {
		return "VipsTarget*"
	}
	if cType == "Blob*" {
		return "VipsBlob*"
	}

	// Check if it's an enum type with a pointer suffix
	baseType := strings.TrimSuffix(cType, "*")

	// First check if the type without Vips prefix is an enum
	if !strings.HasPrefix(baseType, "Vips") && strings.HasSuffix(cType, "*") {
		// Try with Vips prefix
		vipsBaseType := "Vips" + baseType
		if v.isEnumType(vipsBaseType) {
			return vipsBaseType // Return without the pointer
		}
	}

	// Next check if the base type itself is an enum
	if v.isEnumType(baseType) && strings.HasSuffix(cType, "*") {
		return baseType // Return without the pointer
	}

	return cType
}

func (v *Introspection) UpdateImageInputOutputFlags(op *vipsgen.Operation) {
	op.HasImageInput = false
	op.HasImageOutput = false

	// Check each argument to see if this operation takes/returns an image
	for _, arg := range op.Arguments {
		// Check for "in" parameter with VipsImage* type
		if arg.Name == "in" && (arg.Type == "VipsImage" || arg.CType == "VipsImage*") && !arg.IsOutput {
			op.HasImageInput = true
		}

		// Check for "out" parameter with VipsImage* type
		if arg.Name == "out" && (arg.Type == "VipsImage" || arg.CType == "VipsImage**") && arg.IsOutput {
			op.HasImageOutput = true
		}
	}
}

func (v *Introspection) FixParameterTypes(op *vipsgen.Operation) {
	// Fix buffer length parameters
	for i, arg := range op.Arguments {
		// Fix gsize* -> size_t for buffer length parameters
		if arg.CType == "gsize*" && (arg.Name == "len" || strings.HasSuffix(arg.Name, "len")) {
			op.Arguments[i].CType = "size_t"
			op.Arguments[i].GoType = "int"
		}
	}
}

func (v *Introspection) FixVoidParameters(op *vipsgen.Operation) {
	// Special cases for functions with void* parameters
	if op.Name == "bandjoin_const" {
		for i, arg := range op.Arguments {
			if arg.Name == "c" && (arg.CType == "void*" || arg.GoType == "interface{}") {
				// This is an array of constants - should be []float64
				op.Arguments[i].CType = "double*"
				op.Arguments[i].GoType = "[]float64"
			}
		}
	} else if op.Name == "linear" {
		for i, arg := range op.Arguments {
			if (arg.Name == "a" || arg.Name == "b") && (arg.CType == "void*" || arg.GoType == "interface{}") {
				// These are arrays of constants - should be []float64
				op.Arguments[i].CType = "double*"
				op.Arguments[i].GoType = "[]float64"
			}
		}
	} else if op.Name == "composite" {
		// Already fixed in FixParameterTypes
	} else {
		// General case for void* parameters
		for i, arg := range op.Arguments {
			if arg.CType == "void*" {
				// For general void* parameters, determine the most likely type based on context
				if arg.Name == "buf" || strings.HasSuffix(arg.Name, "_buf") {
					// Buffer parameters are typically []byte
					op.Arguments[i].GoType = "[]byte"
				} else if arg.Name == "data" || strings.HasSuffix(arg.Name, "_data") {
					// Data parameters could be []byte or unsafe.Pointer depending on context
					op.Arguments[i].GoType = "[]byte"
				} else {
					// Default for other void* parameters
					op.Arguments[i].GoType = "unsafe.Pointer"
				}
			}
		}
	}
}
