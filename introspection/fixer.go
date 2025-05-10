package introspection

import (
	"github.com/cshum/vipsgen"
	"strings"
)

func (v *Introspection) FixCType(cType string) string {
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
	// Special case for specific functions
	if op.Name == "composite" {
		// Fix the input image array parameter
		for i, arg := range op.Arguments {
			if arg.Name == "in" {
				op.Arguments[i].CType = "VipsImage**"
				op.Arguments[i].GoType = "[]*C.VipsImage"
			} else if arg.Name == "mode" {
				op.Arguments[i].CType = "int*"
				op.Arguments[i].GoType = "[]int"
			}
		}
	}

	// Fix buffer length parameters
	for i, arg := range op.Arguments {
		// Fix gsize* -> size_t for buffer length parameters
		if arg.CType == "gsize*" && (arg.Name == "len" || strings.HasSuffix(arg.Name, "len")) {
			op.Arguments[i].CType = "size_t"
			op.Arguments[i].GoType = "int"
		}
	}
}

func (v *Introspection) FixArrayParameters(op *vipsgen.Operation) {
	// Identify array parameters by name and context
	for i, arg := range op.Arguments {
		// Array parameters often have corresponding 'n' or 'count' parameters
		isArray := false
		hasCountParam := false

		// Check for a corresponding count parameter
		for _, other := range op.Arguments {
			// Common patterns for count parameters
			if other.Name == "n" ||
				other.Name == "count" ||
				other.Name == (arg.Name+"_n") ||
				other.Name == (arg.Name+"_count") {
				hasCountParam = true
				break
			}
		}

		// If there's a count parameter and this is a pointer type, it's likely an array
		if hasCountParam && strings.HasSuffix(arg.CType, "*") {
			isArray = true
		}

		// Special case for common array parameter names
		if (arg.Name == "array" || arg.Name == "items" || arg.Name == "elements") &&
			strings.HasSuffix(arg.CType, "*") {
			isArray = true
		}

		// If this parameter is identified as an array
		if isArray {
			// Extract the base type (remove one level of pointer)
			baseType := strings.TrimSuffix(arg.CType, "*")

			// Adjust the Go type to be a slice
			if baseType == "VipsImage" {
				op.Arguments[i].GoType = "[]*C.VipsImage"
			} else if baseType == "int" || baseType == "gint" {
				op.Arguments[i].GoType = "[]int"
			} else if baseType == "double" || baseType == "gdouble" {
				op.Arguments[i].GoType = "[]float64"
			} else if baseType == "float" || baseType == "gfloat" {
				op.Arguments[i].GoType = "[]float32"
			} else if baseType == "char" || baseType == "gchar" {
				op.Arguments[i].GoType = "[]string"
			}

			// Keep the C type as a pointer, which is correct for arrays in C
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
