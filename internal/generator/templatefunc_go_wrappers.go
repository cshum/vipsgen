package generator

import (
	"fmt"
	"strings"

	"github.com/cshum/vipsgen/internal/introspection"
)

// getOutputScalarCType returns the cgo scalar type used for temporary output storage.
func getOutputScalarCType(arg introspection.Argument) string {
	cType := strings.TrimSpace(strings.TrimSuffix(arg.CType, "*"))

	switch {
	case strings.Contains(cType, "gboolean"):
		return "C.gboolean"
	case strings.Contains(cType, "unsigned int"), strings.Contains(cType, "guint"):
		return "C.uint"
	case strings.Contains(cType, "gint"):
		return "C.gint"
	case cType == "int":
		return "C.int"
	case cType == "double":
		return "C.double"
	case cType == "float":
		return "C.float"
	}

	// Fallback to Go type when ctype metadata is not specific enough.
	switch arg.GoType {
	case "bool":
		return "C.gboolean"
	case "int":
		return "C.int"
	case "float64":
		return "C.double"
	case "float32":
		return "C.float"
	default:
		return ""
	}
}

func generateScalarConversionLine(goType, targetExpr, sourceExpr string) string {
	switch goType {
	case "float64":
		return fmt.Sprintf("%s = float64(%s)", targetExpr, sourceExpr)
	case "int":
		return fmt.Sprintf("%s = int(%s)", targetExpr, sourceExpr)
	case "bool":
		return fmt.Sprintf("%s = %s != 0", targetExpr, sourceExpr)
	default:
		return ""
	}
}

func isVectorOutputArg(arg introspection.Argument) bool {
	return arg.Name == "vector" || arg.Name == "out_array"
}

func generatePostCallScalarConversions(op introspection.Operation, withOptions bool) string {
	var conversions []string

	if !op.HasOneImageOutput && !op.HasBufferOutput {
		for _, arg := range op.RequiredOutputs {
			if isVectorOutputArg(arg) {
				continue
			}
			if getOutputScalarCType(arg) == "" {
				continue
			}
			if line := generateScalarConversionLine(arg.GoType, arg.GoName, "*c"+arg.GoName); line != "" {
				conversions = append(conversions, line)
			}
		}
	}

	if withOptions {
		for _, opt := range getSupportedOptionalOutputs(op) {
			if getOutputScalarCType(opt) == "" {
				continue
			}
			line := generateScalarConversionLine(opt.GoType, "*"+opt.GoName, "c"+opt.GoName+"Value")
			if line == "" {
				continue
			}
			conversions = append(conversions, fmt.Sprintf("if %s != nil {\n\t\t%s\n\t}", opt.GoName, line))
		}
	}

	return strings.Join(conversions, "\n\t")
}

func generateOutputErrorReturn(outputs []introspection.Argument, errorExpr string) string {
	var returnValues []string
	for _, arg := range outputs {
		if arg.IsOutputN {
			continue
		}
		if isVectorOutputArg(arg) {
			returnValues = append(returnValues, "nil")
		} else {
			returnValues = append(returnValues, formatDefaultValue(arg.GoType))
		}
	}
	return "return " + strings.Join(returnValues, ", ") + ", " + errorExpr
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

	// Convert temporary C scalar outputs back into Go values.
	if conversions := generatePostCallScalarConversions(op, withOptions); conversions != "" {
		result.WriteString(conversions)
		result.WriteString("\n\t")
	}

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
		return generateOutputErrorReturn(outputs, "handleVipsError()")
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
		return generateOutputErrorReturn(op.RequiredOutputs, "err")
	} else {
		return "return err"
	}
}

// Helper function to generate safe default values for array types
func generateSafeDefaultForArray(goType string) string {
	switch goType {
	case "[]float64":
		return "[]float64{}"
	case "[]float32":
		return "[]float32{}"
	case "[]int":
		return "[]int{}"
	case "[]BlendMode":
		return "[]BlendMode{}"
	case "[]*Image", "[]*C.VipsImage":
		return "[]*C.VipsImage{}"
	default:
		if strings.HasPrefix(goType, "[]") {
			return goType + "{}"
		}
		return "[]float64{}"
	}
}

func generateTypedArrayConversionDeclaration(arg introspection.Argument, errorReturn, convertFunc, freeFunc string) string {
	lengthVar := fmt.Sprintf("c%sLength", arg.GoName)
	if arg.IsRequired {
		lengthVar = "_"
	}

	if arg.IsRequired {
		return fmt.Sprintf(
			"if %s == nil {\n"+
				"\t\t%s = %s\n"+
				"\t}\n"+
				"\tc%s, %s, err := %s(%s)\n"+
				"\tif err != nil {\n"+
				"\t\t%s\n"+
				"\t}\n"+
				"\tif c%s != nil {\n"+
				"\t\tdefer %s(c%s)\n"+
				"\t}",
			arg.GoName,
			arg.GoName,
			generateSafeDefaultForArray(arg.GoType),
			arg.GoName,
			lengthVar,
			convertFunc,
			arg.GoName,
			errorReturn,
			arg.GoName,
			freeFunc,
			arg.GoName,
		)
	}

	return fmt.Sprintf(
		"c%s, %s, err := %s(%s)\n"+
			"\tif err != nil {\n"+
			"\t\t%s\n"+
			"\t}\n"+
			"\tif c%s != nil {\n"+
			"\t\tdefer %s(c%s)\n"+
			"\t}",
		arg.GoName,
		lengthVar,
		convertFunc,
		arg.GoName,
		errorReturn,
		arg.GoName,
		freeFunc,
		arg.GoName,
	)
}

func generateOptionalOutputCallArg(opt introspection.Argument) string {
	if getOutputScalarCType(opt) != "" {
		return "c" + opt.GoName
	}
	return "&" + opt.GoName
}

func generateRequiredOutputCallArg(arg introspection.Argument) string {
	if getOutputScalarCType(arg) != "" {
		return "c" + arg.GoName
	}
	return "&" + arg.GoName
}

func generateOptionalOutputDeclaration(opt introspection.Argument) string {
	cType := getOutputScalarCType(opt)
	if cType == "" {
		return ""
	}

	return fmt.Sprintf("var c%sValue %s\n\tvar c%s *%s\n\tif %s != nil {\n\t\tc%s = &c%sValue\n\t}",
		opt.GoName,
		cType,
		opt.GoName,
		cType,
		opt.GoName,
		opt.GoName,
		opt.GoName,
	)
}

func generateRequiredOutputDeclaration(arg introspection.Argument) string {
	if isVectorOutputArg(arg) {
		return "var out *C.double"
	}
	return fmt.Sprintf("var %s %s", arg.GoName, arg.GoType)
}

func arrayNeedsLengthParam(arg introspection.Argument) bool {
	return !arg.IsRequired && (arg.GoType == "[]float64" || arg.GoType == "[]float32" ||
		arg.GoType == "[]int" || arg.GoType == "[]BlendMode" ||
		arg.GoType == "[]*C.VipsImage" || arg.GoType == "[]*Image")
}

func generateArrayInputCallArgs(arg introspection.Argument, withOptions bool) []string {
	arrayVarName := "c" + arg.GoName
	argStr := arrayVarName

	if !withOptions && arg.GoType == "[]*C.VipsImage" {
		argStr = "(**C.VipsImage)(" + arrayVarName + ")"
	}

	callArgs := []string{argStr}
	if arrayNeedsLengthParam(arg) {
		callArgs = append(callArgs, "c"+arg.GoName+"Length")
	}

	return callArgs
}

// generateGoArgList formats a list of function arguments for a Go function
// e.g., "in *C.VipsImage, c []float64, n int"
func generateGoArgList(op introspection.Operation, withOptions bool) string {
	args := op.Arguments
	if withOptions {
		args = append(args, op.OptionalInputs...)
	}
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
		if arg.IsInputN {
			continue
		}
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

	if withOptions {
		supportedOptionalOutputs := getSupportedOptionalOutputs(op)
		for _, opt := range supportedOptionalOutputs {
			params = append(params, fmt.Sprintf("%s *%s", opt.GoName, opt.GoType))
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
			if arg.IsOutputN {
				continue
			}
			if isVectorOutputArg(arg) {
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
		hasVipsBlob := false
		for _, arg := range op.RequiredOutputs {
			if arg.CType == "VipsBlob**" && arg.IsOutput {
				hasVipsBlob = true
				decls = append(decls, fmt.Sprintf("var %s *C.VipsBlob", arg.GoName))
				break
			}
		}

		if !hasVipsBlob {
			decls = append(decls, "var buf unsafe.Pointer")
			decls = append(decls, "var length C.size_t")
		}
	} else {
		for _, arg := range op.RequiredOutputs {
			if arg.CType == "VipsBlob**" && arg.IsOutput {
				decls = append(decls, fmt.Sprintf("var %s *C.VipsBlob", arg.GoName))
				continue
			}
			decls = append(decls, generateRequiredOutputDeclaration(arg))
			if !isVectorOutputArg(arg) {
				if cType := getOutputScalarCType(arg); cType != "" {
					decls = append(decls, fmt.Sprintf("c%s := new(%s)", arg.GoName, cType))
				}
			}
		}
	}

	if stringConv := formatStringConversions(op.Arguments); stringConv != "" {
		decls = append(decls, stringConv)
	}

	args := op.Arguments
	if withOptions {
		args = append(args, op.OptionalInputs...)
	}

	for _, arg := range args {
		if !arg.IsOutput && strings.HasPrefix(arg.GoType, "[]") {
			if arg.GoType == "[]byte" && strings.Contains(arg.Name, "buf") {
				continue
			}

			errorReturn := generateErrorReturnForUtilityCall(op)

			if arg.GoType == "[]float64" || arg.GoType == "[]float32" {
				decls = append(decls, generateTypedArrayConversionDeclaration(arg, errorReturn, "convertToDoubleArray", "freeDoubleArray"))
			} else if arg.GoType == "[]int" {
				decls = append(decls, generateTypedArrayConversionDeclaration(arg, errorReturn, "convertToIntArray", "freeIntArray"))
			} else if arg.GoType == "[]BlendMode" {
				decls = append(decls, generateTypedArrayConversionDeclaration(arg, errorReturn, "convertToBlendModeArray", "freeIntArray"))
			} else if arg.GoType == "[]*Image" || arg.GoType == "[]*C.VipsImage" {
				decls = append(decls, generateTypedArrayConversionDeclaration(arg, errorReturn, "convertToImageArray", "freeImageArray"))
			} else {
				decls = append(decls, fmt.Sprintf(
					"var c%s unsafe.Pointer\n"+
						"\tif len(%s) > 0 {\n"+
						"\t\tc%s = unsafe.Pointer(&%s[0])\n"+
						"\t}",
					arg.GoName, arg.GoName, arg.GoName, arg.GoName))
			}
		}
	}

	if withOptions {
		if stringConv := formatStringConversions(op.OptionalInputs); stringConv != "" {
			decls = append(decls, stringConv)
		}

		supportedOptionalOutputs := getSupportedOptionalOutputs(op)
		for _, opt := range supportedOptionalOutputs {
			if decl := generateOptionalOutputDeclaration(opt); decl != "" {
				decls = append(decls, decl)
			}
		}
	}

	return strings.Join(decls, "\n\t")
}

// formatStringConversions formats C string conversions for string parameters
func formatStringConversions(args []introspection.Argument) string {
	var conversions []string
	for _, arg := range args {
		if !arg.IsOutput && arg.GoType == "string" {
			conversions = append(conversions, fmt.Sprintf("c%s := C.CString(%s)\n\tdefer freeCString(c%s)",
				arg.GoName, arg.GoName, arg.GoName))
		}
	}
	return strings.Join(conversions, "\n\t")
}

// generateFunctionCallArgs formats the arguments for the C function call
func generateFunctionCallArgs(op introspection.Operation, withOptions bool) string {
	args := op.Arguments
	if withOptions {
		args = append(args, op.OptionalInputs...)
	}
	var callArgs []string

	for _, arg := range args {
		var argStr string

		if arg.IsOutput {
			if arg.Name == "out" || op.HasOneImageOutput {
				if arg.GoType == "*C.VipsImage" {
					argStr = "&out"
				} else {
					argStr = "c" + arg.GoName
				}
			} else if isVectorOutputArg(arg) {
				argStr = "&out"
			} else if arg.CType == "size_t*" && arg.Name == "len" {
				argStr = "&length"
			} else {
				argStr = generateRequiredOutputCallArg(arg)
			}
			callArgs = append(callArgs, argStr)
		} else {
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
				argStr = "unsafe.Pointer(&src[0])"
				callArgs = append(callArgs, argStr)
			} else if arg.GoType == "*Interpolate" {
				argStr = "vipsInterpolateToC(" + arg.GoName + ")"
				callArgs = append(callArgs, argStr)
			} else if arg.Name == "len" && arg.CType == "size_t" {
				argStr = "C.size_t(len(src))"
				callArgs = append(callArgs, argStr)
			} else if strings.HasPrefix(arg.GoType, "[]") {
				callArgs = append(callArgs, generateArrayInputCallArgs(arg, withOptions)...)
			} else if arg.IsEnum {
				argStr = "C." + arg.Type + "(" + arg.GoName + ")"
				callArgs = append(callArgs, argStr)
			} else if arg.CType == "void**" && arg.Name == "buf" {
				argStr = "&buf"
				callArgs = append(callArgs, argStr)
			} else if arg.CType == "size_t*" && arg.Name == "len" {
				argStr = "&length"
				callArgs = append(callArgs, argStr)
			} else {
				argStr = "C." + arg.CType + "(" + arg.GoName + ")"
				callArgs = append(callArgs, argStr)
			}
		}
	}

	if withOptions {
		supportedOptionalOutputs := getSupportedOptionalOutputs(op)
		for _, opt := range supportedOptionalOutputs {
			callArgs = append(callArgs, generateOptionalOutputCallArg(opt))
		}
	}

	return strings.Join(callArgs, ", ")
}

// generateReturnValues formats the return values for the Go function
func generateReturnValues(op introspection.Operation) string {
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
		var conversionLines []string
		var values []string

		for _, arg := range op.RequiredOutputs {
			if arg.IsOutputN {
				continue
			}
			if isVectorOutputArg(arg) {
				nParam := "n"
				for _, outArg := range op.RequiredOutputs {
					if outArg.Name == "n" {
						nParam = outArg.GoName
						break
					}
				}
				conversionLines = append(conversionLines,
					fmt.Sprintf("result := make([]float64, %s)", nParam))
				conversionLines = append(conversionLines,
					fmt.Sprintf("copy(result, (*[1024]float64)(unsafe.Pointer(out))[:%s:%s])", nParam, nParam))
				conversionLines = append(conversionLines,
					"gFreePointer(unsafe.Pointer(out))")
				values = append(values, "result")
			} else {
				values = append(values, arg.GoName)
			}
		}

		var result strings.Builder
		if len(conversionLines) > 0 {
			for _, line := range conversionLines {
				result.WriteString(line + "\n\t")
			}
		}
		result.WriteString("return " + strings.Join(values, ", ") + ", nil")
		return result.String()
	} else {
		return "return nil"
	}
}
