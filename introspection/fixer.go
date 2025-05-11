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

// FixConstFunctions add a special case fix function for _const functions in introspection/fixer.go:
func (v *Introspection) FixConstFunctions(op *vipsgen.Operation) {
	// Special handling for _const functions that operate on arrays
	if strings.HasSuffix(op.Name, "_const") {
		for i, arg := range op.Arguments {
			// Parameters named 'a', 'b', 'c' in _const functions are typically arrays
			if (arg.Name == "a" || arg.Name == "b" || arg.Name == "c") &&
				strings.HasPrefix(arg.CType, "double") && !arg.IsOutput {
				// Fix the type to be a double array
				op.Arguments[i].GoType = "[]float64"
			} else if arg.Name == "c" && (arg.CType == "double" || arg.CType == "const double*") && !arg.IsOutput {
				// Fix specific known cases like boolean_const, math2_const, remainder_const
				op.Arguments[i].GoType = "[]float64"
			}
		}
	} else if op.Name == "linear" || op.Name == "boolean_const" || op.Name == "math2_const" || op.Name == "remainder_const" {
		// Special case for specific functions
		for i, arg := range op.Arguments {
			if (arg.Name == "a" || arg.Name == "b" || arg.Name == "c") && !arg.IsOutput {
				op.Arguments[i].GoType = "[]float64"
			}
		}
	}
}

// FixOperationTypes examines operations and adjusts their types based on patterns
func (v *Introspection) FixOperationTypes(op *vipsgen.Operation) {
	// Pattern detection: Vector return operations
	// If function has output param named "vector" paired with output param "n", it's returning an array
	hasVectorParam := false
	hasNParam := false

	for _, arg := range op.Outputs {
		if arg.Name == "vector" {
			hasVectorParam = true
		}
		if arg.Name == "n" {
			hasNParam = true
		}
	}

	// If we have both vector and n params, this is a vector return function
	if hasVectorParam && hasNParam {
		for i, arg := range op.Outputs {
			if arg.Name == "vector" {
				// Update the type to be a slice
				op.Outputs[i].GoType = "[]float64"

				// Also update in Arguments if present
				for j, mainArg := range op.Arguments {
					if mainArg.Name == "vector" {
						op.Arguments[j].GoType = "[]float64"
					}
				}
			}
		}
	}

	// Find ink parameter and check if paired with n parameter
	inkParam := -1
	nParam := -1

	for i, arg := range op.Arguments {
		if arg.Name == "ink" {
			inkParam = i
		}
		if arg.Name == "n" {
			nParam = i
		}
	}
	// If both found, modify the ink parameter to be an array
	if inkParam >= 0 && nParam >= 0 && op.Arguments[inkParam].GoType == "float64" {
		op.Arguments[inkParam].GoType = "[]float64"

		// Also update in RequiredInputs if present
		for i, arg := range op.RequiredInputs {
			if arg.Name == "ink" {
				op.RequiredInputs[i].GoType = "[]float64"
			}
		}
	}
}
