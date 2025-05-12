package vipsgen

import (
	"fmt"
	"strings"
	"text/template"
	"unicode"
)

// GetTemplateFuncMap Helper functions for templates
func GetTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
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

// formatImageFuncArgList formats arguments for NewImage* functions, changing *C.VipsImage to *Image
func formatImageFuncArgList(args []Argument) string {
	var params []string
	for _, arg := range args {
		params = append(params, fmt.Sprintf("%s %s", arg.GoName, convertParamType(arg)))
	}
	return strings.Join(params, ", ")
}

// formatImageFuncCallArgs formats arguments for function calls, converting *Image to image field
func formatImageFuncCallArgs(args []Argument) string {
	var callArgs []string
	for _, arg := range args {
		if arg.GoType == "*C.VipsImage" {
			callArgs = append(callArgs, fmt.Sprintf("%s.image", arg.GoName))
		} else if arg.GoType == "[]*C.VipsImage" {
			// Convert []*Image to []*C.VipsImage
			callArgs = append(callArgs, fmt.Sprintf("convertImagesToVipsImages(%s)", arg.GoName))
		} else {
			callArgs = append(callArgs, arg.GoName)
		}
	}
	return strings.Join(callArgs, ", ")
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

// FormatDefaultValue returns the appropriate "zero value" for a given Go type
func FormatDefaultValue(goType string) string {
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

// FormatErrorReturn formats the error return statement for a function
func FormatErrorReturn(hasImageOutput, hasBufferOutput bool, outputs []Argument) string {
	if hasImageOutput {
		return "return nil, handleImageError(out)"
	} else if hasBufferOutput {
		return "return nil, handleVipsError()"
	} else if len(outputs) > 0 {
		var returnValues []string
		for _, arg := range outputs {
			if arg.Name == "vector" || arg.Name == "out_array" {
				returnValues = append(returnValues, "nil")
			} else {
				returnValues = append(returnValues, FormatDefaultValue(arg.GoType))
			}
		}
		return "return " + strings.Join(returnValues, ", ") + ", handleVipsError()"
	} else {
		return "return handleVipsError()"
	}
}

// FormatGoArgList formats a list of function arguments for a Go function
// e.g., "in *C.VipsImage, c []float64, n int"
func FormatGoArgList(args []Argument) string {
	// Find buffer param if exists
	var inBufferParam *Argument
	var hasOutBufParam bool
	for i := range args {
		if args[i].GoType == "[]byte" && args[i].Name == "buf" {
			inBufferParam = &args[i]
			break
		}
	}
	var params []string
	for _, arg := range args {
		// Skip buffer length parameters
		if inBufferParam != nil && (arg.GoType == "int" || strings.Contains(arg.CType, "size_t")) && arg.Name == "len" {
			continue
		}
		if arg.CType == "void**" && arg.Name == "buf" {
			hasOutBufParam = true
			continue
		}
		if hasOutBufParam && arg.GoType == "int" && arg.Name == "len" {
			continue
		}
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
	} else if op.HasBufferOutput {
		return "[]byte, error"
	} else if len(op.Outputs) > 0 {
		var types []string
		for _, arg := range op.Outputs {
			// Special handling for vector/array return types
			if arg.Name == "vector" || arg.Name == "out_array" {
				types = append(types, "[]float64")
			} else {
				types = append(types, arg.GoType)
			}
		}
		types = append(types, "error")
		return strings.Join(types, ", ")
	} else {
		return "error"
	}
}

// FormatVarDeclarations formats variable declarations for output parameters
func FormatVarDeclarations(op Operation) string {
	var decls []string
	if op.HasBufferInput {
		decls = append(decls, fmt.Sprintf("src := %s", getBufferParamName(op.Arguments)))
		decls = append(decls, "// Reference src here so it's not garbage collected during image initialization.")
		decls = append(decls, "defer runtime.KeepAlive(src)")
	}
	if op.HasImageOutput {
		decls = append(decls, "var out *C.VipsImage")
	} else if op.HasBufferOutput {
		decls = append(decls, "var buf unsafe.Pointer")
		decls = append(decls, "var length C.size_t")
	} else {
		for _, arg := range op.Outputs {
			// Special handling for vector/array outputs
			if arg.Name == "vector" || arg.Name == "out_array" {
				decls = append(decls, "var out *C.double")
				decls = append(decls, "defer gFreePointer(unsafe.Pointer(out))")
			} else {
				decls = append(decls, fmt.Sprintf("var %s %s", arg.GoName, arg.GoType))

				// Add C type conversion if needed (for non-VipsImage outputs)
				if arg.GoType == "float64" {
					decls = append(decls, fmt.Sprintf("c%s := (*C.double)(unsafe.Pointer(&%s))",
						arg.GoName, arg.GoName))
				} else if arg.GoType == "int" {
					decls = append(decls, fmt.Sprintf("c%s := (*C.int)(unsafe.Pointer(&%s))",
						arg.GoName, arg.GoName))
				} else if arg.GoType == "bool" {
					decls = append(decls, fmt.Sprintf("c%s := (*C.int)(unsafe.Pointer(&%s))",
						arg.GoName, arg.GoName))
				}
			}
		}
	}
	if stringConv := FormatStringConversions(op.Arguments); stringConv != "" {
		decls = append(decls, stringConv)
	}
	if arrayConv := FormatArrayConversions(op.Arguments); arrayConv != "" {
		decls = append(decls, arrayConv)
	}
	return strings.Join(decls, "\n	")
}

// FormatStringConversions formats C string conversions for string parameters
func FormatStringConversions(args []Argument) string {
	var conversions []string
	for _, arg := range args {
		if !arg.IsOutput && arg.GoType == "string" {
			conversions = append(conversions, fmt.Sprintf("c%s := C.CString(%s)\n	defer freeCString(c%s)",
				arg.GoName, arg.GoName, arg.GoName))
		}
	}
	return strings.Join(conversions, "\n	")
}

// FormatArrayConversions formats array conversions for slice parameters
func FormatArrayConversions(args []Argument) string {
	var conversions []string
	for _, arg := range args {
		if !arg.IsOutput && strings.HasPrefix(arg.GoType, "[]") {
			if arg.GoType == "[]byte" && strings.Contains(arg.Name, "buf") {
				continue // Skip buffer parameters
			} else if arg.GoType == "[]string" {
				conversions = append(conversions, fmt.Sprintf(
					"c%s_ptrs := make([]*C.char, len(%s))\n"+
						"	for i, s := range %s {\n"+
						"		c%s_ptrs[i] = C.CString(s)\n"+
						"		defer freeCString(c%s_ptrs[i])\n"+
						"	}\n"+
						"	c%s := unsafe.Pointer(&c%s_ptrs[0])",
					arg.GoName, arg.GoName, arg.GoName,
					arg.GoName, arg.GoName, arg.GoName, arg.GoName))
			} else if arg.GoType == "[]BlendMode" {
				// Special handling for BlendMode arrays
				conversions = append(conversions, fmt.Sprintf(
					"c%s_arr := make([]C.int, len(%s))\n"+
						"	for i, v := range %s {\n"+
						"		c%s_arr[i] = C.int(v)\n"+
						"	}\n"+
						"	var c%s unsafe.Pointer\n"+
						"	if len(%s) > 0 {\n"+
						"		c%s = unsafe.Pointer(&c%s_arr[0])\n"+
						"	}",
					arg.GoName, arg.GoName, arg.GoName,
					arg.GoName, arg.GoName, arg.GoName, arg.GoName, arg.GoName))
			} else if arg.GoType == "[]float64" || arg.GoType == "[]float32" {
				// Special handling for float arrays - common in libvips const functions
				conversions = append(conversions, fmt.Sprintf(
					"var c%s unsafe.Pointer\n"+
						"	if len(%s) > 0 {\n"+
						"		c%s = unsafe.Pointer(&%s[0])\n"+
						"	}",
					arg.GoName, arg.GoName, arg.GoName, arg.GoName))
			} else if arg.GoType == "[]int" {
				// Standard int arrays
				conversions = append(conversions, fmt.Sprintf(
					"var c%s unsafe.Pointer\n"+
						"	if len(%s) > 0 {\n"+
						"		c%s = unsafe.Pointer(&%s[0])\n"+
						"	}",
					arg.GoName, arg.GoName, arg.GoName, arg.GoName))
			} else {
				// Generic array handling
				conversions = append(conversions, fmt.Sprintf(
					"var c%s unsafe.Pointer\n"+
						"	if len(%s) > 0 {\n"+
						"		c%s = unsafe.Pointer(&%s[0])\n"+
						"	}",
					arg.GoName, arg.GoName, arg.GoName, arg.GoName))
			}
		}
	}
	return strings.Join(conversions, "\n\n	")
}

// FormatFunctionCallArgs formats the arguments for the C function call
func FormatFunctionCallArgs(args []Argument) string {
	var callArgs []string
	for _, arg := range args {
		var argStr string
		if arg.IsOutput {
			if arg.Name == "out" {
				if arg.GoType == "*C.VipsImage" {
					argStr = "&out"
				} else {
					// Non-image output parameters should use c-prefixed variables
					argStr = "c" + arg.GoName
				}
			} else if arg.Name == "vector" || arg.Name == "out_array" {
				// Vector return value needs a double pointer
				argStr = "&out"
			} else if arg.CType == "size_t*" && arg.Name == "len" {
				// buffer output
				argStr = "&length"
			} else {
				// Non-out named output parameters
				if arg.GoType == "float64" || arg.GoType == "int" || arg.GoType == "bool" {
					argStr = "c" + arg.GoName
				} else {
					argStr = "&" + arg.GoName
				}
			}
		} else {
			// Handle input parameters
			if arg.GoType == "string" {
				argStr = "c" + arg.GoName
			} else if arg.GoType == "bool" {
				argStr = "C.int(boolToInt(" + arg.GoName + "))"
			} else if arg.GoType == "*C.VipsImage" {
				argStr = arg.GoName
			} else if arg.GoType == "[]byte" && strings.Contains(arg.Name, "buf") {
				// Special handling for byte buffers
				argStr = "unsafe.Pointer(&src[0])"
			} else if arg.Name == "len" && arg.CType == "size_t" {
				// input buffer
				argStr = "C.size_t(len(src))"
			} else if strings.HasPrefix(arg.GoType, "[]") {
				// For array parameters, handle each type specifically
				if arg.GoType == "[]*C.VipsImage" {
					// Use the appropriate casting for VipsImage arrays
					argStr = "(**C.VipsImage)(c" + arg.GoName + ")"
				} else if arg.GoType == "[]int" {
					// Use the appropriate casting for int arrays
					argStr = "(*C.int)(c" + arg.GoName + ")"
				} else if arg.GoType == "[]BlendMode" {
					// Special handling for BlendMode arrays
					argStr = "(*C.int)(c" + arg.GoName + ")"
				} else if arg.GoType == "[]float64" {
					// Use the appropriate casting for float arrays
					argStr = "(*C.double)(c" + arg.GoName + ")"
				} else if arg.GoType == "[]float32" {
					// Use the appropriate casting for float arrays
					argStr = "(*C.float)(c" + arg.GoName + ")"
				} else {
					// Generic unsafe pointer for other array types
					argStr = "c" + arg.GoName
				}
			} else if arg.IsEnum {
				argStr = "C." + arg.Type + "(" + arg.GoName + ")"
			} else if arg.CType == "void**" && arg.Name == "buf" {
				// buffer output
				argStr = "&buf"
			} else if arg.CType == "size_t*" && arg.Name == "len" {
				// buffer output
				argStr = "&length"
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
	} else if op.HasBufferOutput {
		return "return bufferToBytes(buf, length), nil"
	} else if len(op.Outputs) > 0 {
		var values []string

		for _, arg := range op.Outputs {
			// Special handling for vector outputs like getpoint
			if arg.Name == "vector" || arg.Name == "out_array" {
				// Get the n parameter which should be the second output
				nParam := "n"
				for _, outArg := range op.Outputs {
					if outArg.Name == "n" {
						nParam = outArg.GoName
						break
					}
				}
				// Convert the C array to a Go slice
				values = append(values, fmt.Sprintf("(*[1024]float64)(unsafe.Pointer(out))[:%s:%s]", nParam, nParam))
			} else {
				values = append(values, arg.GoName)
			}
		}

		return "return " + strings.Join(values, ", ") + ", nil"
	} else {
		return "return nil"
	}
}

// FormatImageMethodSignature formats the method signature for an image operation
func FormatImageMethodSignature(op Operation) string {
	// Get input parameters (excluding the image itself)
	inputParams := FilterInputParams(op.Arguments)

	// Format parameters
	var params []string
	for _, arg := range inputParams {
		params = append(params, fmt.Sprintf("%s %s", arg.GoName, arg.GoType))
	}

	// Format return type
	var returnType string
	if op.HasImageOutput {
		returnType = "error"
	} else if len(op.Outputs) > 0 {
		var types []string
		for _, arg := range op.Outputs {
			types = append(types, arg.GoType)
		}
		types = append(types, "error")
		returnType = strings.Join(types, ", ")
	} else {
		returnType = "error"
	}

	return fmt.Sprintf("func (r *Image) %s(%s) (%s)",
		ImageMethodName(op.GoName),
		strings.Join(params, ", "),
		returnType)
}

// FormatFunctionCall formats the call to the underlying vipsgen function
func FormatFunctionCall(op Operation) string {
	var args []string
	args = append(args, "r.image")

	for _, arg := range op.Arguments {
		if !arg.IsOutput && arg.Name != "in" && arg.Name != "out" {
			args = append(args, arg.GoName)
		}
	}

	return fmt.Sprintf("%s(%s)", op.GoName, strings.Join(args, ", "))
}

// FormatErrorReturnValues formats return values in case of error
func FormatErrorReturnValues(op Operation) string {
	if op.HasImageOutput {
		return "err"
	} else if len(op.Outputs) > 0 {
		var values []string
		for _, arg := range op.Outputs {
			if arg.Name == "vector" || arg.Name == "out_array" {
				values = append(values, "nil")
			} else if strings.HasPrefix(arg.GoType, "[]") {
				values = append(values, "nil")
			} else if arg.GoType == "bool" {
				values = append(values, "false")
			} else if arg.GoType == "string" {
				values = append(values, "\"\"")
			} else if arg.GoType == "*C.VipsImage" {
				values = append(values, "nil")
			} else {
				values = append(values, "0")
			}
		}
		values = append(values, "err")
		return strings.Join(values, ", ")
	} else {
		return "err"
	}
}

// FormatSuccessReturnValues formats return values in case of success
func FormatSuccessReturnValues(op Operation) string {
	if op.HasImageOutput {
		return "nil"
	} else if len(op.Outputs) > 0 {
		var values []string

		for _, arg := range op.Outputs {
			// Special handling for vector outputs like getpoint
			if arg.Name == "vector" || arg.Name == "out_array" {
				// Get the n parameter which should be the second output
				nParam := "n"
				for _, outArg := range op.Outputs {
					if outArg.Name == "n" {
						nParam = outArg.GoName
						break
					}
				}
				// Convert the C array to a Go slice
				values = append(values, fmt.Sprintf("(*[1024]float64)(unsafe.Pointer(out))[:%s:%s]", nParam, nParam))
			} else {
				values = append(values, arg.GoName)
			}
		}
		values = append(values, "nil")
		return strings.Join(values, ", ")
	} else {
		return "nil"
	}
}

// FormatImageMethodBody formats the body of an image method using improved argument detection
func FormatImageMethodBody(op Operation) string {
	methodArgs := DetectMethodArguments(op)

	// Format the arguments for the function call
	var callArgs []string
	callArgs = append(callArgs, "r.image") // The main input image

	for _, arg := range methodArgs {
		if arg.GoType == "*C.VipsImage" {
			callArgs = append(callArgs, fmt.Sprintf("%s.image", arg.GoName))
		} else if arg.GoType == "[]*C.VipsImage" {
			callArgs = append(callArgs, fmt.Sprintf("convertImagesToVipsImages(%s)", arg.GoName))
		} else {
			callArgs = append(callArgs, arg.GoName)
		}
	}

	// Check if the operation has any image outputs
	hasImageOutputs := false
	for _, arg := range op.Outputs {
		if arg.GoType == "*C.VipsImage" || arg.GoType == "[]*C.VipsImage" {
			hasImageOutputs = true
			break
		}
	}

	// Generate different function bodies based on operation type
	if op.HasImageOutput {
		return fmt.Sprintf(`out, err := %s(%s)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil`,
			op.GoName,
			strings.Join(callArgs, ", "))
	} else if op.HasBufferOutput {
		return fmt.Sprintf(`buf, err := %s(%s)
	if err != nil {
		return nil, err
	}
	return buf, nil`,
			op.GoName,
			strings.Join(callArgs, ", "))
	} else if len(op.Outputs) > 0 {
		// Check for specific operation patterns that need special handling
		if hasVectorReturn(op) {
			// For vector-returning operations like getpoint
			return fmt.Sprintf(`vector, n, err := %s(%s)
	if err != nil {
		return nil, 0, err
	}
	return vector, n, nil`,
				op.GoName,
				strings.Join(callArgs, ", "))
		} else if isSingleFloatReturn(op) {
			// For single float-returning operations like avg
			return fmt.Sprintf(`out, err := %s(%s)
	if err != nil {
		return 0, err
	}
	return out, nil`,
				op.GoName,
				strings.Join(callArgs, ", "))
		} else if hasImageOutputs {
			// For operations that return images

			// Get the names of the result variables
			var resultVars []string
			for _, arg := range op.Outputs {
				resultVars = append(resultVars, arg.GoName)
			}

			// Form the function call line
			callLine := fmt.Sprintf("%s, err := %s(%s)",
				strings.Join(resultVars, ", "),
				op.GoName,
				strings.Join(callArgs, ", "))

			// Form the error return line
			var errorValues []string
			for _, arg := range op.Outputs {
				if arg.GoType == "*C.VipsImage" || arg.GoType == "[]*C.VipsImage" {
					errorValues = append(errorValues, "nil")
				} else if strings.HasPrefix(arg.GoType, "[]") {
					errorValues = append(errorValues, "nil")
				} else if arg.GoType == "int" {
					errorValues = append(errorValues, "0")
				} else if arg.GoType == "float64" {
					errorValues = append(errorValues, "0")
				} else if arg.GoType == "bool" {
					errorValues = append(errorValues, "false")
				} else if arg.GoType == "string" {
					errorValues = append(errorValues, "\"\"")
				} else {
					errorValues = append(errorValues, "nil")
				}
			}
			errorLine := "return " + strings.Join(errorValues, ", ") + ", err"

			// Form the conversion code for each image output
			var conversionCode strings.Builder
			for i, arg := range op.Outputs {
				if arg.GoType == "*C.VipsImage" {
					// Convert *C.VipsImage to *Image
					conversionCode.WriteString(fmt.Sprintf(`
	var %sImage *Image
	if %s != nil {
		%sImage = newImageRef(%s, ImageTypeUnknown, nil)
	}`,
						arg.GoName, arg.GoName, arg.GoName, arg.GoName))
					resultVars[i] = arg.GoName + "Image"
				} else if arg.GoType == "[]*C.VipsImage" {
					// Convert []*C.VipsImage to []*Image
					conversionCode.WriteString(fmt.Sprintf(`
	%sImages := convertVipsImagesToImages(%s)`,
						arg.GoName, arg.GoName))
					resultVars[i] = arg.GoName + "Images"
				}
			}

			// Form the success return line
			successLine := "return " + strings.Join(resultVars, ", ") + ", nil"

			return callLine + `
	if err != nil {
		` + errorLine + `
	}` + conversionCode.String() + `
	` + successLine
		} else {
			// Regular operation with non-image outputs

			// Get the names of the result variables
			var resultVars []string
			for _, arg := range op.Outputs {
				resultVars = append(resultVars, arg.GoName)
			}

			// Form the function call line
			callLine := fmt.Sprintf("%s, err := %s(%s)",
				strings.Join(resultVars, ", "),
				op.GoName,
				strings.Join(callArgs, ", "))

			// Form the error return line
			var errorValues []string
			for _, arg := range op.Outputs {
				if strings.HasPrefix(arg.GoType, "[]") {
					errorValues = append(errorValues, "nil")
				} else if arg.GoType == "int" {
					errorValues = append(errorValues, "0")
				} else if arg.GoType == "float64" {
					errorValues = append(errorValues, "0")
				} else if arg.GoType == "bool" {
					errorValues = append(errorValues, "false")
				} else if arg.GoType == "string" {
					errorValues = append(errorValues, "\"\"")
				} else {
					errorValues = append(errorValues, "nil")
				}
			}
			errorLine := "return " + strings.Join(errorValues, ", ") + ", err"

			// Form the success return line
			successLine := "return " + strings.Join(resultVars, ", ") + ", nil"

			return callLine + `
	if err != nil {
		` + errorLine + `
	}
	` + successLine
		}
	} else {
		return fmt.Sprintf(`err := %s(%s)
	if err != nil {
		return err
	}
	return nil`,
			op.GoName,
			strings.Join(callArgs, ", "))
	}
}

// DetectMethodArguments analyzes an operation's arguments to determine which should be included in the method signature
func DetectMethodArguments(op Operation) []Argument {
	var methodArgs []Argument
	var firstImageFound bool
	var hasBufParam bool

	// Get all arguments except the first image input and output parameters
	for _, arg := range op.Arguments {
		// Skip output parameters
		if arg.IsOutput {
			continue
		}
		if arg.CType == "void**" && arg.Name == "buf" {
			hasBufParam = true
			continue
		} else if arg.Name == "len" && hasBufParam {
			continue
		}

		// Skip the first image input parameter (which will be the receiver)
		if (arg.Type == "VipsImage" || arg.CType == "VipsImage*") && !firstImageFound {
			firstImageFound = true
			continue
		}

		// Skip the output image
		if arg.Name == "out" {
			continue
		}

		// Include all other input parameters
		methodArgs = append(methodArgs, arg)
	}

	return methodArgs
}

// FormatImageMethodParams formats parameters for image methods using improved detection
func FormatImageMethodParams(op Operation) string {
	methodArgs := DetectMethodArguments(op)

	var params []string
	for _, arg := range methodArgs {
		// Convert parameter types for image methods
		var paramType string
		if arg.GoType == "*C.VipsImage" {
			paramType = "*Image"
		} else if arg.GoType == "[]*C.VipsImage" {
			paramType = "[]*Image"
		} else if arg.CType == "void*" {
			paramType = "[]byte"
		} else {
			paramType = arg.GoType
		}

		params = append(params, fmt.Sprintf("%s %s", arg.GoName, paramType))
	}
	return strings.Join(params, ", ")
}

// FilterInputParams correctly filters out output parameters and the input image parameter
func FilterInputParams(args []Argument) []Argument {
	var result []Argument

	// First, check for required, non-image input parameters (these must be preserved)
	for _, arg := range args {
		// Include required input arguments that aren't the image ("in") or output params
		if !arg.IsOutput && arg.Required && arg.Name != "in" && arg.Name != "out" {
			result = append(result, arg)
		}
	}

	// Then add optional input parameters
	for _, arg := range args {
		// Include optional input arguments
		if !arg.IsOutput && !arg.Required && arg.Name != "in" && arg.Name != "out" {
			result = append(result, arg)
		}
	}

	return result
}

// FormatImageMethodReturnTypes formats return types for image methods
func FormatImageMethodReturnTypes(op Operation) string {
	if op.HasImageOutput {
		return "error"
	} else if op.HasBufferOutput {
		return "[]byte, error"
	} else if len(op.Outputs) > 0 {
		var types []string
		for _, arg := range op.Outputs {
			// Special handling for vector return types
			if arg.Name == "vector" || arg.Name == "out_array" {
				types = append(types, "[]float64")
			} else if arg.GoType == "*C.VipsImage" {
				// Convert VipsImage output to *Image
				types = append(types, "*Image")
			} else if arg.GoType == "[]*C.VipsImage" {
				// Convert VipsImage array output to []*Image
				types = append(types, "[]*Image")
			} else {
				types = append(types, arg.GoType)
			}
		}
		types = append(types, "error")
		return strings.Join(types, ", ")
	} else {
		return "error"
	}
}

// FormatCreatorMethodParams formats the parameters for a creator method
func FormatCreatorMethodParams(op Operation) string {
	inputParams := op.RequiredInputs
	var hasBufParam bool

	var params []string
	for _, arg := range inputParams {
		var paramType string
		if arg.GoType == "*C.VipsImage" {
			paramType = "*Image"
		} else if arg.GoType == "[]*C.VipsImage" {
			paramType = "[]*Image"
		} else if arg.CType == "void*" && arg.Name == "buf" {
			paramType = "[]byte"
			hasBufParam = true
		} else if arg.Name == "len" && hasBufParam {
			continue
		} else {
			paramType = arg.GoType
		}
		params = append(params, fmt.Sprintf("%s %s", arg.GoName, paramType))
	}

	return strings.Join(params, ", ")
}

// FormatCreatorMethodBody formats the body of a creator method
func FormatCreatorMethodBody(op Operation) string {
	inputParams := op.RequiredInputs
	var hasBufParam bool

	// Format the arguments for the function call
	var callArgs []string
	for _, arg := range inputParams {
		if arg.GoType == "*C.VipsImage" {
			callArgs = append(callArgs, fmt.Sprintf("%s.image", arg.GoName))
		} else if arg.GoType == "[]*C.VipsImage" {
			callArgs = append(callArgs, fmt.Sprintf("convertImagesToVipsImages(%s)", arg.GoName))
		} else if arg.Name == "len" && arg.CType == "size_t" && hasBufParam {
			continue
		} else {
			if arg.Name == "buf" && arg.CType == "void*" {
				hasBufParam = true
			}
			callArgs = append(callArgs, arg.GoName)
		}
	}

	var imageRefBuf = "nil"
	if op.HasBufferInput {
		imageRefBuf = "buf"
	}
	return fmt.Sprintf(`StartupDefault()
	vipsImage, err := %s(%s)
	if err != nil {
		return nil, err
	}
	return newImageRef(vipsImage, %s, %s), nil`,
		op.GoName,
		strings.Join(callArgs, ", "),
		op.ImageTypeString,
		imageRefBuf)
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
