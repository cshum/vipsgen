package vipsgen

import (
	"fmt"
	"strings"
	"text/template"
	"unicode"
)

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

// FormatImageMethodArgs formats arguments for image methods, skipping the first Image argument if it exists
func FormatImageMethodArgs(args []Argument) string {
	var params []string
	for i, arg := range args {
		if i == 0 && arg.Type == "VipsImage" {
			// Skip the first Image argument only if it's a VipsImage
			continue
		}
		params = append(params, fmt.Sprintf("%s %s", arg.GoName, arg.GoType))
	}
	return strings.Join(params, ", ")
}

// FormatGoFunctionName formats an operation name to a Go function name
func FormatGoFunctionName(name string) string {
	// Convert operation names to match existing Go function style
	// e.g., "rotate" -> "vipsgenRotate", "extract_area" -> "vipsgenExtractArea"
	parts := strings.Split(name, "_")

	// Convert each part to title case
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[0:1]) + part[1:]
		}
	}

	// Join with vipsgen prefix instead of vips
	return "vipsgen" + strings.Join(parts, "")
}

// FormatIdentifier formats a name to an identifier
func FormatIdentifier(name string) string {
	// Handle Go keywords
	switch name {
	case "type", "func", "map", "range", "select", "case", "default":
		return name + "_"
	}

	// Handle special cases
	name = strings.Replace(name, "-", "_", -1)

	return name
}

// FormatGoIdentifier formats a name to a go identifier
func FormatGoIdentifier(name string) string {
	s := SnakeToCamel(FormatIdentifier(name))

	// first letter lower case
	if len(s) == 0 {
		return s
	}

	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// GetGoEnumName converts a C enum type name to a Go type name
func GetGoEnumName(cName string) string {
	// Strip "Vips" prefix if present
	if strings.HasPrefix(cName, "Vips") {
		cName = cName[4:]
	}

	// Also strip "Foreign" prefix if present in both the original name
	// and after removing "Vips" prefix
	if strings.HasPrefix(cName, "Foreign") {
		cName = cName[7:]
	}

	return cName
}

// SnakeToCamel converts a snake_case string to CamelCase
func SnakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		parts[i] = strings.Title(parts[i])
	}
	return strings.Join(parts, "")
}

// FormatEnumValueName converts a C enum name to a Go name
func FormatEnumValueName(typeName, valueName string) string {
	// Convert to CamelCase
	camelValue := SnakeToCamel(strings.ToLower(valueName))

	// Check if the value already contains "Vips" + typeName or "VipsForeign" + typeName
	lowerCamelValue := strings.ToLower(camelValue)
	lowerTypeName := strings.ToLower(typeName)

	if strings.HasPrefix(lowerCamelValue, "vips"+lowerTypeName) ||
		strings.HasPrefix(lowerCamelValue, "vipsforeign"+lowerTypeName) {
		return GetGoEnumName(camelValue)
	}

	// Otherwise, prepend the type name
	return typeName + GetGoEnumName(camelValue)
}

// FilterInputParams filters the arguments to include only those that should be input parameters for an Image method
func FilterInputParams(args []Argument) []Argument {
	var result []Argument
	for _, arg := range args {
		// Include only non-output arguments that aren't the input image or predefined output params
		if !arg.IsOutput && arg.Name != "in" && arg.Name != "out" && arg.Name != "columns" && arg.Name != "rows" {
			result = append(result, arg)
		}
	}
	return result
}

// GetTemplateFuncMap Helper functions for templates
func GetTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"formatArgs": func(args []Argument) string {
			var params []string
			for _, arg := range args {
				params = append(params, fmt.Sprintf("%s %s", arg.GoName, arg.GoType))
			}
			return strings.Join(params, ", ")
		},
		"hasVipsImageInput": func(args []Argument) bool {
			for _, arg := range args {
				if arg.Type == "VipsImage" {
					return true
				}
			}
			return false
		},
		"cArgList": func(args []Argument) string {
			var params []string
			for _, arg := range args {
				params = append(params, fmt.Sprintf("%s %s", arg.CType, arg.Name))
			}
			return strings.Join(params, ", ")
		},
		"callArgList": func(args []Argument) string {
			var params []string
			for _, arg := range args {
				params = append(params, arg.Name)
			}
			return strings.Join(params, ", ")
		},
		"imageMethodName":       ImageMethodName,
		"generateDocUrl":        GenerateDocUrl,
		"formatImageMethodArgs": FormatImageMethodArgs,
		"split":                 strings.Split,
		"filterInputParams":     FilterInputParams,
		"isPointerType":         isPointerType,
		"formatDefaultValue":    formatDefaultValue,
		"formatErrorReturn":     formatErrorReturn,

		"hasPrefix":  strings.HasPrefix,
		"hasSuffix":  strings.HasSuffix,
		"trimPrefix": strings.TrimPrefix,
		"trimSuffix": strings.TrimSuffix,

		"isArrayType": func(goType string) bool {
			return strings.HasPrefix(goType, "[]")
		},

		"arrayElementType": func(goType string) string {
			if strings.HasPrefix(goType, "[]") {
				return strings.TrimPrefix(goType, "[]")
			}
			return goType
		},

		"arrayCType": func(cType string) string {
			// Remove one level of pointer for array element type
			if strings.HasSuffix(cType, "*") {
				return strings.TrimSuffix(cType, "*")
			}
			return cType
		},
	}
}

func isPointerType(typeName string) bool {
	return strings.Contains(typeName, "*")
}

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

// formatErrorReturn formats the error return statement for a function
func formatErrorReturn(hasImageOutput bool, outputs []Argument) string {
	if hasImageOutput {
		return "return nil, handleImageError(out)"
	} else if len(outputs) > 0 {
		var returnValues []string
		for _, arg := range outputs {
			returnValues = append(returnValues, formatDefaultValue(arg.GoType))
		}
		return "return " + strings.Join(returnValues, ", ") + ", handleVipsError()"
	} else {
		return "return handleVipsError()"
	}
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
