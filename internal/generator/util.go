package generator

import (
	"github.com/cshum/vipsgen/internal/introspection"
	"strings"
)

// formatDefaultValue returns the appropriate "zero value" for a given Go type
func formatDefaultValue(goType string) string {
	// Handle slice types
	if strings.HasPrefix(goType, "[]") {
		return "nil"
	}

	// Handle specific types
	switch goType {
	case "bool":
		return "false"
	case "string":
		return "\"\""
	case "error":
		return "nil"
	}

	// Handle pointer types
	if isPointerType(goType) {
		return "nil"
	}

	// Default for numeric types
	return "0"
}

// Helper function to check if an operation returns a single float value
func isSingleFloatReturn(op introspection.Operation) bool {
	return len(op.RequiredOutputs) == 1 && op.RequiredOutputs[0].GoType == "float64"
}

func getBufferParamName(args []introspection.Argument) string {
	for _, arg := range args {
		if arg.GoType == "[]byte" && strings.Contains(arg.Name, "buf") {
			return arg.GoName
		}
	}
	return "buf" // Default fallback
}

// Helper function to check if an operation returns a vector
func hasVectorReturn(op introspection.Operation) bool {
	hasVector := false
	hasN := false
	for _, arg := range op.RequiredOutputs {
		if arg.Name == "vector" && arg.GoType == "[]float64" {
			hasVector = true
		}
		if arg.Name == "n" {
			hasN = true
		}
	}
	return hasVector && hasN
}

func isPointerType(typeName string) bool {
	return strings.Contains(typeName, "*")
}

// Helper function to detect array type for proper VipsArray creation
func getArrayType(goType string) string {
	if strings.HasPrefix(goType, "[]float64") || strings.HasPrefix(goType, "[]float32") {
		return "double"
	} else if strings.HasPrefix(goType, "[]int") || strings.HasPrefix(goType, "[]BlendMode") {
		return "int"
	} else if strings.HasPrefix(goType, "[]*C.VipsImage") {
		return "image"
	} else {
		return "unknown"
	}
}

func getBufferParameter(args []introspection.Argument) *introspection.Argument {
	for _, arg := range args {
		if arg.IsBuffer && arg.IsInput && arg.GoType == "[]byte" {
			return &arg
		}
	}
	return nil
}
