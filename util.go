package vipsgen

import (
	"fmt"
	"strings"
	"text/template"
	"unicode"
)

// ImageMethodName converts vipsFooBar to FooBar for method names
func ImageMethodName(name string) string {
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
	// e.g., "rotate" -> "vipsRotate", "extract_area" -> "vipsExtractArea"
	parts := strings.Split(name, "_")

	// Convert each part to title case
	for i, part := range parts {
		parts[i] = strings.Title(part)
	}

	// Join with vips prefix
	return "vips" + strings.Join(parts, "")
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

	// For most categories, the URL format seems to be:
	return fmt.Sprintf("https://www.libvips.org/API/current/%s.html#vips-%s", docCategory, funcName)
}
