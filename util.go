package vipsgen

import (
	"fmt"
	"strings"
)

// Helper function to check if an operation returns a single float value
func isSingleFloatReturn(op Operation) bool {
	return len(op.Outputs) == 1 && op.Outputs[0].GoType == "float64"
}

func hasLengthParam(args []Argument) bool {
	for _, arg := range args {
		if (arg.GoType == "int" || strings.Contains(arg.CType, "size_t")) &&
			(strings.Contains(arg.Name, "len") || strings.Contains(arg.Name, "length")) {
			return true
		}
	}
	return false
}

func getBufferParamName(args []Argument) string {
	for _, arg := range args {
		if arg.GoType == "[]byte" && strings.Contains(arg.Name, "buf") {
			return arg.GoName
		}
	}
	return "buf" // Default fallback
}

// convertParam converts *C.VipsImage to *Image parameters
func convertParamType(arg Argument) string {
	if arg.GoType == "*C.VipsImage" {
		return "*Image"
	}
	if arg.GoType == "[]*C.VipsImage" {
		return "[]*Image"
	}
	return arg.GoType
}

// ImageMethodName converts vipsFooBar to FooBar for method names
func ImageMethodName(name string) string {
	if strings.HasPrefix(name, "vipsgen") {
		return name[7:]
	}
	if strings.HasPrefix(name, "vips") {
		return name[4:]
	}
	return name
}

// HasArrayImageInput checks if any of the arguments are array image inputs
func HasArrayImageInput(args []Argument) bool {
	for _, arg := range args {
		// Look for array of VipsImage pointers
		if strings.HasPrefix(arg.GoType, "[]*C.VipsImage") {
			return true
		}
		// Also check the C type, which might indicate an array
		if strings.Contains(arg.CType, "VipsImage**") && !arg.IsOutput {
			return true
		}
	}
	return false
}

var categoryToDocMap = map[string]string{
	"foreign": "VipsForeignSave",
}

func GenerateDocUrl(funcName string, sourceCategory string) string {
	// Look up the documentation category
	docCategory, exists := categoryToDocMap[sourceCategory]
	if !exists {
		// Default to the source category if no mapping exists
		docCategory = "libvips-" + sourceCategory
	}

	funcName = strings.ReplaceAll(funcName, "_", "-")

	// For most categories, the URL format seems to be:
	return fmt.Sprintf("https://www.libvips.org/API/current/%s.html#vips-%s", docCategory, funcName)
}

func isPointerType(typeName string) bool {
	return strings.Contains(typeName, "*")
}
