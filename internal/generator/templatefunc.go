package generator

import (
	"fmt"
	"github.com/cshum/vipsgen/internal/introspection"
	"strings"
	"text/template"
)

// GetTemplateFuncMap Helper functions for templates
func GetTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"generateGoFunctionBody":             generateGoFunctionBody,
		"generateFunctionCallArgs":           generateFunctionCallArgs,
		"generateFunctionCall":               generateFunctionCall,
		"generateImageMethodBody":            generateImageMethodBody,
		"generateImageArgumentsComment":      generateImageArgumentsComment,
		"generateImageMethodParams":          generateImageMethodParams,
		"generateImageMethodReturnTypes":     generateImageMethodReturnTypes,
		"generateMethodParams":               generateMethodParams,
		"generateCreatorMethodBody":          generateCreatorMethodBody,
		"generateCFunctionDeclaration":       generateCFunctionDeclaration,
		"generateCFunctionImplementation":    generateCFunctionImplementation,
		"generateOptionalInputsStruct":       generateOptionalInputsStruct,
		"generateUtilFunctionCallArgs":       generateUtilFunctionCallArgs,
		"generateUtilityFunctionReturnTypes": generateUtilityFunctionReturnTypes,
	}
}

// generateGoFunctionBody generates the shared body for Go wrapper functions
func generateGoFunctionBody(op introspection.Operation, withOptions bool) string {
	var result strings.Builder
	// Function name and comment
	if withOptions {
		result.WriteString(fmt.Sprintf("// vipsgen%sWithOptions %s with optional arguments\n",
			op.GoName, op.Description))
		result.WriteString(fmt.Sprintf("func vipsgen%sWithOptions(", op.GoName))
	} else {
		result.WriteString(fmt.Sprintf("// vipsgen%s %s\n", op.GoName, op.Description))
		result.WriteString(fmt.Sprintf("func vipsgen%s(", op.GoName))
	}

	// Function arguments
	result.WriteString(generateGoArgList(op, withOptions))
	result.WriteString(") (")
	result.WriteString(generateReturnTypes(op))
	result.WriteString(") {\n\t")

	// Variable declarations
	result.WriteString(generateVarDeclarations(op, withOptions))
	result.WriteString("\n\t")

	// Function call
	if withOptions {
		result.WriteString(fmt.Sprintf("if err := C.vipsgen_%s_with_options(", op.Name))
	} else {
		result.WriteString(fmt.Sprintf("if err := C.vipsgen_%s(", op.Name))
	}
	result.WriteString(generateFunctionCallArgs(op, withOptions))
	result.WriteString("); err != 0 {\n\t\t")

	// Error handling
	result.WriteString(generateErrorReturn(op.HasOneImageOutput, op.HasBufferOutput, op.RequiredOutputs))
	result.WriteString("\n\t}\n\t")

	// Return values
	result.WriteString(generateReturnValues(op))
	result.WriteString("\n}")

	return result.String()
}

// generateErrorReturn formats the error return statement for a function
func generateErrorReturn(HasOneImageOutput, hasBufferOutput bool, outputs []introspection.Argument) string {
	if HasOneImageOutput {
		return "return nil, handleImageError(out)"
	} else if hasBufferOutput {
		return "return nil, handleVipsError()"
	} else if len(outputs) > 0 {
		var returnValues []string
		for _, arg := range outputs {
			// Skip returning the length parameter if it's marked as IsOutputN
			if arg.IsOutputN {
				continue
			}
			if arg.Name == "vector" || arg.Name == "out_array" {
				returnValues = append(returnValues, "nil")
			} else {
				returnValues = append(returnValues, formatDefaultValue(arg.GoType))
			}
		}
		return "return " + strings.Join(returnValues, ", ") + ", handleVipsError()"
	} else {
		return "return handleVipsError()"
	}
}

// Helper function to determine error return based on function type
func generateErrorReturnForUtilityCall(op introspection.Operation) string {
	// Determine the appropriate error return based on output type
	if op.HasOneImageOutput {
		return "return nil, err"
	} else if op.HasBufferOutput {
		return "return nil, err"
	} else if len(op.RequiredOutputs) > 0 {
		var values []string
		for _, arg := range op.RequiredOutputs {
			if arg.Name == "vector" || arg.Name == "out_array" {
				values = append(values, "nil")
			} else {
				values = append(values, formatDefaultValue(arg.GoType))
			}
		}
		return "return " + strings.Join(values, ", ") + ", err"
	} else {
		return "return err"
	}
}

// generateGoArgList formats a list of function arguments for a Go function
// e.g., "in *C.VipsImage, c []float64, n int"
func generateGoArgList(op introspection.Operation, withOptions bool) string {
	args := op.Arguments
	if withOptions {
		args = append(args, op.OptionalInputs...)
	}
	// Find buffer param if exists
	var inBufferParam *introspection.Argument
	var hasOutBufParam bool
	for i := range args {
		if args[i].GoType == "[]byte" && args[i].Name == "buf" {
			inBufferParam = &args[i]
			break
		}
	}
	var params []string
	for _, arg := range args {
		// Skip n parameters that can be automatically calculated
		if arg.IsInputN {
			continue
		}
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

// generateReturnTypes formats the return types for a Go function
// e.g., "*C.VipsImage, error" or "int, float64, error"
func generateReturnTypes(op introspection.Operation) string {
	if op.HasOneImageOutput {
		return "*C.VipsImage, error"
	} else if op.HasBufferOutput {
		return "[]byte, error"
	} else if len(op.RequiredOutputs) > 0 {
		var types []string
		for _, arg := range op.RequiredOutputs {
			// Skip returning the length parameter if it's marked as IsOutputN
			if arg.IsOutputN {
				continue
			}
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

// generateVarDeclarations formats variable declarations for output parameters
func generateVarDeclarations(op introspection.Operation, withOptions bool) string {
	var decls []string
	if op.HasBufferInput {
		decls = append(decls, fmt.Sprintf("src := %s", getBufferParamName(op.Arguments)))
		decls = append(decls, "// Reference src here so it's not garbage collected during image initialization.")
		decls = append(decls, "defer runtime.KeepAlive(src)")
	}

	if op.HasOneImageOutput {
		decls = append(decls, "var out *C.VipsImage")
	} else if op.HasBufferOutput {
		// Check if we have a VipsBlob output parameter
		hasVipsBlob := false
		for _, arg := range op.RequiredOutputs {
			if arg.CType == "VipsBlob**" && arg.IsOutput {
				hasVipsBlob = true
				decls = append(decls, fmt.Sprintf("var %s *C.VipsBlob", arg.GoName))
				break
			}
		}

		if !hasVipsBlob {
			// Regular buffer output
			decls = append(decls, "var buf unsafe.Pointer")
			decls = append(decls, "var length C.size_t")
		}
	} else {
		for _, arg := range op.RequiredOutputs {
			// Special handling for VipsBlob
			if arg.CType == "VipsBlob**" && arg.IsOutput {
				decls = append(decls, fmt.Sprintf("var %s *C.VipsBlob", arg.GoName))
				continue
			}
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

	if stringConv := formatStringConversions(op.Arguments); stringConv != "" {
		decls = append(decls, stringConv)
	}

	// Process array conversions using updated utility functions
	args := op.Arguments
	if withOptions {
		args = append(args, op.OptionalInputs...)
	}

	for _, arg := range args {
		if !arg.IsOutput && strings.HasPrefix(arg.GoType, "[]") {
			if arg.GoType == "[]byte" && strings.Contains(arg.Name, "buf") {
				continue // Skip buffer parameters
			}

			// Use utility functions with proper error handling
			errorReturn := generateErrorReturnForUtilityCall(op)

			if arg.GoType == "[]float64" || arg.GoType == "[]float32" {
				// For required array parameters in non-options function, we don't need the length
				lengthVar := fmt.Sprintf("c%sLength", arg.GoName)
				if arg.IsRequired {
					lengthVar = "_" // Use underscore for unused length
				}

				decls = append(decls, fmt.Sprintf(
					"c%s, %s, err := convertToDoubleArray(%s)\n"+
						"	if err != nil {\n"+
						"		%s\n"+
						"	}\n"+
						"	if c%s != nil {\n"+
						"		defer freeDoubleArray(c%s)\n"+
						"	}",
					arg.GoName, lengthVar, arg.GoName, errorReturn, arg.GoName, arg.GoName))
			} else if arg.GoType == "[]int" {
				// For required array parameters in non-options function, we don't need the length
				lengthVar := fmt.Sprintf("c%sLength", arg.GoName)
				if arg.IsRequired {
					lengthVar = "_" // Use underscore for unused length
				}

				decls = append(decls, fmt.Sprintf(
					"c%s, %s, err := convertToIntArray(%s)\n"+
						"	if err != nil {\n"+
						"		%s\n"+
						"	}\n"+
						"	if c%s != nil {\n"+
						"		defer freeIntArray(c%s)\n"+
						"	}",
					arg.GoName, lengthVar, arg.GoName, errorReturn, arg.GoName, arg.GoName))
			} else if arg.GoType == "[]BlendMode" {
				// For required array parameters in non-options function, we don't need the length
				lengthVar := fmt.Sprintf("c%sLength", arg.GoName)
				if arg.IsRequired {
					lengthVar = "_" // Use underscore for unused length
				}

				decls = append(decls, fmt.Sprintf(
					"c%s, %s, err := convertToBlendModeArray(%s)\n"+
						"	if err != nil {\n"+
						"		%s\n"+
						"	}\n"+
						"	if c%s != nil {\n"+
						"		defer freeIntArray(c%s)\n"+
						"	}",
					arg.GoName, lengthVar, arg.GoName, errorReturn, arg.GoName, arg.GoName))
			} else if arg.GoType == "[]*Image" || arg.GoType == "[]*C.VipsImage" {
				// For required array parameters in non-options function, we don't need the length
				lengthVar := fmt.Sprintf("c%sLength", arg.GoName)
				if arg.IsRequired {
					lengthVar = "_" // Use underscore for unused length
				}

				// Use utility function for image arrays
				decls = append(decls, fmt.Sprintf(
					"c%s, %s, err := convertToImageArray(%s)\n"+
						"	if err != nil {\n"+
						"		%s\n"+
						"	}\n"+
						"	if c%s != nil {\n"+
						"		defer freeImageArray(c%s)\n"+
						"	}",
					arg.GoName, lengthVar, arg.GoName, errorReturn, arg.GoName, arg.GoName))
			} else {
				// Legacy handling for other array types
				decls = append(decls, fmt.Sprintf(
					"var c%s unsafe.Pointer\n"+
						"	if len(%s) > 0 {\n"+
						"		c%s = unsafe.Pointer(&%s[0])\n"+
						"	}",
					arg.GoName, arg.GoName, arg.GoName, arg.GoName))
			}
		}
	}

	if withOptions {
		if stringConv := formatStringConversions(op.OptionalInputs); stringConv != "" {
			decls = append(decls, stringConv)
		}
	}

	return strings.Join(decls, "\n	")
}

// formatStringConversions formats C string conversions for string parameters
func formatStringConversions(args []introspection.Argument) string {
	var conversions []string
	for _, arg := range args {
		if !arg.IsOutput && arg.GoType == "string" {
			conversions = append(conversions, fmt.Sprintf("c%s := C.CString(%s)\n	defer freeCString(c%s)",
				arg.GoName, arg.GoName, arg.GoName))
		}
	}
	return strings.Join(conversions, "\n	")
}

// generateFunctionCallArgs formats the arguments for the C function call
func generateFunctionCallArgs(op introspection.Operation, withOptions bool) string {
	args := op.Arguments
	if withOptions {
		args = append(args, op.OptionalInputs...)
	}
	var callArgs []string

	// Track which arrays we've processed to handle their lengths
	processedArrays := make(map[string]bool)

	// Map to store array lengths
	arrayLengths := make(map[string]string)

	for _, arg := range args {
		var argStr string

		if arg.IsOutput {
			// Handle output parameters (unchanged)
			if arg.Name == "out" || op.HasOneImageOutput {
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
			callArgs = append(callArgs, argStr)
		} else {
			// Handle IsInputN parameters specially - calculate from the referenced array
			if arg.IsInputN && arg.NInputFrom != "" {
				argStr = fmt.Sprintf("C.int(len(%s))", arg.NInputFrom)
				callArgs = append(callArgs, argStr)
				continue
			}
			if arg.IsSource || arg.IsTarget {
				callArgs = append(callArgs, arg.GoName)
			} else if arg.GoType == "string" {
				argStr = "c" + arg.GoName
				callArgs = append(callArgs, argStr)
			} else if arg.GoType == "bool" {
				argStr = "C.int(boolToInt(" + arg.GoName + "))"
				callArgs = append(callArgs, argStr)
			} else if arg.GoType == "*C.VipsImage" {
				argStr = arg.GoName
				callArgs = append(callArgs, argStr)
			} else if arg.GoType == "[]byte" && strings.Contains(arg.Name, "buf") {
				// Special handling for byte buffers
				argStr = "unsafe.Pointer(&src[0])"
				callArgs = append(callArgs, argStr)
			} else if arg.GoType == "*Interpolate" {
				// Handle Interpolate parameters - convert from Go to C type
				argStr = "vipsInterpolateToC(" + arg.GoName + ")"
				callArgs = append(callArgs, argStr)
			} else if arg.Name == "len" && arg.CType == "size_t" {
				// input buffer
				argStr = "C.size_t(len(src))"
				callArgs = append(callArgs, argStr)
			} else if strings.HasPrefix(arg.GoType, "[]") {
				// For array parameters, add both the array pointer and its length

				// Store the array name and length for possible reference by IsInputN parameters
				arrayLengths[arg.Name] = fmt.Sprintf("len(%s)", arg.GoName)

				// Check if we should add array length parameter based on type
				needsLengthParam := false
				if !arg.IsRequired && (arg.GoType == "[]float64" || arg.GoType == "[]float32" ||
					arg.GoType == "[]int" || arg.GoType == "[]BlendMode" ||
					arg.GoType == "[]*C.VipsImage" || arg.GoType == "[]*Image") {
					needsLengthParam = true
				}

				// Mark this array as processed so we don't duplicate
				processedArrays[arg.Name] = true

				// Determine the array pointer variable name - different for with_options vs basic functions
				arrayVarName := "c" + arg.GoName

				// Add the array parameter - NO ADDITIONAL TYPE CASTING for utility function results
				if withOptions {
					// For functions with options, we use the utility function result directly
					argStr = arrayVarName
				} else {
					// For basic functions without options, we may need type casting
					if arg.GoType == "[]*C.VipsImage" {
						argStr = "(**C.VipsImage)(" + arrayVarName + ")"
					} else if arg.GoType == "[]int" || arg.GoType == "[]BlendMode" {
						argStr = arrayVarName // No additional casting needed
					} else if arg.GoType == "[]float64" || arg.GoType == "[]float32" {
						argStr = arrayVarName // No additional casting needed
					} else {
						// Generic unsafe pointer for other array types
						argStr = arrayVarName
					}
				}
				callArgs = append(callArgs, argStr)

				// Add the length parameter if needed
				if needsLengthParam {
					lengthArg := "c" + arg.GoName + "Length"
					callArgs = append(callArgs, lengthArg)
				}
			} else if arg.IsEnum {
				argStr = "C." + arg.Type + "(" + arg.GoName + ")"
				callArgs = append(callArgs, argStr)
			} else if arg.CType == "void**" && arg.Name == "buf" {
				// buffer output
				argStr = "&buf"
				callArgs = append(callArgs, argStr)
			} else if arg.CType == "size_t*" && arg.Name == "len" {
				// buffer output
				argStr = "&length"
				callArgs = append(callArgs, argStr)
			} else {
				// For regular scalar types, use normal C casting
				argStr = "C." + arg.CType + "(" + arg.GoName + ")"
				callArgs = append(callArgs, argStr)
			}
		}
	}

	return strings.Join(callArgs, ", ")
}

// generateReturnValues formats the return values for the Go function
func generateReturnValues(op introspection.Operation) string {
	// Special handling for VipsBlob
	for _, arg := range op.RequiredOutputs {
		if arg.CType == "VipsBlob**" && arg.IsOutput {
			return fmt.Sprintf("return vipsBlobToBytes(%s), nil", arg.GoName)
		}
	}
	if op.HasOneImageOutput {
		return "return out, nil"
	} else if op.HasBufferOutput {
		return "return bufferToBytes(buf, length), nil"
	} else if len(op.RequiredOutputs) > 0 {
		var values []string

		for _, arg := range op.RequiredOutputs {
			// Skip returning the length parameter if it's marked as IsOutputN
			if arg.IsOutputN {
				continue
			}
			// Special handling for vector outputs like getpoint
			if arg.Name == "vector" || arg.Name == "out_array" {
				// Get the n parameter which should be the second output
				nParam := "n"
				for _, outArg := range op.RequiredOutputs {
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

// generateFunctionCall formats the call to the underlying vipsgen function
func generateFunctionCall(op introspection.Operation) string {
	var args []string
	args = append(args, "r.image")

	for _, arg := range op.Arguments {
		if !arg.IsOutput && arg.Name != "in" && arg.Name != "out" {
			args = append(args, arg.GoName)
		}
	}

	return fmt.Sprintf("%s(%s)", op.GoName, strings.Join(args, ", "))
}

// generateImageMethodBody formats the body of an image method using improved argument detection
func generateImageMethodBody(op introspection.Operation) string {
	methodArgs := detectMethodArguments(op)
	goFuncName := "vipsgen" + op.GoName
	goFuncNameWithOptions := "vipsgen" + op.GoName + "WithOptions"

	// Format the arguments for the function call
	var callArgs []string
	callArgs = append(callArgs, "r.image") // The main input image

	for _, arg := range methodArgs {
		if arg.GoType == "*C.VipsImage" {
			callArgs = append(callArgs, fmt.Sprintf("%s.image", arg.GoName))
		} else if arg.IsTarget {
			callArgs = append(callArgs, fmt.Sprintf("%s.target", arg.GoName))
		} else if arg.GoType == "[]*C.VipsImage" {
			callArgs = append(callArgs, fmt.Sprintf("convertImagesToVipsImages(%s)", arg.GoName))
		} else {
			callArgs = append(callArgs, arg.GoName)
		}
	}

	// Generate different function bodies based on operation type
	if op.HasOneImageOutput {
		var body string

		// Handle options if present
		if len(op.OptionalInputs) > 0 {
			// Create options arguments
			var optionsCallArgs = make([]string, len(callArgs))
			copy(optionsCallArgs, callArgs)

			for _, opt := range op.OptionalInputs {
				var optStr string
				if opt.GoType == "*C.VipsImage" {
					optStr = fmt.Sprintf("options.%s.image", strings.Title(opt.GoName))
				} else if opt.GoType == "[]*C.VipsImage" {
					optStr = fmt.Sprintf("convertImagesToVipsImages(options.%s)", strings.Title(opt.GoName))
				} else {
					optStr = fmt.Sprintf("options.%s", strings.Title(opt.GoName))
				}
				optionsCallArgs = append(optionsCallArgs, optStr)
			}

			body = fmt.Sprintf(`if options != nil {
		out, err := %s(%s)
		if err != nil {
			return err
		}
		r.setImage(out)
		return nil
	}
	`, goFuncNameWithOptions, strings.Join(optionsCallArgs, ", "))
		}

		// Add regular function call
		body += fmt.Sprintf(`out, err := %s(%s)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil`,
			goFuncName,
			strings.Join(callArgs, ", "))
		return body
	} else if op.HasBufferOutput {
		var body string

		// Handle options if present
		if len(op.OptionalInputs) > 0 {
			// Create options arguments
			var optionsCallArgs = make([]string, len(callArgs))
			copy(optionsCallArgs, callArgs)

			for _, opt := range op.OptionalInputs {
				var optStr string
				if opt.GoType == "*C.VipsImage" {
					optStr = fmt.Sprintf("options.%s.image", strings.Title(opt.GoName))
				} else if opt.GoType == "[]*C.VipsImage" {
					optStr = fmt.Sprintf("convertImagesToVipsImages(options.%s)", strings.Title(opt.GoName))
				} else {
					optStr = fmt.Sprintf("options.%s", strings.Title(opt.GoName))
				}
				optionsCallArgs = append(optionsCallArgs, optStr)
			}

			// For buffer output with options
			body = fmt.Sprintf(`if options != nil {
		buf, err := %s(%s)
		if err != nil {
			return nil, err
		}
		return buf, nil
	}
	`, goFuncNameWithOptions, strings.Join(optionsCallArgs, ", "))
		}

		body += fmt.Sprintf(`buf, err := %s(%s)
	if err != nil {
		return nil, err
	}
	return buf, nil`,
			goFuncName,
			strings.Join(callArgs, ", "))
		return body
	} else if len(op.RequiredOutputs) > 0 {
		// Check for specific operation patterns that need special handling
		if hasVectorReturn(op) {
			// For vector-returning operations like getpoint
			var body string

			// Handle options if present
			if len(op.OptionalInputs) > 0 {
				// Create options arguments
				var optionsCallArgs = make([]string, len(callArgs))
				copy(optionsCallArgs, callArgs)

				for _, opt := range op.OptionalInputs {
					var optStr string
					if opt.GoType == "*C.VipsImage" {
						optStr = fmt.Sprintf("options.%s.image", strings.Title(opt.GoName))
					} else if opt.GoType == "[]*C.VipsImage" {
						optStr = fmt.Sprintf("convertImagesToVipsImages(options.%s)", strings.Title(opt.GoName))
					} else {
						optStr = fmt.Sprintf("options.%s", strings.Title(opt.GoName))
					}
					optionsCallArgs = append(optionsCallArgs, optStr)
				}

				// With options for vector return
				body = fmt.Sprintf(`if options != nil {
		vector, n, err := %s(%s)
		if err != nil {
			return nil, 0, err
		}
		return vector, n, nil
	}
	`, goFuncNameWithOptions, strings.Join(optionsCallArgs, ", "))
			}

			body += fmt.Sprintf(`vector, n, err := %s(%s)
	if err != nil {
		return nil, 0, err
	}
	return vector, n, nil`,
				goFuncName,
				strings.Join(callArgs, ", "))
			return body
		} else if isSingleFloatReturn(op) {
			// For single float-returning operations like avg
			var body string

			// Handle options if present
			if len(op.OptionalInputs) > 0 {
				// Create options arguments
				var optionsCallArgs = make([]string, len(callArgs))
				copy(optionsCallArgs, callArgs)

				for _, opt := range op.OptionalInputs {
					var optStr string
					if opt.GoType == "*C.VipsImage" {
						optStr = fmt.Sprintf("options.%s.image", strings.Title(opt.GoName))
					} else if opt.GoType == "[]*C.VipsImage" {
						optStr = fmt.Sprintf("convertImagesToVipsImages(options.%s)", strings.Title(opt.GoName))
					} else {
						optStr = fmt.Sprintf("options.%s", strings.Title(opt.GoName))
					}
					optionsCallArgs = append(optionsCallArgs, optStr)
				}

				// With options for float return
				body = fmt.Sprintf(`if options != nil {
		out, err := %s(%s)
		if err != nil {
			return 0, err
		}
		return out, nil
	}
	`, goFuncNameWithOptions, strings.Join(optionsCallArgs, ", "))
			}

			body += fmt.Sprintf(`out, err := %s(%s)
	if err != nil {
		return 0, err
	}
	return out, nil`,
				goFuncName,
				strings.Join(callArgs, ", "))
			return body
		} else if op.HasImageOutput {
			// For operations that return images
			// Get the names of the result variables
			var resultVars []string
			for _, arg := range op.RequiredOutputs {
				resultVars = append(resultVars, arg.GoName)
			}

			// Form the error return line
			var errorValues []string
			for _, arg := range op.RequiredOutputs {
				// Skip returning the length parameter if it's marked as IsOutputN
				if arg.IsOutputN {
					continue
				}
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

			var body string

			// Handle options if present
			if len(op.OptionalInputs) > 0 {
				// Create options arguments
				var optionsCallArgs = make([]string, len(callArgs))
				copy(optionsCallArgs, callArgs)

				for _, opt := range op.OptionalInputs {
					var optStr string
					if opt.GoType == "*C.VipsImage" {
						optStr = fmt.Sprintf("options.%s.image", strings.Title(opt.GoName))
					} else if opt.GoType == "[]*C.VipsImage" {
						optStr = fmt.Sprintf("convertImagesToVipsImages(options.%s)", strings.Title(opt.GoName))
					} else {
						optStr = fmt.Sprintf("options.%s", strings.Title(opt.GoName))
					}
					optionsCallArgs = append(optionsCallArgs, optStr)
				}

				// Create options block for image output
				optionsResultVars := make([]string, len(resultVars))
				copy(optionsResultVars, resultVars)

				optionsErrorLine := errorLine // Same error line applies

				// Form conversion code for each image output with options
				var optionsConversionCode strings.Builder
				for i, arg := range op.RequiredOutputs {
					// Skip returning the length parameter if it's marked as IsOutputN
					if arg.IsOutputN {
						continue
					}
					if arg.GoType == "*C.VipsImage" {
						// Convert *C.VipsImage to *Image
						optionsConversionCode.WriteString(fmt.Sprintf(`
		%sImage := newImageRef(%s, r.format, nil)`,
							arg.GoName, arg.GoName))
						optionsResultVars[i] = arg.GoName + "Image"
					} else if arg.GoType == "[]*C.VipsImage" {
						// Convert []*C.VipsImage to []*Image
						optionsConversionCode.WriteString(fmt.Sprintf(`
		%sImages := convertVipsImagesToImages(%s)`,
							arg.GoName, arg.GoName))
						optionsResultVars[i] = arg.GoName + "Images"
					}
				}

				optionsSuccessLine := "return " + strings.Join(optionsResultVars, ", ") + ", nil"

				body = fmt.Sprintf(`if options != nil {
		%s, err := %s(%s)
		if err != nil {
			%s
		}%s
		%s
	}
	`,
					strings.Join(resultVars, ", "),
					goFuncNameWithOptions,
					strings.Join(optionsCallArgs, ", "),
					optionsErrorLine,
					optionsConversionCode.String(),
					optionsSuccessLine)
			}

			// Form the function call line
			callLine := fmt.Sprintf("%s, err := %s(%s)",
				strings.Join(resultVars, ", "),
				goFuncName,
				strings.Join(callArgs, ", "))

			// Form the conversion code for each image output
			var conversionCode strings.Builder
			for i, arg := range op.RequiredOutputs {
				// Skip returning the length parameter if it's marked as IsOutputN
				if arg.IsOutputN {
					continue
				}
				if arg.GoType == "*C.VipsImage" {
					// Convert *C.VipsImage to *Image
					conversionCode.WriteString(fmt.Sprintf(`
	%sImage := newImageRef(%s, r.format, nil)`,
						arg.GoName, arg.GoName))
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

			body += callLine + `
	if err != nil {
		` + errorLine + `
	}` + conversionCode.String() + `
	` + successLine
			return body
		} else {
			// Regular operation with non-image outputs
			// Get the names of the result variables
			var resultVars []string
			for _, arg := range op.RequiredOutputs {
				// Skip returning the length parameter if it's marked as IsOutputN
				if arg.IsOutputN {
					continue
				}
				resultVars = append(resultVars, arg.GoName)
			}

			// Form the error return line
			var errorValues []string
			for _, arg := range op.RequiredOutputs {
				// Skip returning the length parameter if it's marked as IsOutputN
				if arg.IsOutputN {
					continue
				}
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

			var body string

			// Handle options if present
			if len(op.OptionalInputs) > 0 {
				// Create options arguments
				var optionsCallArgs = make([]string, len(callArgs))
				copy(optionsCallArgs, callArgs)

				for _, opt := range op.OptionalInputs {
					var optStr string
					if opt.GoType == "*C.VipsImage" {
						optStr = fmt.Sprintf("options.%s.image", strings.Title(opt.GoName))
					} else if opt.GoType == "[]*C.VipsImage" {
						optStr = fmt.Sprintf("convertImagesToVipsImages(options.%s)", strings.Title(opt.GoName))
					} else {
						optStr = fmt.Sprintf("options.%s", strings.Title(opt.GoName))
					}
					optionsCallArgs = append(optionsCallArgs, optStr)
				}

				// Options block for regular output
				body = fmt.Sprintf(`if options != nil {
		%s, err := %s(%s)
		if err != nil {
			%s
		}
		return %s, nil
	}
	`,
					strings.Join(resultVars, ", "),
					goFuncNameWithOptions,
					strings.Join(optionsCallArgs, ", "),
					errorLine,
					strings.Join(resultVars, ", "))
			}

			// Form the function call line
			callLine := fmt.Sprintf("%s, err := %s(%s)",
				strings.Join(resultVars, ", "),
				goFuncName,
				strings.Join(callArgs, ", "))

			// Form the success return line
			successLine := "return " + strings.Join(resultVars, ", ") + ", nil"

			body += callLine + `
	if err != nil {
		` + errorLine + `
	}
	` + successLine
			return body
		}
	} else {
		// Simple void return operation
		var body string

		// Handle options if present
		if len(op.OptionalInputs) > 0 {
			// Create options arguments
			var optionsCallArgs = make([]string, len(callArgs))
			copy(optionsCallArgs, callArgs)

			for _, opt := range op.OptionalInputs {
				var optStr string
				if opt.GoType == "*C.VipsImage" {
					optStr = fmt.Sprintf("options.%s.image", strings.Title(opt.GoName))
				} else if opt.GoType == "[]*C.VipsImage" {
					optStr = fmt.Sprintf("convertImagesToVipsImages(options.%s)", strings.Title(opt.GoName))
				} else {
					optStr = fmt.Sprintf("options.%s", strings.Title(opt.GoName))
				}
				optionsCallArgs = append(optionsCallArgs, optStr)
			}

			body = fmt.Sprintf(`if options != nil {
		err := %s(%s)
		if err != nil {
			return err
		}
		return nil
	}
	`, goFuncNameWithOptions, strings.Join(optionsCallArgs, ", "))
		}

		body += fmt.Sprintf(`err := %s(%s)
	if err != nil {
		return err
	}
	return nil`,
			goFuncName,
			strings.Join(callArgs, ", "))
		return body
	}
}

// generateImageArgumentsComment generates parameter descriptions following Go doc conventions
func generateImageArgumentsComment(op introspection.Operation) string {
	methodArgs := detectMethodArguments(op)
	var result strings.Builder

	if len(methodArgs) > 0 {
		// Add blank comment line for paragraph break only if there are arguments
		result.WriteString("\n//")

		for _, arg := range methodArgs {
			if arg.IsInputN {
				continue
			}
			if arg.Description != "" {
				cleanDesc := strings.TrimSpace(arg.Description)
				if cleanDesc != "" {
					if len(cleanDesc) > 0 {
						cleanDesc = strings.ToLower(string(cleanDesc[0])) + cleanDesc[1:]
						if !strings.HasSuffix(cleanDesc, ".") {
							cleanDesc += "."
						}
					}

					result.WriteString(fmt.Sprintf("\n// The %s specifies %s", arg.GoName, cleanDesc))
				}
			}
		}
	}
	return result.String() // Returns empty string if no arguments
}

// detectMethodArguments analyzes an operation's arguments to determine which should be included in the method signature
func detectMethodArguments(op introspection.Operation) []introspection.Argument {
	var methodArgs []introspection.Argument
	var firstImageFound bool
	var hasBufParam bool
	// Get all arguments except the first image input and output parameters
	for _, arg := range op.Arguments {
		// Skip output parameters
		if arg.IsOutput {
			continue
		}
		// Skip IsInputN parameters (auto-calculated)
		if arg.IsInputN {
			continue
		}
		if arg.IsBuffer {
			hasBufParam = true
			continue
		} else if arg.Name == "len" && hasBufParam {
			continue
		}
		// Skip the first image input parameter (which will be the receiver)
		if arg.IsImage && !arg.IsArray && !firstImageFound {
			firstImageFound = true
			continue
		}
		if arg.IsOutput && arg.IsImage {
			continue
		}
		// Include all other input parameters
		methodArgs = append(methodArgs, arg)
	}

	return methodArgs
}

// generateImageMethodParams formats parameters for image methods using improved detection
func generateImageMethodParams(op introspection.Operation) string {
	methodArgs := detectMethodArguments(op)
	var params []string
	for _, arg := range methodArgs {
		// Skip parameters marked as IsInputN (auto-calculated)
		if arg.IsInputN {
			continue
		}
		// Convert parameter types for image methods
		var paramType string
		if arg.GoType == "*C.VipsImage" {
			paramType = "*Image"
		} else if arg.GoType == "[]*C.VipsImage" {
			paramType = "[]*Image"
		} else if arg.CType == "void*" {
			paramType = "[]byte"
		} else if arg.IsTarget {
			paramType = "*Target"
		} else {
			paramType = arg.GoType
		}

		params = append(params, fmt.Sprintf("%s %s", arg.GoName, paramType))
	}
	if len(op.OptionalInputs) > 0 {
		params = append(params, fmt.Sprintf("options *%sOptions", op.GoName))
	}
	return strings.Join(params, ", ")
}

// generateImageMethodReturnTypes formats return types for image methods
func generateImageMethodReturnTypes(op introspection.Operation) string {
	if op.HasOneImageOutput {
		return "error"
	} else if op.HasBufferOutput {
		return "[]byte, error"
	} else if len(op.RequiredOutputs) > 0 {
		var types []string
		for _, arg := range op.RequiredOutputs {
			// Skip returning the length parameter if it's marked as IsOutputN
			if arg.IsOutputN {
				continue
			}
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

// generateMethodParams formats the parameters for a method
func generateMethodParams(op introspection.Operation) string {
	inputParams := op.RequiredInputs
	var hasBufParam bool
	var params []string
	for _, arg := range inputParams {
		// Skip IsInputN parameters (auto-calculated)
		if arg.IsInputN {
			continue
		}
		var paramType string
		if arg.GoType == "*C.VipsImage" {
			paramType = "*Image"
		} else if arg.GoType == "[]*C.VipsImage" {
			paramType = "[]*Image"
		} else if arg.IsSource {
			paramType = "*Source"
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
	if len(op.OptionalInputs) > 0 {
		params = append(params, fmt.Sprintf("options *%sOptions", op.GoName))
	}
	return strings.Join(params, ", ")
}

// generateCreatorMethodBody formats the body of a creator method
func generateCreatorMethodBody(op introspection.Operation) string {
	inputParams := op.RequiredInputs
	var hasBufParam bool
	goFuncName := "vipsgen" + op.GoName
	goFuncNameWithOptions := "vipsgen" + op.GoName + "WithOptions"

	var callArgs []string
	for _, arg := range inputParams {
		// Skip IsInputN parameters (auto-calculated)
		if arg.IsInputN {
			continue
		}
		if arg.GoType == "*C.VipsImage" {
			callArgs = append(callArgs, fmt.Sprintf("%s.image", arg.GoName))
		} else if arg.GoType == "[]*C.VipsImage" {
			callArgs = append(callArgs, fmt.Sprintf("convertImagesToVipsImages(%s)", arg.GoName))
		} else if arg.IsSource {
			callArgs = append(callArgs, fmt.Sprintf("%s.src", arg.GoName))
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

	var body string

	// Add startup line
	body = "Startup(nil)\n\t"

	// Add buffer validation for operations with buffer input
	if op.HasBufferInput {
		if bufParam := getBufferParameter(op.RequiredInputs); bufParam != nil {
			body += fmt.Sprintf(`if len(%s) == 0 {
		return nil, fmt.Errorf("%s: buffer is empty")
	}
	`, bufParam.GoName, op.Name)
		}
	}

	imageTypeString := op.ImageTypeString
	if strings.Contains(op.Name, "thumbnail") {
		imageTypeString = "vipsDetermineImageType(vipsImage)"
	}

	// Handle options if present
	if len(op.OptionalInputs) > 0 {
		// Create options arguments
		var optionsCallArgs = make([]string, len(callArgs))
		copy(optionsCallArgs, callArgs)

		for _, opt := range op.OptionalInputs {
			var optStr string
			if opt.GoType == "*C.VipsImage" {
				optStr = fmt.Sprintf("options.%s.image", strings.Title(opt.GoName))
			} else if opt.GoType == "[]*C.VipsImage" {
				optStr = fmt.Sprintf("convertImagesToVipsImages(options.%s)", strings.Title(opt.GoName))
			} else {
				optStr = fmt.Sprintf("options.%s", strings.Title(opt.GoName))
			}
			optionsCallArgs = append(optionsCallArgs, optStr)
		}

		// Add options handling block
		body += fmt.Sprintf(`if options != nil {
		vipsImage, err := %s(%s)
		if err != nil {
			return nil, err
		}
		return newImageRef(vipsImage, %s, %s), nil
	}
	`,
			goFuncNameWithOptions,
			strings.Join(optionsCallArgs, ", "),
			imageTypeString,
			imageRefBuf)
	}

	// Add regular function call
	body += fmt.Sprintf(`vipsImage, err := %s(%s)
	if err != nil {
		return nil, err
	}
	return newImageRef(vipsImage, %s, %s), nil`,
		goFuncName,
		strings.Join(callArgs, ", "),
		imageTypeString,
		imageRefBuf)

	return body
}

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

// generateCFunctionDeclaration generates header declarations for vips operations
func generateCFunctionDeclaration(op introspection.Operation) string {
	var result strings.Builder
	if len(op.Arguments) == 0 {
		result.WriteString(fmt.Sprintf("int vipsgen_%s();", op.Name))
	} else {
		result.WriteString(generateCFunctionSignature(op, true))
		result.WriteString(";")
	}

	// with_options function declaration if needed
	if len(op.OptionalInputs) > 0 {
		result.WriteString("\n")

		// Generate function declaration with array length parameters
		result.WriteString(fmt.Sprintf("int vipsgen_%s_with_options(", op.Name))

		// Regular arguments
		if len(op.Arguments) > 0 {
			for i, arg := range op.Arguments {
				if i > 0 {
					result.WriteString(", ")
				}
				result.WriteString(fmt.Sprintf("%s %s", arg.CType, arg.Name))
			}
		}

		// Add optional arguments and array length parameters
		for i, opt := range op.OptionalInputs {
			if i > 0 || len(op.Arguments) > 0 {
				result.WriteString(", ")
			}
			result.WriteString(fmt.Sprintf("%s %s", opt.CType, opt.Name))

			// Add array length parameter if needed
			if strings.HasPrefix(opt.GoType, "[]") {
				// Check if this array type needs a length parameter
				if opt.GoType == "[]float64" || opt.GoType == "[]float32" ||
					opt.GoType == "[]int" || opt.GoType == "[]BlendMode" ||
					opt.GoType == "[]*C.VipsImage" || opt.GoType == "[]*Image" {
					result.WriteString(fmt.Sprintf(", int %s_n", opt.Name))
				}
			}
		}
		result.WriteString(");")
	}
	return result.String()
}

// generateCFunctionImplementation generates C implementations for vips operations
func generateCFunctionImplementation(op introspection.Operation) string {
	var result strings.Builder

	// Handle basic function (no options)
	if len(op.Arguments) == 0 {
		result.WriteString(fmt.Sprintf("int vipsgen_%s() {\n", op.Name))
		result.WriteString(fmt.Sprintf("    return vips_%s(NULL);\n}", op.Name))
	} else {
		result.WriteString(generateCFunctionSignature(op, true))
		result.WriteString(" {\n")

		// Handle direct C function calls for simple operations without options
		result.WriteString(fmt.Sprintf("    return vips_%s(", op.Name))
		for i, arg := range op.Arguments {
			if i > 0 {
				result.WriteString(", ")
			}
			if arg.IsSource {
				// Add type casting for VipsSourceCustom
				result.WriteString("(VipsSource*) " + arg.Name)
			} else if arg.IsTarget {
				// Add type casting for VipsTargetCustom
				result.WriteString("(VipsTarget*) " + arg.Name)
			} else {
				result.WriteString(arg.Name)
			}
		}
		result.WriteString(", NULL);\n}")
	}

	// Generate the with_options variant
	if len(op.OptionalInputs) > 0 {
		result.WriteString("\n\n")
		// Generate function signature with array length parameters for array arguments
		result.WriteString(fmt.Sprintf("int vipsgen_%s_with_options(", op.Name))

		// Add regular arguments
		if len(op.Arguments) > 0 {
			for i, arg := range op.Arguments {
				if i > 0 {
					result.WriteString(", ")
				}
				result.WriteString(fmt.Sprintf("%s %s", arg.CType, arg.Name))
			}
		}

		// Add optional arguments and array length parameters
		for i, opt := range op.OptionalInputs {
			if i > 0 || len(op.Arguments) > 0 {
				result.WriteString(", ")
			}
			result.WriteString(fmt.Sprintf("%s %s", opt.CType, opt.Name))

			// Add array length parameter if needed
			if strings.HasPrefix(opt.GoType, "[]") {
				// Check if this array type needs a length parameter
				if opt.GoType == "[]float64" || opt.GoType == "[]float32" ||
					opt.GoType == "[]int" || opt.GoType == "[]BlendMode" ||
					opt.GoType == "[]*C.VipsImage" || opt.GoType == "[]*Image" {
					result.WriteString(fmt.Sprintf(", int %s_n", opt.Name))
				}
			}
		}
		result.WriteString(") {\n")

		// Create operation using vips_operation_new
		result.WriteString(fmt.Sprintf("    VipsOperation *operation = vips_operation_new(\"%s\");\n", op.Name))
		result.WriteString("    if (!operation) return 1;\n")

		// Detect if this is a buffer operation that needs special handling
		isBufferLoadOperation := strings.Contains(op.Name, "load_buffer")
		isBufferSaveOperation := strings.Contains(op.Name, "save_buffer")

		// Special handling for buffer load operations - create a VipsBlob
		if isBufferLoadOperation {
			result.WriteString("    VipsBlob *blob = vips_blob_new(NULL, buf, len);\n")
			result.WriteString("    if (!blob) { g_object_unref(operation); return 1; }\n")
		}

		// Create VipsArray objects for array inputs from BOTH required and optional inputs
		for _, arg := range op.RequiredInputs {
			if strings.HasPrefix(arg.GoType, "[]") {
				arrayType := getArrayType(arg.GoType)
				if arrayType == "double" {
					result.WriteString(fmt.Sprintf("    VipsArrayDouble *%s_array = NULL;\n", arg.Name))
					result.WriteString(fmt.Sprintf("    if (%s != NULL && n > 0) { %s_array = vips_array_double_new(%s, n); }\n", arg.Name, arg.Name, arg.Name))
				} else if arrayType == "int" {
					result.WriteString(fmt.Sprintf("    VipsArrayInt *%s_array = NULL;\n", arg.Name))
					// Special case for composite operation: mode array should be n-1
					if op.Name == "composite" && arg.Name == "mode" {
						result.WriteString(fmt.Sprintf("    if (%s != NULL && n > 1) { %s_array = vips_array_int_new(%s, n-1); }\n", arg.Name, arg.Name, arg.Name))
					} else {
						result.WriteString(fmt.Sprintf("    if (%s != NULL && n > 0) { %s_array = vips_array_int_new(%s, n); }\n", arg.Name, arg.Name, arg.Name))
					}
				} else if arrayType == "image" {
					result.WriteString(fmt.Sprintf("    VipsArrayImage *%s_array = NULL;\n", arg.Name))
					result.WriteString(fmt.Sprintf("    if (%s != NULL && n > 0) { %s_array = vips_array_image_new(%s, n); }\n", arg.Name, arg.Name, arg.Name))
				}
			}
		}
		for _, opt := range op.OptionalInputs {
			if strings.HasPrefix(opt.GoType, "[]") {
				arrayType := getArrayType(opt.GoType)
				if arrayType == "double" {
					result.WriteString(fmt.Sprintf("    VipsArrayDouble *%s_array = NULL;\n", opt.Name))
					result.WriteString(fmt.Sprintf("    if (%s != NULL && %s_n > 0) { %s_array = vips_array_double_new(%s, %s_n); }\n", opt.Name, opt.Name, opt.Name, opt.Name, opt.Name))
				} else if arrayType == "int" {
					result.WriteString(fmt.Sprintf("    VipsArrayInt *%s_array = NULL;\n", opt.Name))
					result.WriteString(fmt.Sprintf("    if (%s != NULL && %s_n > 0) { %s_array = vips_array_int_new(%s, %s_n); }\n", opt.Name, opt.Name, opt.Name, opt.Name, opt.Name))
				} else if arrayType == "image" {
					result.WriteString(fmt.Sprintf("    VipsArrayImage *%s_array = NULL;\n", opt.Name))
					result.WriteString(fmt.Sprintf("    if (%s != NULL && %s_n > 0) { %s_array = vips_array_image_new(%s, %s_n); }\n", opt.Name, opt.Name, opt.Name, opt.Name, opt.Name))
				}
			}
		}

		// Combine required and optional parameters in a single condition
		var allParamsList []string

		// Add required parameters first
		for _, arg := range op.Arguments {
			if arg.IsOutput {
				continue // Skip output arguments, they'll be handled after build
			}
			if arg.IsInputN {
				continue // Skip n
			}

			// Special handling for different types of arguments
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
				// For buffer load operations, set the VipsBlob as the "buffer" property
				allParamsList = append(allParamsList,
					fmt.Sprintf("vips_object_set(VIPS_OBJECT(operation), \"buffer\", blob, NULL)"))
			} else if arg.Name == "len" && isBufferLoadOperation {
				// Skip length parameter for buffer load operations, as it's included in the VipsBlob
				continue
			} else if arg.GoType == "string" {
				// String parameter
				allParamsList = append(allParamsList,
					fmt.Sprintf("vips_object_set(VIPS_OBJECT(operation), \"%s\", %s, NULL)", arg.Name, arg.Name))
			} else if arg.GoType == "*C.VipsImage" {
				// Image parameter
				allParamsList = append(allParamsList,
					fmt.Sprintf("vips_object_set(VIPS_OBJECT(operation), \"%s\", %s, NULL)", arg.Name, arg.Name))
			} else {
				// Other scalar parameters
				allParamsList = append(allParamsList,
					fmt.Sprintf("vips_object_set(VIPS_OBJECT(operation), \"%s\", %s, NULL)", arg.Name, arg.Name))
			}
		}

		// Add optional parameters using type-specific setter functions
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
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_int(operation, \"%s\", %s)", opt.Name, opt.Name))
			} else if opt.GoType == "*C.VipsImage" {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_image(operation, \"%s\", %s)", opt.Name, opt.Name))
			} else if opt.GoType == "*Interpolate" || opt.GoType == "*C.VipsInterpolate" {
				// Handle interpolate parameters
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_interpolate(operation, \"%s\", %s)", opt.Name, opt.Name))
			} else if opt.IsSource {
				// Handle source parameters
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_source(operation, \"%s\", %s)", opt.Name, opt.Name))
			} else if opt.IsTarget {
				// Handle target parameters
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_target(operation, \"%s\", %s)", opt.Name, opt.Name))
			} else if opt.GoType == "int" {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_int(operation, \"%s\", %s)", opt.Name, opt.Name))
			} else if opt.GoType == "float64" {
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_double(operation, \"%s\", %s)", opt.Name, opt.Name))
			} else if strings.Contains(opt.CType, "guint64") {
				// Handle guint64 parameters
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_guint64(operation, \"%s\", %s)", opt.Name, opt.Name))
			} else if strings.Contains(opt.CType, "unsigned int") || strings.Contains(opt.CType, "guint") {
				// Handle unsigned int parameters
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_int(operation, \"%s\", %s)", opt.Name, opt.Name))
			} else if strings.Contains(opt.CType, "*") || strings.Contains(opt.GoType, "*") {
				// This is a pointer type - use general pointer handler
				allParamsList = append(allParamsList,
					fmt.Sprintf("vips_object_set(VIPS_OBJECT(operation), \"%s\", %s, NULL)", opt.Name, opt.Name))
			} else {
				// For any other non-pointer scalar types, default to int
				allParamsList = append(allParamsList,
					fmt.Sprintf("vipsgen_set_int(operation, \"%s\", %s)", opt.Name, opt.Name))
			}
		}

		// Join all parameters with the || operator for short-circuit evaluation
		if len(allParamsList) > 0 {
			result.WriteString("    if (\n        ")
			result.WriteString(strings.Join(allParamsList, " ||\n        "))
			result.WriteString("\n    ) {\n")

			// Additional cleanup for VipsBlob if this is a buffer load operation
			if isBufferLoadOperation {
				result.WriteString("        vips_area_unref((VipsArea *)blob);\n")
			}

			result.WriteString("        g_object_unref(operation);\n")

			// Free all array resources on error - handle BOTH required AND optional arrays
			for _, cleanupArg := range op.RequiredInputs {
				if strings.HasPrefix(cleanupArg.GoType, "[]") {
					arrayType := getArrayType(cleanupArg.GoType)
					if arrayType != "unknown" {
						result.WriteString(fmt.Sprintf("        if (%s_array != NULL) { vips_area_unref(VIPS_AREA(%s_array)); }\n", cleanupArg.Name, cleanupArg.Name))
					}
				}
			}
			for _, cleanupOpt := range op.OptionalInputs {
				if strings.HasPrefix(cleanupOpt.GoType, "[]") {
					arrayType := getArrayType(cleanupOpt.GoType)
					if arrayType != "unknown" {
						result.WriteString(fmt.Sprintf("        if (%s_array != NULL) { vips_area_unref(VIPS_AREA(%s_array)); }\n", cleanupOpt.Name, cleanupOpt.Name))
					}
				}
			}

			result.WriteString("        return 1;\n    }\n")
		}

		// Unreference VipsBlob for buffer operations after the operation takes its reference
		if isBufferLoadOperation {
			result.WriteString("    vips_area_unref((VipsArea *)blob);\n")
		}

		// Generate the call to the helper function
		if isBufferSaveOperation {
			// For buffer save operations, use the vipsgen_operation_save_buffer helper
			result.WriteString("    int result = vipsgen_operation_save_buffer(operation, buf, len);\n")
		} else {
			// Collect the output parameters
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
			// Add NULL terminator
			outputParams = append(outputParams, "NULL")
			result.WriteString(fmt.Sprintf("    int result = vipsgen_operation_execute(operation, %s);\n", strings.Join(outputParams, ", ")))
		}

		// Clean up array objects - handle BOTH required AND optional arrays
		// Clean up arrays from required inputs
		for _, arg := range op.RequiredInputs {
			if strings.HasPrefix(arg.GoType, "[]") {
				arrayType := getArrayType(arg.GoType)
				if arrayType != "unknown" {
					result.WriteString(fmt.Sprintf("    if (%s_array != NULL) { vips_area_unref(VIPS_AREA(%s_array)); }\n", arg.Name, arg.Name))
				}
			}
		}

		// Clean up arrays from optional inputs
		for _, opt := range op.OptionalInputs {
			if strings.HasPrefix(opt.GoType, "[]") {
				arrayType := getArrayType(opt.GoType)
				if arrayType != "unknown" {
					result.WriteString(fmt.Sprintf("    if (%s_array != NULL) { vips_area_unref(VIPS_AREA(%s_array)); }\n", opt.Name, opt.Name))
				}
			}
		}

		result.WriteString("    return result;\n}")
	}

	return result.String()
}

// generateOptionalInputsStruct generates a parameter struct for an operation
func generateOptionalInputsStruct(op introspection.Operation) string {
	if len(op.OptionalInputs) == 0 {
		return ""
	}
	var result strings.Builder

	// Determine the struct name
	var structName = op.GoName + "Options"

	result.WriteString(fmt.Sprintf("// %s optional arguments for vips_%s\n", structName, op.Name))
	result.WriteString(fmt.Sprintf("type %s struct {\n", structName))

	// Add all optional parameters to the struct
	for _, opt := range op.OptionalInputs {
		fieldName := strings.Title(opt.GoName)
		var fieldType string
		// Convert parameter types for struct
		if opt.GoType == "*C.VipsImage" {
			fieldType = "*Image"
		} else if opt.GoType == "[]*C.VipsImage" {
			fieldType = "[]*Image"
		} else if opt.CType == "void*" {
			fieldType = "[]byte"
		} else {
			fieldType = opt.GoType
		}
		// Handle enum types by using the proper Go enum type
		if opt.IsEnum && opt.EnumType != "" {
			fieldType = opt.EnumType
		}
		// Add comment with description if available
		if opt.Description != "" {
			result.WriteString(fmt.Sprintf("\t// %s %s\n", fieldName, opt.Description))
		}
		result.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, fieldType))
	}
	result.WriteString("}\n\n")

	// Create a constructor with default values
	result.WriteString(fmt.Sprintf("// Default%s creates default value for vips_%s optional arguments\n",
		structName, op.Name))
	result.WriteString(fmt.Sprintf("func Default%s() *%s {\n", structName, structName))
	result.WriteString(fmt.Sprintf("\treturn &%s{\n", structName))
	// Add default values for each parameter
	for _, opt := range op.OptionalInputs {
		fieldName := strings.Title(opt.GoName)

		// Only include non-zero defaults
		if opt.DefaultValue != nil {
			switch v := opt.DefaultValue.(type) {
			case bool:
				if v {
					result.WriteString(fmt.Sprintf("\t\t%s: %t,\n", fieldName, v))
				}
			case int:
				if v != 0 {
					// For enum types, cast the integer to the enum type
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
			// Skip returning the length parameter if it's marked as IsOutputN
			if arg.IsOutputN {
				continue
			}
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
