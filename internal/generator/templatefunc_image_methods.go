package generator

import (
	"fmt"
	"strings"

	"github.com/cshum/vipsgen/internal/introspection"
)

const (
	imageOptionArgSafePointer = "safe-pointer"
	imageOptionArgImageField  = "image-field"
)

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

func buildImageOptionsCallArgs(baseArgs []string, optionalInputs, optionalOutputs []introspection.Argument, imageArgMode string) []string {
	optionsCallArgs := make([]string, len(baseArgs))
	copy(optionsCallArgs, baseArgs)

	for _, opt := range optionalInputs {
		fieldName := strings.Title(opt.GoName)
		switch {
		case opt.GoType == "*C.VipsImage" && imageArgMode == imageOptionArgSafePointer:
			optionsCallArgs = append(optionsCallArgs, fmt.Sprintf("getImagePointer(options.%s)", fieldName))
		case opt.GoType == "*C.VipsImage":
			optionsCallArgs = append(optionsCallArgs, fmt.Sprintf("options.%s.image", fieldName))
		case opt.GoType == "[]*C.VipsImage":
			optionsCallArgs = append(optionsCallArgs, fmt.Sprintf("convertImagesToVipsImages(options.%s)", fieldName))
		default:
			optionsCallArgs = append(optionsCallArgs, fmt.Sprintf("options.%s", fieldName))
		}
	}

	for _, opt := range optionalOutputs {
		optionsCallArgs = append(optionsCallArgs, fmt.Sprintf("&options.%s", strings.Title(opt.GoName)))
	}

	return optionsCallArgs
}

func generateImageMethodErrorLine(outputs []introspection.Argument, includeImagePointers bool) string {
	var errorValues []string
	for _, arg := range outputs {
		if arg.IsOutputN {
			continue
		}
		if includeImagePointers && arg.GoType == "*C.VipsImage" {
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
	return "return " + strings.Join(errorValues, ", ") + ", err"
}

func generateImageOutputConversions(outputs []introspection.Argument, resultVars []string, indent string) string {
	var conversionCode strings.Builder
	for i, arg := range outputs {
		if arg.IsOutputN {
			continue
		}
		if arg.GoType == "*C.VipsImage" {
			conversionCode.WriteString(fmt.Sprintf("\n%s%sImage := newImageRef(%s, r.format, nil)", indent, arg.GoName, arg.GoName))
			resultVars[i] = arg.GoName + "Image"
		} else if arg.GoType == "[]*C.VipsImage" {
			conversionCode.WriteString(fmt.Sprintf("\n%s%sImages := convertVipsImagesToImages(%s)", indent, arg.GoName, arg.GoName))
			resultVars[i] = arg.GoName + "Images"
		}
	}
	return conversionCode.String()
}

// generateImageMethodBody formats the body of an image method using improved argument detection
func generateImageMethodBody(op introspection.Operation) string {
	methodArgs := detectMethodArguments(op)
	goFuncName := "vipsgen" + op.GoName
	goFuncNameWithOptions := "vipsgen" + op.GoName + "WithOptions"

	var callArgs []string
	callArgs = append(callArgs, "r.image")

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

	if op.HasOneImageOutput {
		var body string

		supportedOptionalOutputs := getSupportedOptionalOutputs(op)
		if len(op.OptionalInputs) > 0 || len(supportedOptionalOutputs) > 0 {
			optionsCallArgs := buildImageOptionsCallArgs(callArgs, op.OptionalInputs, supportedOptionalOutputs, imageOptionArgSafePointer)

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

		if len(op.OptionalInputs) > 0 {
			optionsCallArgs := buildImageOptionsCallArgs(callArgs, op.OptionalInputs, nil, imageOptionArgSafePointer)

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
		if hasVectorReturn(op) {
			var body string

			if len(op.OptionalInputs) > 0 {
				optionsCallArgs := buildImageOptionsCallArgs(callArgs, op.OptionalInputs, nil, imageOptionArgImageField)

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
			var body string

			supportedOptionalOutputs := getSupportedOptionalOutputs(op)
			if len(op.OptionalInputs) > 0 || len(supportedOptionalOutputs) > 0 {
				optionsCallArgs := buildImageOptionsCallArgs(callArgs, op.OptionalInputs, supportedOptionalOutputs, imageOptionArgImageField)

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
			var resultVars []string
			for _, arg := range op.RequiredOutputs {
				resultVars = append(resultVars, arg.GoName)
			}

			errorLine := generateImageMethodErrorLine(op.RequiredOutputs, true)

			var body string

			if len(op.OptionalInputs) > 0 {
				optionsCallArgs := buildImageOptionsCallArgs(callArgs, op.OptionalInputs, nil, imageOptionArgImageField)

				optionsResultVars := make([]string, len(resultVars))
				copy(optionsResultVars, resultVars)

				optionsErrorLine := errorLine
				optionsConversionCode := generateImageOutputConversions(op.RequiredOutputs, optionsResultVars, "\t\t")

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
					optionsConversionCode,
					optionsSuccessLine)
			}

			callLine := fmt.Sprintf("%s, err := %s(%s)",
				strings.Join(resultVars, ", "),
				goFuncName,
				strings.Join(callArgs, ", "))

			conversionCode := generateImageOutputConversions(op.RequiredOutputs, resultVars, "\t")

			successLine := "return " + strings.Join(resultVars, ", ") + ", nil"

			body += callLine + `
	if err != nil {
		` + errorLine + `
	}` + conversionCode + `
	` + successLine
			return body
		} else {
			var resultVars []string
			for _, arg := range op.RequiredOutputs {
				if arg.IsOutputN {
					continue
				}
				resultVars = append(resultVars, arg.GoName)
			}

			errorLine := generateImageMethodErrorLine(op.RequiredOutputs, false)

			var body string

			if len(op.OptionalInputs) > 0 {
				optionsCallArgs := buildImageOptionsCallArgs(callArgs, op.OptionalInputs, nil, imageOptionArgImageField)

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

			callLine := fmt.Sprintf("%s, err := %s(%s)",
				strings.Join(resultVars, ", "),
				goFuncName,
				strings.Join(callArgs, ", "))

			successLine := "return " + strings.Join(resultVars, ", ") + ", nil"

			body += callLine + `
	if err != nil {
		` + errorLine + `
	}
	` + successLine
			return body
		}
	} else {
		var body string

		supportedOptionalOutputs := getSupportedOptionalOutputs(op)
		if len(op.OptionalInputs) > 0 || len(supportedOptionalOutputs) > 0 {
			optionsCallArgs := buildImageOptionsCallArgs(callArgs, op.OptionalInputs, supportedOptionalOutputs, imageOptionArgSafePointer)

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
	return result.String()
}

// detectMethodArguments analyzes an operation's arguments to determine which should be included in the method signature
func detectMethodArguments(op introspection.Operation) []introspection.Argument {
	var methodArgs []introspection.Argument
	var firstImageFound bool
	var hasBufParam bool
	for _, arg := range op.Arguments {
		if arg.IsOutput {
			continue
		}
		if arg.IsInputN {
			continue
		}
		if arg.IsBuffer {
			hasBufParam = true
			continue
		} else if arg.Name == "len" && hasBufParam {
			continue
		}
		if arg.IsImage && !arg.IsArray && !firstImageFound {
			firstImageFound = true
			continue
		}
		if arg.IsOutput && arg.IsImage {
			continue
		}
		methodArgs = append(methodArgs, arg)
	}

	return methodArgs
}

// generateImageMethodParams formats parameters for image methods using improved detection
func generateImageMethodParams(op introspection.Operation) string {
	methodArgs := detectMethodArguments(op)
	var params []string
	for _, arg := range methodArgs {
		if arg.IsInputN {
			continue
		}
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
	supportedOptionalOutputs := getSupportedOptionalOutputs(op)
	if len(op.OptionalInputs) > 0 || len(supportedOptionalOutputs) > 0 {
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
			if arg.IsOutputN {
				continue
			}
			if arg.Name == "vector" || arg.Name == "out_array" {
				types = append(types, "[]float64")
			} else if arg.GoType == "*C.VipsImage" {
				types = append(types, "*Image")
			} else if arg.GoType == "[]*C.VipsImage" {
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

	body = "Startup(nil)\n\t"

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

	supportedOptionalOutputs := getSupportedOptionalOutputs(op)
	if len(op.OptionalInputs) > 0 || len(supportedOptionalOutputs) > 0 {
		optionsCallArgs := buildImageOptionsCallArgs(callArgs, op.OptionalInputs, supportedOptionalOutputs, imageOptionArgImageField)

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
