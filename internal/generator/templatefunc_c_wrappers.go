package generator

import (
	"fmt"
	"strings"

	"github.com/cshum/vipsgen/internal/introspection"
)

// generateCFunctionSignature generates just the function signature for vips operations
func generateCFunctionSignature(op introspection.Operation, includeParamNames bool) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("int vipsgen_%s(", op.Name))
	if len(op.Arguments) > 0 {
		for i, arg := range op.Arguments {
			if i > 0 {
				result.WriteString(", ")
			}
			if includeParamNames {
				result.WriteString(fmt.Sprintf("%s %s", arg.CType, arg.Name))
			} else {
				result.WriteString(arg.CType)
			}
		}
	}
	result.WriteString(")")
	return result.String()
}

func hasCArrayLengthParam(goType string) bool {
	return goType == "[]float64" || goType == "[]float32" ||
		goType == "[]int" || goType == "[]BlendMode" ||
		goType == "[]*C.VipsImage" || goType == "[]*Image"
}

func generateCWithOptionsSignature(op introspection.Operation) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("int vipsgen_%s_with_options(", op.Name))

	if len(op.Arguments) > 0 {
		for i, arg := range op.Arguments {
			if i > 0 {
				result.WriteString(", ")
			}
			result.WriteString(fmt.Sprintf("%s %s", arg.CType, arg.Name))
		}
	}

	for i, opt := range op.OptionalInputs {
		if i > 0 || len(op.Arguments) > 0 {
			result.WriteString(", ")
		}
		result.WriteString(fmt.Sprintf("%s %s", opt.CType, opt.Name))

		if hasCArrayLengthParam(opt.GoType) {
			result.WriteString(fmt.Sprintf(", int %s_n", opt.Name))
		}
	}

	supportedOptionalOutputs := getSupportedOptionalOutputs(op)
	for i, opt := range supportedOptionalOutputs {
		if i > 0 || len(op.Arguments) > 0 || len(op.OptionalInputs) > 0 {
			result.WriteString(", ")
		}
		result.WriteString(fmt.Sprintf("%s %s", opt.CType, opt.Name))
	}

	result.WriteString(")")
	return result.String()
}

func writeCArrayDeclaration(result *strings.Builder, opName string, arg introspection.Argument, lengthExpr string) {
	arrayType := getArrayType(arg.GoType)
	switch arrayType {
	case "double":
		result.WriteString(fmt.Sprintf("    VipsArrayDouble *%s_array = NULL;\n", arg.Name))
		result.WriteString(fmt.Sprintf("    if (%s != NULL && %s > 0) { %s_array = vips_array_double_new(%s, %s); }\n", arg.Name, lengthExpr, arg.Name, arg.Name, lengthExpr))
	case "int":
		result.WriteString(fmt.Sprintf("    VipsArrayInt *%s_array = NULL;\n", arg.Name))
		if opName == "composite" && arg.Name == "mode" && lengthExpr == "n" {
			result.WriteString(fmt.Sprintf("    if (%s != NULL && %s > 1) { %s_array = vips_array_int_new(%s, %s-1); }\n", arg.Name, lengthExpr, arg.Name, arg.Name, lengthExpr))
		} else {
			result.WriteString(fmt.Sprintf("    if (%s != NULL && %s > 0) { %s_array = vips_array_int_new(%s, %s); }\n", arg.Name, lengthExpr, arg.Name, arg.Name, lengthExpr))
		}
	case "image":
		result.WriteString(fmt.Sprintf("    VipsArrayImage *%s_array = NULL;\n", arg.Name))
		result.WriteString(fmt.Sprintf("    if (%s != NULL && %s > 0) { %s_array = vips_array_image_new(%s, %s); }\n", arg.Name, lengthExpr, arg.Name, arg.Name, lengthExpr))
	}
}

func writeCArrayCleanups(result *strings.Builder, indent string, args []introspection.Argument) {
	for _, arg := range args {
		if strings.HasPrefix(arg.GoType, "[]") {
			arrayType := getArrayType(arg.GoType)
			if arrayType != "unknown" {
				result.WriteString(fmt.Sprintf("%sif (%s_array != NULL) { vips_area_unref(VIPS_AREA(%s_array)); }\n", indent, arg.Name, arg.Name))
			}
		}
	}
}

// generateCFunctionDeclaration generates header declarations for vips operations
func generateCFunctionDeclaration(op introspection.Operation) string {
	var result strings.Builder
	if len(op.Arguments) == 0 {
		result.WriteString(fmt.Sprintf("int vipsgen_%s();", op.Name))
	} else {
		result.WriteString(generateCFunctionSignature(op, true))
		result.WriteString(";")
	}

	supportedOptionalOutputs := getSupportedOptionalOutputs(op)
	if len(op.OptionalInputs) > 0 || len(supportedOptionalOutputs) > 0 {
		result.WriteString("\n")
		result.WriteString(generateCWithOptionsSignature(op))
		result.WriteString(";")
	}
	return result.String()
}

// generateCFunctionImplementation generates C implementations for vips operations
func generateCFunctionImplementation(op introspection.Operation) string {
	var result strings.Builder

	if len(op.Arguments) == 0 {
		result.WriteString(fmt.Sprintf("int vipsgen_%s() {\n", op.Name))
		result.WriteString(fmt.Sprintf("    return vips_%s(NULL);\n}", op.Name))
	} else {
		result.WriteString(generateCFunctionSignature(op, true))
		result.WriteString(" {\n")

		result.WriteString(fmt.Sprintf("    return vips_%s(", op.Name))
		for i, arg := range op.Arguments {
			if i > 0 {
				result.WriteString(", ")
			}
			if arg.IsSource {
				result.WriteString("(VipsSource*) " + arg.Name)
			} else if arg.IsTarget {
				result.WriteString("(VipsTarget*) " + arg.Name)
			} else {
				result.WriteString(arg.Name)
			}
		}
		result.WriteString(", NULL);\n}")
	}

	supportedOptionalOutputs := getSupportedOptionalOutputs(op)
	if len(op.OptionalInputs) > 0 || len(supportedOptionalOutputs) > 0 {
		result.WriteString("\n\n")
		result.WriteString(generateCWithOptionsSignature(op))
		result.WriteString(" {\n")
		result.WriteString(fmt.Sprintf("    VipsOperation *operation = vips_operation_new(\"%s\");\n", op.Name))
		result.WriteString("    if (!operation) return 1;\n")

		isBufferLoadOperation := strings.Contains(op.Name, "load_buffer") || op.Name == "thumbnail_buffer"
		isBufferSaveOperation := strings.Contains(op.Name, "save_buffer")

		if isBufferLoadOperation {
			result.WriteString("    VipsBlob *blob = vips_blob_new(NULL, buf, len);\n")
			result.WriteString("    if (!blob) { g_object_unref(operation); return 1; }\n")
		}

		for _, arg := range op.RequiredInputs {
			if strings.HasPrefix(arg.GoType, "[]") {
				writeCArrayDeclaration(&result, op.Name, arg, "n")
			}
		}
		for _, opt := range op.OptionalInputs {
			if strings.HasPrefix(opt.GoType, "[]") {
				writeCArrayDeclaration(&result, op.Name, opt, fmt.Sprintf("%s_n", opt.Name))
			}
		}

		var allParamsList []string

		for _, arg := range op.Arguments {
			if arg.IsOutput {
				continue
			}
			if arg.IsInputN {
				continue
			}

			if arg.IsArray {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vips_object_set(VIPS_OBJECT(operation), \"%s\", %s_array, NULL)", arg.Name, arg.Name))
			} else if arg.IsSource {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vips_object_set(VIPS_OBJECT(operation), \"%s\", (VipsSource*)%s, NULL)", arg.Name, arg.Name))
			} else if arg.IsTarget {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vips_object_set(VIPS_OBJECT(operation), \"%s\", (VipsTarget*)%s, NULL)", arg.Name, arg.Name))
			} else if (arg.Name == "buf" || arg.Name == "buffer") && isBufferLoadOperation {
				allParamsList = append(allParamsList,
					"vips_object_set(VIPS_OBJECT(operation), \"buffer\", blob, NULL)")
			} else if arg.Name == "len" && isBufferLoadOperation {
				continue
			} else if arg.GoType == "string" {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vips_object_set(VIPS_OBJECT(operation), \"%s\", %s, NULL)", arg.Name, arg.Name))
			} else if arg.GoType == "*C.VipsImage" {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vips_object_set(VIPS_OBJECT(operation), \"%s\", %s, NULL)", arg.Name, arg.Name))
			} else {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vips_object_set(VIPS_OBJECT(operation), \"%s\", %s, NULL)", arg.Name, arg.Name))
			}
		}

		for _, opt := range op.OptionalInputs {
			if strings.HasPrefix(opt.GoType, "[]") {
				arrayType := getArrayType(opt.GoType)
				if arrayType == "double" {
					allParamsList = append(allParamsList,
						fmt.Sprintf("vipsgen_set_array_double(operation, \"%s\", %s_array)", opt.Name, opt.Name))
				} else if arrayType == "int" {
					allParamsList = append(allParamsList,
						fmt.Sprintf("vipsgen_set_array_int(operation, \"%s\", %s_array)", opt.Name, opt.Name))
				} else if arrayType == "image" {
					allParamsList = append(allParamsList,
						fmt.Sprintf("vipsgen_set_array_image(operation, \"%s\", %s_array)", opt.Name, opt.Name))
				}
			} else if opt.GoType == "bool" {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_bool(operation, \"%s\", %s)", opt.Name, opt.Name))
			} else if opt.GoType == "string" {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_string(operation, \"%s\", %s)", opt.Name, opt.Name))
			} else if opt.IsEnum {
				if opt.Name == "keep" && opt.EnumType == "Keep" {
					allParamsList = append(allParamsList,
						fmt.Sprintf("vipsgen_set_keep(operation, %s)", opt.Name))
				} else {
					allParamsList = append(allParamsList,
						fmt.Sprintf("vipsgen_set_int(operation, \"%s\", %s)", opt.Name, opt.Name))
				}
			} else if opt.GoType == "*C.VipsImage" {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_image(operation, \"%s\", %s)", opt.Name, opt.Name))
			} else if opt.GoType == "*Interpolate" || opt.GoType == "*C.VipsInterpolate" {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_interpolate(operation, \"%s\", %s)", opt.Name, opt.Name))
			} else if opt.IsSource {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_source(operation, \"%s\", %s)", opt.Name, opt.Name))
			} else if opt.IsTarget {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_target(operation, \"%s\", %s)", opt.Name, opt.Name))
			} else if opt.GoType == "int" {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_int(operation, \"%s\", %s)", opt.Name, opt.Name))
			} else if opt.GoType == "float64" {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_double(operation, \"%s\", %s)", opt.Name, opt.Name))
			} else if strings.Contains(opt.CType, "guint64") {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_guint64(operation, \"%s\", %s)", opt.Name, opt.Name))
			} else if strings.Contains(opt.CType, "unsigned int") || strings.Contains(opt.CType, "guint") {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_int(operation, \"%s\", %s)", opt.Name, opt.Name))
			} else if strings.Contains(opt.CType, "*") || strings.Contains(opt.GoType, "*") {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vips_object_set(VIPS_OBJECT(operation), \"%s\", %s, NULL)", opt.Name, opt.Name))
			} else {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_int(operation, \"%s\", %s)", opt.Name, opt.Name))
			}
		}

		if len(allParamsList) > 0 {
			result.WriteString("    if (\n        ")
			result.WriteString(strings.Join(allParamsList, " ||\n        "))
			result.WriteString("\n    ) {\n")

			if isBufferLoadOperation {
				result.WriteString("        vips_area_unref((VipsArea *)blob);\n")
			}

			result.WriteString("        g_object_unref(operation);\n")
			writeCArrayCleanups(&result, "        ", op.RequiredInputs)
			writeCArrayCleanups(&result, "        ", op.OptionalInputs)

			result.WriteString("        return 1;\n    }\n")
		}

		if isBufferLoadOperation {
			result.WriteString("    vips_area_unref((VipsArea *)blob);\n")
		}

		if isBufferSaveOperation {
			result.WriteString("    int result = vipsgen_operation_save_buffer(operation, buf, len);\n")
		} else {
			var outputParams []string
			for _, arg := range op.Arguments {
				if arg.IsOutput {
					if arg.Name == "out" {
						outputParams = append(outputParams, "\"out\", out")
					} else if arg.CType == "double*" {
						outputParams = append(outputParams, fmt.Sprintf("\"%s\", %s", arg.Name, arg.Name))
					} else if arg.CType == "int*" {
						outputParams = append(outputParams, fmt.Sprintf("\"%s\", %s", arg.Name, arg.Name))
					} else {
						outputParams = append(outputParams, fmt.Sprintf("\"%s\", %s", arg.Name, arg.Name))
					}
				}
			}

			for _, opt := range supportedOptionalOutputs {
				outputParams = append(outputParams, fmt.Sprintf("\"%s\", %s", opt.Name, opt.Name))
			}

			outputParams = append(outputParams, "NULL")
			result.WriteString(fmt.Sprintf("    int result = vipsgen_operation_execute(operation, %s);\n", strings.Join(outputParams, ", ")))
		}

		writeCArrayCleanups(&result, "    ", op.RequiredInputs)
		writeCArrayCleanups(&result, "    ", op.OptionalInputs)

		result.WriteString("    return result;\n}")
	}

	return result.String()
}
