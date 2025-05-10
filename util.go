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
		"imageMethodName":         ImageMethodName,
		"generateDocUrl":          GenerateDocUrl,
		"formatImageMethodArgs":   FormatImageMethodArgs,
		"split":                   strings.Split,
		"filterInputParams":       FilterInputParams,
		"isPointerType":           isPointerType,
		"formatDefaultValue":      formatDefaultValue,
		"formatErrorReturn":       formatErrorReturn,
		"formatGoArgList":         FormatGoArgList,
		"formatReturnTypes":       FormatReturnTypes,
		"formatVarDeclarations":   FormatVarDeclarations,
		"formatStringConversions": FormatStringConversions,
		"formatArrayConversions":  FormatArrayConversions,
		"formatFunctionCallArgs":  FormatFunctionCallArgs,
		"formatReturnValues":      FormatReturnValues,

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

// FormatGoArgList formats a list of function arguments for a Go function
// e.g., "in *C.VipsImage, c []float64, n int"
func FormatGoArgList(args []Argument) string {
	var params []string
	for _, arg := range args {
		if !arg.IsOutput {
			params = append(params, fmt.Sprintf("%s %s", arg.GoName, arg.GoType))
		}
	}
	return strings.Join(params, ", ")
}

// FormatReturnTypes formats the return types for a Go function
// e.g., "*C.VipsImage, error" or "int, float64, error"
func FormatReturnTypes(op Operation) string {
	if op.HasImageOutput {
		return "*C.VipsImage, error"
	} else if len(op.Outputs) > 0 {
		var types []string
		for _, arg := range op.Outputs {
			types = append(types, arg.GoType)
		}
		types = append(types, "error")
		return strings.Join(types, ", ")
	} else {
		return "error"
	}
}

// FormatVarDeclarations formats variable declarations for output parameters
func FormatVarDeclarations(op Operation) string {
	if op.HasImageOutput {
		return "var out *C.VipsImage"
	} else if len(op.Outputs) > 0 {
		var decls []string
		for _, arg := range op.Outputs {
			decls = append(decls, fmt.Sprintf("var %s %s", arg.GoName, arg.GoType))
		}
		return strings.Join(decls, "\n    ")
	}
	return ""
}

// FormatStringConversions formats C string conversions for string parameters
func FormatStringConversions(args []Argument) string {
	var conversions []string
	for _, arg := range args {
		if !arg.IsOutput && arg.GoType == "string" {
			conversions = append(conversions, fmt.Sprintf("c%s := C.CString(%s)\n    defer freeCString(c%s)",
				arg.GoName, arg.GoName, arg.GoName))
		}
	}
	return strings.Join(conversions, "\n    ")
}

// FormatArrayConversions formats array conversions for slice parameters
func FormatArrayConversions(args []Argument) string {
	var conversions []string
	for _, arg := range args {
		if !arg.IsOutput && strings.HasPrefix(arg.GoType, "[]") {
			if arg.GoType == "[]string" {
				conversions = append(conversions, fmt.Sprintf(
					"// Convert []string to C array for %s\n"+
						"    c%s_ptrs := make([]*C.char, len(%s))\n"+
						"    for i, s := range %s {\n"+
						"        c%s_ptrs[i] = C.CString(s)\n"+
						"        defer freeCString(c%s_ptrs[i])\n"+
						"    }\n"+
						"    c%s := unsafe.Pointer(&c%s_ptrs[0])",
					arg.GoName, arg.GoName, arg.GoName, arg.GoName,
					arg.GoName, arg.GoName, arg.GoName, arg.GoName))
			} else if arg.GoType == "[]float64" || arg.GoType == "[]float32" {
				// Special handling for float arrays - common in libvips const functions
				conversions = append(conversions, fmt.Sprintf(
					"// Convert slice to C array for %s\n"+
						"    var c%s unsafe.Pointer\n"+
						"    if len(%s) > 0 {\n"+
						"        c%s = unsafe.Pointer(&%s[0])\n"+
						"    }",
					arg.GoName, arg.GoName, arg.GoName, arg.GoName, arg.GoName))
			} else {
				conversions = append(conversions, fmt.Sprintf(
					"// Convert slice to C array for %s\n"+
						"    var c%s unsafe.Pointer\n"+
						"    if len(%s) > 0 {\n"+
						"        c%s = unsafe.Pointer(&%s[0])\n"+
						"    }",
					arg.GoName, arg.GoName, arg.GoName, arg.GoName, arg.GoName))
			}
		}
	}
	return strings.Join(conversions, "\n\n    ")
}

// Update FormatFunctionCallArgs to handle special float array casting for const operations
func FormatFunctionCallArgs(args []Argument) string {
	var callArgs []string
	for _, arg := range args {
		var argStr string
		if arg.IsOutput {
			if arg.Name == "out" {
				argStr = "&out"
			} else {
				argStr = "&" + arg.GoName
			}
		} else {
			if arg.GoType == "string" {
				argStr = "c" + arg.GoName
			} else if arg.GoType == "bool" {
				argStr = "C.int(boolToInt(" + arg.GoName + "))"
			} else if arg.GoType == "*C.VipsImage" {
				argStr = arg.GoName
			} else if strings.HasPrefix(arg.GoType, "[]") {
				// For array parameters, use the c-prefixed variable with proper casting for const operations
				if arg.Name == "c" && (strings.HasSuffix(arg.CType, "*") || strings.HasSuffix(arg.CType, "const double*")) {
					argStr = "(*C.double)(c" + arg.GoName + ")"
				} else {
					argStr = "c" + arg.GoName // No casting needed, it's already an unsafe.Pointer
				}
			} else if arg.IsEnum {
				argStr = "C." + arg.Type + "(" + arg.GoName + ")"
			} else {
				// For regular scalar types, use normal C casting
				argStr = "C." + arg.CType + "(" + arg.GoName + ")"
			}
		}
		callArgs = append(callArgs, argStr)
	}
	return strings.Join(callArgs, ", ")
}

// FormatReturnValues formats the return values for the Go function
func FormatReturnValues(op Operation) string {
	if op.HasImageOutput {
		return "return out, nil"
	} else if len(op.Outputs) > 0 {
		var values []string
		for _, arg := range op.Outputs {
			values = append(values, arg.GoName)
		}
		return "return " + strings.Join(values, ", ") + ", nil"
	} else {
		return "return nil"
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
