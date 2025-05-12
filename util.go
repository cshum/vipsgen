package vipsgen

import (
	"fmt"
	"strings"
	"text/template"
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
		"imageMethodName":              ImageMethodName,
		"generateDocUrl":               GenerateDocUrl,
		"formatImageMethodArgs":        FormatImageMethodArgs,
		"split":                        strings.Split,
		"filterInputParams":            FilterInputParams,
		"isPointerType":                isPointerType,
		"formatDefaultValue":           FormatDefaultValue,
		"formatErrorReturn":            FormatErrorReturn,
		"formatGoArgList":              FormatGoArgList,
		"formatReturnTypes":            FormatReturnTypes,
		"formatVarDeclarations":        FormatVarDeclarations,
		"formatStringConversions":      FormatStringConversions,
		"formatArrayConversions":       FormatArrayConversions,
		"formatFunctionCallArgs":       FormatFunctionCallArgs,
		"formatFunctionCall":           FormatFunctionCall,
		"formatReturnValues":           FormatReturnValues,
		"formatSuccessReturnValues":    FormatSuccessReturnValues,
		"formatErrorReturnValues":      FormatErrorReturnValues,
		"formatImageMethodSignature":   FormatImageMethodSignature,
		"formatImageMethodBody":        FormatImageMethodBody,
		"formatImageFuncArgList":       formatImageFuncArgList,
		"formatImageFuncCallArgs":      formatImageFuncCallArgs,
		"formatImageMethodParams":      FormatImageMethodParams,
		"formatImageMethodReturnTypes": FormatImageMethodReturnTypes,
		"formatCreatorMethodParams":    FormatCreatorMethodParams,
		"formatCreatorMethodBody":      FormatCreatorMethodBody,
		"hasBufferParam":               hasBufferParam,
		"hasLengthParam":               hasLengthParam,
		"getBufferParamName":           getBufferParamName,

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
		"hasArrayImageInput": HasArrayImageInput,
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

func isPointerType(typeName string) bool {
	return strings.Contains(typeName, "*")
}
