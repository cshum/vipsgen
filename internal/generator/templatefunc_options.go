package generator

import (
	"fmt"
	"strings"

	"github.com/cshum/vipsgen/internal/introspection"
)

// generateOptionalInputsStruct generates a parameter struct for an operation
func generateOptionalInputsStruct(op introspection.Operation) string {
	supportedOptionalOutputs := getSupportedOptionalOutputs(op)
	if len(op.OptionalInputs) == 0 && len(supportedOptionalOutputs) == 0 {
		return ""
	}
	var result strings.Builder

	structName := op.GoName + "Options"

	result.WriteString(fmt.Sprintf("// %s optional arguments for vips_%s\n", structName, op.Name))
	result.WriteString(fmt.Sprintf("type %s struct {\n", structName))

	for _, opt := range op.OptionalInputs {
		fieldName := strings.Title(opt.GoName)
		var fieldType string
		if opt.GoType == "*C.VipsImage" {
			fieldType = "*Image"
		} else if opt.GoType == "[]*C.VipsImage" {
			fieldType = "[]*Image"
		} else if opt.CType == "void*" {
			fieldType = "[]byte"
		} else {
			fieldType = opt.GoType
		}
		if opt.IsEnum && opt.EnumType != "" {
			fieldType = opt.EnumType
		}
		if opt.Description != "" {
			result.WriteString(fmt.Sprintf("\t// %s %s\n", fieldName, opt.Description))
		}
		result.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, fieldType))
	}

	if len(supportedOptionalOutputs) > 0 {
		for _, opt := range supportedOptionalOutputs {
			fieldName := strings.Title(opt.GoName)
			fieldType := opt.GoType
			if opt.Description != "" {
				result.WriteString(fmt.Sprintf("\t// %s Output, %s\n", fieldName, opt.Description))
			} else {
				result.WriteString(fmt.Sprintf("\t// %s Output\n", fieldName))
			}
			result.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, fieldType))
		}
	}

	result.WriteString("}\n\n")

	result.WriteString(fmt.Sprintf("// Default%s creates default value for vips_%s optional arguments\n",
		structName, op.Name))
	result.WriteString(fmt.Sprintf("func Default%s() *%s {\n", structName, structName))
	result.WriteString(fmt.Sprintf("\treturn &%s{\n", structName))
	for _, opt := range op.OptionalInputs {
		fieldName := strings.Title(opt.GoName)

		if opt.DefaultValue != nil {
			switch v := opt.DefaultValue.(type) {
			case bool:
				if v {
					result.WriteString(fmt.Sprintf("\t\t%s: %t,\n", fieldName, v))
				}
			case int:
				if v != 0 {
					if opt.IsEnum && opt.EnumType != "" {
						result.WriteString(fmt.Sprintf("\t\t%s: %s(%d),\n", fieldName, opt.EnumType, v))
					} else {
						result.WriteString(fmt.Sprintf("\t\t%s: %d,\n", fieldName, v))
					}
				}
			case float64:
				if v != 0 {
					result.WriteString(fmt.Sprintf("\t\t%s: %g,\n", fieldName, v))
				}
			case string:
				if v != "" {
					result.WriteString(fmt.Sprintf("\t\t%s: %q,\n", fieldName, v))
				}
			}
		}
	}
	result.WriteString("\t}\n}\n")

	return result.String()
}

// generateUtilFunctionCallArgs formats function call arguments without the 'this' pointer
func generateUtilFunctionCallArgs(op introspection.Operation) string {
	var args []string
	for _, arg := range op.RequiredInputs {
		if arg.IsInputN {
			continue
		}
		if arg.GoType == "*C.VipsImage" {
			args = append(args, fmt.Sprintf("%s.image", arg.GoName))
		} else if arg.GoType == "[]*C.VipsImage" {
			args = append(args, fmt.Sprintf("convertImagesToVipsImages(%s)", arg.GoName))
		} else {
			args = append(args, arg.GoName)
		}
	}
	return strings.Join(args, ", ")
}

// generateUtilityFunctionReturnTypes formats return types for utility functions (non-image operations)
func generateUtilityFunctionReturnTypes(op introspection.Operation) string {
	if op.HasBufferOutput {
		return "[]byte, error"
	} else if len(op.RequiredOutputs) > 0 {
		var types []string
		for _, arg := range op.RequiredOutputs {
			if arg.IsOutputN {
				continue
			}
			if arg.Name == "vector" || arg.Name == "out_array" {
				types = append(types, "[]float64")
			} else {
				types = append(types, arg.GoType)
			}
		}
		types = append(types, "error")
		return strings.Join(types, ", ")
	}
	return "error"
}