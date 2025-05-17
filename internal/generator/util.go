package generator

import (
	"github.com/cshum/vipsgen/internal/introspection"
	"strings"
)

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
