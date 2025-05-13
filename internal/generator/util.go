package generator

import (
	"strings"
)

// Helper function to check if an operation returns a single float value
func isSingleFloatReturn(op Operation) bool {
	return len(op.Outputs) == 1 && op.Outputs[0].GoType == "float64"
}

func getBufferParamName(args []Argument) string {
	for _, arg := range args {
		if arg.GoType == "[]byte" && strings.Contains(arg.Name, "buf") {
			return arg.GoName
		}
	}
	return "buf" // Default fallback
}

// SnakeToCamel converts a snake_case string to CamelCase
func SnakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		parts[i] = strings.Title(parts[i])
	}
	return strings.Join(parts, "")
}

// Helper function to check if an operation returns a vector
func hasVectorReturn(op Operation) bool {
	hasVector := false
	hasN := false
	for _, arg := range op.Outputs {
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
