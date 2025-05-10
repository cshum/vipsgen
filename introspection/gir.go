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

// processVipsFunction processes a Function to VipsFunctionInfo without skipping non-introspectable functions
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

	// Process parameters
	if fn.InstanceParam != nil {
		param := processVipsParam(*fn.InstanceParam)
		info.Params = append(info.Params, param)
		if !param.IsOptional {
			info.RequiredParams = append(info.RequiredParams, param)
		}
	}

	for i, param := range fn.Parameters {
		// Check for output parameter (typically a VipsImage**)
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
		info.Params = append(info.Params, paramInfo)

		if paramInfo.IsOptional {
			info.OptionalParams = append(info.OptionalParams, paramInfo)
		} else {
			info.RequiredParams = append(info.RequiredParams, paramInfo)
		}
	}

	log.Printf("Processed function: %s (C identifier: %s) with %d parameters (%d required, %d optional)",
		info.Name, info.CIdentifier, len(info.Params), len(info.RequiredParams), len(info.OptionalParams))

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

// processVipsParam converts a Parameter to VipsParamInfo
func processVipsParam(param girparser.Parameter) VipsParamInfo {
	paramInfo := VipsParamInfo{
		Name:       param.Name,
		CType:      param.Type.CType,
		IsOptional: param.Optional,
		IsVarArgs:  param.VarArgs,
	}

	// 1. Check the direction attribute first
	if param.Direction == "out" {
		paramInfo.IsOutput = true
	}

	// 2. Then check the parameter name (as a fallback)
	if !paramInfo.IsOutput && param.Name == "out" {
		paramInfo.IsOutput = true
	}

	// 3. Check the C type (as another fallback)
	// Double pointers often indicate output parameters
	if !paramInfo.IsOutput && strings.HasSuffix(paramInfo.CType, "**") {
		paramInfo.IsOutput = true
	}

	// 4. Special case: buffer length parameters are often outputs
	if strings.HasSuffix(param.Name, "_len") || param.Name == "len" {
		if paramInfo.CType == "gsize" || paramInfo.CType == "size_t" {
			// Make sure it's a pointer for output parameters
			if paramInfo.IsOutput && !strings.HasSuffix(paramInfo.CType, "*") {
				paramInfo.CType = paramInfo.CType + "*"
			}
		}
	}

	// Fix for empty CType - provide a default
	if paramInfo.CType == "" {
		if param.Type.Name == "none" {
			paramInfo.CType = "void"
		} else if param.Type.Name != "" {
			// Try to guess the C type based on the name
			switch param.Type.Name {
			case "gint":
				paramInfo.CType = "int"
			case "gdouble":
				paramInfo.CType = "double"
			case "gfloat":
				paramInfo.CType = "float"
			case "gboolean":
				paramInfo.CType = "gboolean"
			case "gchar":
				paramInfo.CType = "char"
			case "guchar":
				paramInfo.CType = "unsigned char"
			case "gpointer":
				paramInfo.CType = "void*"
			case "gchar*":
				paramInfo.CType = "char*"
			case "Image":
				// For Image type, check if it's an output parameter
				if paramInfo.IsOutput {
					paramInfo.CType = "VipsImage**"
				} else {
					paramInfo.CType = "VipsImage*"
				}
			default:
				// Default fallback - use the name as type
				paramInfo.CType = param.Type.Name + "*"
			}
		} else {
			// Last resort: use void* for unknown types
			paramInfo.CType = "void*"
			log.Printf("Warning: Unknown type for parameter %s, using void*", param.Name)
		}
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

		// Process arguments
		for _, param := range fn.Params {
			// Skip varargs placeholder
			if param.Name == "..." {
				continue
			}

			// Fix the C type for special cases
			cType := v.FixCType(param.CType)

			// Add pointer to output parameters
			if param.IsOutput && !strings.HasSuffix(cType, "*") {
				cType = cType + "*"
			}

			arg := vipsgen.Argument{
				Name:        param.Name,
				GoName:      formatGoIdentifier(param.Name),
				Type:        extractBaseType(cType),
				GoType:      v.mapCTypeToGoType(cType, param.IsOutput),
				CType:       cType,
				Description: fmt.Sprintf("%s parameter", param.Name),
				Required:    !param.IsOptional,
				IsInput:     !param.IsOutput,
				IsOutput:    param.IsOutput,
				IsEnum:      v.isEnumType(cType),
				Flags:       determineFlags(param.IsOutput, !param.IsOptional),
			}

			// Check for "in" parameter with *C.VipsImage type
			if arg.Name == "in" && arg.Type == "VipsImage" && !arg.IsOutput {
				op.HasImageInput = true
			}

			// Check for "out" parameter with VipsImage type
			if arg.Name == "out" && arg.Type == "VipsImage" && arg.IsOutput {
				op.HasImageOutput = true
			}

			// Determine enum type if applicable
			if arg.IsEnum {
				arg.EnumType = v.GetGoEnumName(param.CType)
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

		// 1. Fix specific known functions with special parameter handling
		v.FixParameterTypes(&op)

		// 2. Detect and fix array parameters (more general approach)
		v.FixArrayParameters(&op)

		// 3. Fix void* parameters to appropriate Go types
		v.FixVoidParameters(&op)

		// 4. Check for image input/output after all fixes have been applied
		v.UpdateImageInputOutputFlags(&op)

		operations = append(operations, op)
	}

	return operations
}

// extractBaseType removes pointer symbols and array notation from C type strings
// to get the base type name. For example:
// - "VipsImage*" → "VipsImage"
// - "int[]" → "int"
// - "const char*" → "char"
func extractBaseType(cType string) string {
	// Handle special cases first
	if cType == "utf8*" {
		return "utf8"
	}
	if cType == "Source*" {
		return "Source"
	}
	if cType == "Target*" {
		return "Target"
	}
	if cType == "Blob*" {
		return "Blob"
	}
	if cType == "const char*" {
		return "char"
	}

	// Remove 'const' qualifier if present
	baseType := strings.TrimPrefix(cType, "const ")

	// Remove pointer and array markers
	baseType = strings.TrimRight(baseType, "*[]")

	// Remove any spaces
	baseType = strings.TrimSpace(baseType)

	// Map C types to vipsgen types
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

func (v *Introspection) mapCTypeToGoType(cType string, isOutput bool) string {
	// Extract the base type without pointers
	baseType := extractBaseType(cType)

	// Special handling for output parameters of scalar types
	if isOutput {
		// If it's a scalar output parameter (indicated by a pointer),
		// determine the appropriate Go type
		baseType := strings.TrimSuffix(cType, "*")
		switch baseType {
		case "double", "gdouble":
			return "float64"
		case "int", "gint":
			return "int"
		case "gboolean":
			return "bool"
		case "size_t", "gsize":
			return "int"
		}
	}

	// Map VIPS types to Go types
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
		return "string" // Important: This maps const char* and utf8* to string
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
	default:
		// Check if it's an enum type
		if v.isEnumType(baseType) {
			return v.GetGoEnumName(baseType)
		}
	}

	// If cType contains "char*" or is utf8*, treat it as a string
	if strings.Contains(cType, "char*") || cType == "utf8*" || cType == "const char*" {
		return "string"
	}

	// Default case for unknown types
	return "interface{}"
}

func (v *Introspection) isEnumType(cType string) bool {
	return v.discoveredEnumTypes[cType] != ""
}

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

func formatGoIdentifier(name string) string {
	// Format snake_case to camelCase
	parts := strings.Split(name, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][0:1]) + parts[i][1:]
		}
	}

	result := strings.Join(parts, "")

	// Handle Go keywords
	switch result {
	case "type", "func", "map", "range", "select", "case", "default":
		return result + "_"
	}

	return result
}
