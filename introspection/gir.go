package introspection

import (
	"fmt"
	"github.com/cshum/vipsgen"
	"github.com/cshum/vipsgen/girparser"
	"io"
	"log"
	"path/filepath"
	"strings"
)

// ParseGir parses a GIR file from a reader
func (v *Introspection) ParseGir(r io.Reader) error {
	// ParseGir the GIR file
	gir, err := girparser.ParseGIR(r)
	if err != nil {
		return fmt.Errorf("failed to parse GIR file: %v", err)
	}

	v.gir = gir
	v.debugInfo = &DebugInfo{}

	// Process functions from the GIR
	functionInfo, debug := v.extractVipsFunctions(gir)
	v.functionInfo = functionInfo
	v.debugInfo = debug

	return nil
}

// extractVipsFunctions extracts vips functions from the GIR data
func (v *Introspection) extractVipsFunctions(gir *girparser.GIR) ([]VipsFunctionInfo, *DebugInfo) {
	var functions []VipsFunctionInfo
	debugInfo := &DebugInfo{}

	// Process top-level functions
	log.Printf("Processing %d top-level functions", len(gir.Namespace.Functions))
	for _, fn := range gir.Namespace.Functions {
		if function := v.processVipsFunction(fn, debugInfo); function != nil {
			functions = append(functions, *function)
			debugInfo.FoundFunctionNames = append(debugInfo.FoundFunctionNames, fn.Name)
		}
	}

	// Process class methods/functions
	for _, class := range gir.Namespace.Classes {
		log.Printf("Processing class %s with %d methods and %d functions",
			class.Name, len(class.Methods), len(class.Functions))

		for _, fn := range class.Methods {
			if function := v.processVipsFunction(fn, debugInfo); function != nil {
				functions = append(functions, *function)
				debugInfo.FoundFunctionNames = append(debugInfo.FoundFunctionNames, fn.Name)
			}
		}
		for _, fn := range class.Functions {
			if function := v.processVipsFunction(fn, debugInfo); function != nil {
				functions = append(functions, *function)
				debugInfo.FoundFunctionNames = append(debugInfo.FoundFunctionNames, fn.Name)
			}
		}
	}

	// Process interface methods/functions
	for _, iface := range gir.Namespace.Interfaces {
		log.Printf("Processing interface %s with %d methods and %d functions",
			iface.Name, len(iface.Methods), len(iface.Functions))

		for _, fn := range iface.Methods {
			if function := v.processVipsFunction(fn, debugInfo); function != nil {
				functions = append(functions, *function)
				debugInfo.FoundFunctionNames = append(debugInfo.FoundFunctionNames, fn.Name)
			}
		}
		for _, fn := range iface.Functions {
			if function := v.processVipsFunction(fn, debugInfo); function != nil {
				functions = append(functions, *function)
				debugInfo.FoundFunctionNames = append(debugInfo.FoundFunctionNames, fn.Name)
			}
		}
	}

	// Process record methods/functions
	for _, record := range gir.Namespace.Records {
		log.Printf("Processing record %s with %d methods and %d functions",
			record.Name, len(record.Methods), len(record.Functions))

		for _, fn := range record.Methods {
			if function := v.processVipsFunction(fn, debugInfo); function != nil {
				functions = append(functions, *function)
				debugInfo.FoundFunctionNames = append(debugInfo.FoundFunctionNames, fn.Name)
			}
		}
		for _, fn := range record.Functions {
			if function := v.processVipsFunction(fn, debugInfo); function != nil {
				functions = append(functions, *function)
				debugInfo.FoundFunctionNames = append(debugInfo.FoundFunctionNames, fn.Name)
			}
		}
	}

	log.Printf("Successfully processed %d functions", len(functions))
	return functions, debugInfo
}

// processVipsFunction processes a Function to VipsFunctionInfo with array awareness
func (v *Introspection) processVipsFunction(fn girparser.Function, debugInfo *DebugInfo) *VipsFunctionInfo {
	// Check for vips-specific functions
	if !vipsPattern.MatchString(fn.CIdentifier) && fn.CIdentifier != "" {
		// Skip non-vips functions
		return nil
	}

	// If the function has no C identifier, try to generate one based on the vips_ prefix and function name
	if fn.CIdentifier == "" {
		fn.CIdentifier = "vips_" + fn.Name
		debugInfo.MissingCIdentifierIncluded++
		log.Printf("Generated C identifier for function %s: %s", fn.Name, fn.CIdentifier)
	}

	v.DiscoverEnumsFromOperation(fn.Name)

	debugInfo.ProcessedFunctions++

	info := &VipsFunctionInfo{
		Name:        fn.Name,
		CIdentifier: fn.CIdentifier,
		ReturnType:  formatReturnType(fn.ReturnValue),
		Category:    extractCategoryFromFilename(fn.SourcePosition.Filename),
		Description: extractDescription(fn.Doc),
		HasVarArgs:  false,
	}

	// Process parameters with array awareness
	if fn.InstanceParam != nil {
		param := processVipsParam(*fn.InstanceParam)

		// Set array flag if this is an array parameter
		if fn.InstanceParam.Array != nil {
			param.IsArray = true
			if fn.InstanceParam.Array.ElementType.CType != "" {
				param.ArrayType = fn.InstanceParam.Array.ElementType.CType
			}
		}

		info.Params = append(info.Params, param)
		if !param.IsOptional {
			info.RequiredParams = append(info.RequiredParams, param)
		}
	}

	for i, param := range fn.Parameters {
		// Check for output parameter
		if param.Direction == "out" {
			info.HasOutParam = true
			info.OutParamIndex = i
		}

		// Check for varargs
		if param.VarArgs {
			info.HasVarArgs = true
			continue
		}

		paramInfo := processVipsParam(param)

		// Set array flag if this is an array parameter
		if param.Array != nil {
			paramInfo.IsArray = true

			// Get array element type if available
			if param.Array.ElementType.CType != "" {
				paramInfo.ArrayType = param.Array.ElementType.CType
			}

			// Use array's C type for the parameter if available
			if param.Array.CType != "" {
				paramInfo.CType = param.Array.CType
			}
		}

		info.Params = append(info.Params, paramInfo)

		if paramInfo.IsOptional {
			info.OptionalParams = append(info.OptionalParams, paramInfo)
		} else {
			info.RequiredParams = append(info.RequiredParams, paramInfo)
		}
	}

	return info
}

func extractCategoryFromFilename(filename string) string {
	// Extract category from paths like "libvips/include/vips/arithmetic.h"
	parts := strings.Split(filename, "/")
	for i, part := range parts {
		if part == "vips" && i+1 < len(parts) {
			// Get the next part after "vips/"
			category := parts[i+1]
			// Remove file extension if present
			return strings.TrimSuffix(category, filepath.Ext(category))
		}
	}
	return "operation" // Default category
}

// processVipsParam converts a Parameter to VipsParamInfo with array detection
func processVipsParam(param girparser.Parameter) VipsParamInfo {
	paramInfo := VipsParamInfo{
		Name:       param.Name,
		IsOptional: param.Optional,
		IsVarArgs:  param.VarArgs,
	}

	// Set C type from parameter or array
	if param.Array != nil && param.Array.CType != "" {
		paramInfo.CType = param.Array.CType
	} else if param.Type.CType != "" {
		paramInfo.CType = param.Type.CType
	}

	// Determine if this is an output parameter
	if param.Direction == "out" {
		paramInfo.IsOutput = true
	} else if param.Name == "out" {
		paramInfo.IsOutput = true
	} else if strings.HasSuffix(paramInfo.CType, "**") && param.Array == nil {
		// Double pointers often indicate output parameters
		// But only if they're not arrays
		paramInfo.IsOutput = true
	}

	return paramInfo
}

// formatReturnType formats a return type
func formatReturnType(ret girparser.ReturnValue) string {
	if ret.Type.CType != "" {
		return ret.Type.CType
	}
	if ret.Type.Name == "none" {
		return "void"
	}
	return ret.Type.Name
}

// ConvertToVipsgenOperations converts girparser functions to vipsgen.Operation format
func (v *Introspection) ConvertToVipsgenOperations() []vipsgen.Operation {
	var operations []vipsgen.Operation

	for _, fn := range v.functionInfo {
		// Skip functions that don't match our vips_ pattern
		if !strings.HasPrefix(fn.CIdentifier, "vips_") {
			continue
		}

		// Create a new operation
		op := vipsgen.Operation{
			Name:        fn.Name,
			GoName:      vipsgen.FormatGoFunctionName(fn.Name),
			Description: fn.Description,
			Category:    fn.Category,
		}

		// Find the original GIR function
		var originalFunc *girparser.Function
		for i := range v.gir.Namespace.Functions {
			if v.gir.Namespace.Functions[i].Name == fn.Name {
				originalFunc = &v.gir.Namespace.Functions[i]
				break
			}
		}

		// Process arguments with full GIR parameter information
		for _, param := range fn.Params {
			// Skip varargs placeholder
			if param.Name == "..." || param.IsVarArgs {
				continue
			}

			// Find original parameter in GIR data
			var originalParam *girparser.Parameter
			if originalFunc != nil {
				for i := range originalFunc.Parameters {
					if originalFunc.Parameters[i].Name == param.Name {
						originalParam = &originalFunc.Parameters[i]
						break
					}
				}
			}

			// Get effective C type based on parameter structure
			cType := getEffectiveCType(param, originalParam)

			// Determine Go type with array awareness
			goType := v.mapCTypeToGoType(cType, param, originalParam)

			// Extract base type
			baseType := extractBaseType(cType)

			// Create argument with original parameter reference
			arg := vipsgen.Argument{
				Name:          param.Name,
				GoName:        vipsgen.FormatGoIdentifier(param.Name),
				Type:          baseType,
				GoType:        goType,
				CType:         cType,
				Description:   fmt.Sprintf("%s parameter", param.Name),
				Required:      !param.IsOptional,
				IsInput:       !param.IsOutput,
				IsOutput:      param.IsOutput,
				IsEnum:        v.isEnumType(baseType),
				Flags:         determineFlags(param.IsOutput, !param.IsOptional),
				OriginalParam: originalParam,
			}

			// Determine enum type if applicable
			if arg.IsEnum {
				arg.EnumType = v.GetGoEnumName(baseType)
			}

			op.Arguments = append(op.Arguments, arg)

			// Categorize arguments
			if arg.IsInput {
				if arg.Required {
					op.RequiredInputs = append(op.RequiredInputs, arg)
				} else {
					op.OptionalInputs = append(op.OptionalInputs, arg)
				}
			} else if arg.IsOutput {
				op.Outputs = append(op.Outputs, arg)
			}
		}

		v.FixConstFunctions(&op)
		v.FixOperationTypes(&op)

		// Update image input/output flags
		v.UpdateImageInputOutputFlags(&op)

		//	b, _ := json.Marshal(op)
		//	fmt.Println(string(b))

		operations = append(operations, op)
	}

	return operations
}

// getEffectiveCType determines the actual C type using original GIR parameter if available
func getEffectiveCType(param VipsParamInfo, originalParam *girparser.Parameter) string {
	// If we have the original parameter with array info, use it
	if originalParam != nil && originalParam.Array != nil && originalParam.Array.CType != "" {
		return originalParam.Array.CType
	}

	// If we have the original parameter with type info, use it
	if originalParam != nil && originalParam.Type.CType != "" {
		return originalParam.Type.CType
	}

	// Fall back to the converted parameter info
	return param.CType
}

// extractBaseType removes pointer symbols and array notation from C type strings
func extractBaseType(cType string) string {
	// Remove 'const' qualifier if present
	baseType := strings.TrimPrefix(cType, "const ")

	// Remove pointer and array markers
	baseType = strings.TrimRight(baseType, "*[]")

	// Remove any spaces
	baseType = strings.TrimSpace(baseType)

	// Handle special cases
	switch baseType {
	case "int":
		return "gint"
	case "double":
		return "gdouble"
	case "float":
		return "gfloat"
	case "char":
		return "gchararray"
	}

	return baseType
}

// Helper functions for type mapping and formatting
func extractTypeFromCType(cType string) string {
	// Strip any pointer/array markers
	baseType := strings.TrimRight(cType, "*[]")

	// Map C types to vipsgen types
	switch baseType {
	case "int", "gint":
		return "gint"
	case "double", "gdouble":
		return "gdouble"
	case "float", "gfloat":
		return "gfloat"
	case "char", "gchar":
		return "gchararray"
	default:
		// For unknown types, use as is
		return baseType
	}
}

// mapCTypeToGoType maps a C type to a Go type with array awareness
func (v *Introspection) mapCTypeToGoType(cType string, param VipsParamInfo, originalParam *girparser.Parameter) string {
	if cType == "const double*" {
		return "[]float64"
	}
	// If we have original parameter with array info, use it
	if originalParam != nil && originalParam.Array != nil {
		// Get element type if available
		elementType := originalParam.Array.ElementType.CType

		if elementType == "VipsImage*" {
			return "[]*C.VipsImage"
		}
		if strings.HasPrefix(elementType, "int") || strings.HasPrefix(elementType, "gint") {
			return "[]int"
		}
		if strings.HasPrefix(elementType, "double") || strings.HasPrefix(elementType, "gdouble") {
			return "[]float64"
		}
		if strings.HasPrefix(elementType, "float") || strings.HasPrefix(elementType, "gfloat") {
			return "[]float32"
		}
		if strings.HasPrefix(elementType, "char") || strings.HasPrefix(elementType, "gchar") {
			return "[]string"
		}

		// For general arrays, use the C type pattern
		if strings.HasPrefix(cType, "VipsImage**") {
			return "[]*C.VipsImage"
		}
		if strings.HasPrefix(cType, "int*") || strings.HasPrefix(cType, "gint*") {
			return "[]int"
		}
		if strings.HasPrefix(cType, "double*") || strings.HasPrefix(cType, "gdouble*") {
			return "[]float64"
		}
	}

	// Handle scalar types
	baseType := extractBaseType(cType)

	switch baseType {
	case "VipsImage":
		return "*C.VipsImage"
	case "gboolean":
		return "bool"
	case "gint", "int":
		return "int"
	case "gdouble", "double", "gfloat", "float":
		return "float64"
	case "gchararray", "char", "gchar", "utf8":
		return "string"
	case "VipsArrayInt":
		return "[]int"
	case "VipsArrayDouble":
		return "[]float64"
	case "VipsArrayImage":
		return "[]*C.VipsImage"
	case "VipsBlob", "Blob":
		return "*C.VipsBlob"
	case "VipsInterpolate":
		return "*C.VipsInterpolate"
	case "VipsSource", "Source":
		return "*C.VipsSource"
	case "VipsTarget", "Target":
		return "*C.VipsTarget"
	}

	// Check if it's an enum type
	if v.isEnumType(baseType) {
		return v.GetGoEnumName(baseType)
	}

	// Handle pointer types
	if strings.HasSuffix(cType, "*") {
		if strings.Contains(cType, "char*") {
			return "string"
		}
		if strings.Contains(cType, "int*") || strings.Contains(cType, "gint*") {
			return "[]int"
		}
		if strings.Contains(cType, "double*") || strings.Contains(cType, "gdouble*") {
			return "[]float64"
		}
		if strings.Contains(cType, "float*") || strings.Contains(cType, "gfloat*") {
			return "[]float32"
		}
		if strings.HasSuffix(cType, "**") {
			return "unsafe.Pointer" // Double pointers other than VipsImage** need special handling
		}
		return "unsafe.Pointer" // Generic pointer
	}

	// Default case
	return "interface{}"
}

func (v *Introspection) isEnumType(cType string) bool {
	return v.discoveredEnumTypes[cType] != ""
}

// determineFlags calculates the flags for an argument
func determineFlags(isOutput bool, isRequired bool) int {
	if isOutput && isRequired {
		return 35 // VIPS_ARGUMENT_REQUIRED | VIPS_ARGUMENT_OUTPUT
	} else if isOutput && !isRequired {
		return 33 // VIPS_ARGUMENT_OPTIONAL | VIPS_ARGUMENT_OUTPUT
	} else if !isOutput && isRequired {
		return 19 // VIPS_ARGUMENT_REQUIRED | VIPS_ARGUMENT_INPUT
	} else {
		return 17 // VIPS_ARGUMENT_OPTIONAL | VIPS_ARGUMENT_INPUT
	}
}

// Extract a concise description from the documentation
func extractDescription(doc string) string {
	if doc == "" {
		return ""
	}

	// Split into lines and find the first non-empty line
	lines := strings.Split(doc, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "Optional arguments:") &&
			!strings.HasPrefix(line, "*") && !strings.HasPrefix(line, "See also:") {
			// Return the first meaningful line as the description
			// Some descriptions can be quite long, so consider truncating
			if len(line) > 100 {
				return line[:97] + "..."
			}
			return line
		}
	}

	return ""
}
