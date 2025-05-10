package introspection

import (
	"fmt"
	"github.com/cshum/vipsgen"
	"github.com/cshum/vipsgen/girparser"
	"io"
	"log"
	"regexp"
	"strings"
)

var vipsPattern = regexp.MustCompile(`^vips_.*`)

// VipsFunctionInfo holds information needed to generate a wrapper function
type VipsFunctionInfo struct {
	Name           string
	CIdentifier    string
	ReturnType     string
	HasOutParam    bool
	OutParamIndex  int
	HasVarArgs     bool
	Params         []VipsParamInfo
	RequiredParams []VipsParamInfo // Non-optional params
	OptionalParams []VipsParamInfo // Optional params that can be passed as named args
}

// VipsParamInfo represents a parameter for a vips function
type VipsParamInfo struct {
	Name       string
	CType      string
	IsOutput   bool
	IsOptional bool
	IsArray    bool
	ArrayType  string
	IsVarArgs  bool
}

// DebugInfo stores debug information during parsing
type DebugInfo struct {
	ProcessedFunctions         int
	FoundFunctionNames         []string
	MissingCIdentifierIncluded int
}

// VipsGIRParser adapts girparser functionality for vipsgen
type VipsGIRParser struct {
	// Original GIR data
	gir *girparser.GIR
	// Parsed function info
	functionInfo []VipsFunctionInfo
	// Debug info from parsing
	debugInfo *DebugInfo
}

// NewVipsGIRParser creates a new parser for vipsgen integration
func NewVipsGIRParser() *VipsGIRParser {
	return &VipsGIRParser{}
}

// Parse parses a GIR file from a reader
func (p *VipsGIRParser) Parse(r io.Reader) error {
	// Parse the GIR file
	gir, err := girparser.ParseGIR(r)
	if err != nil {
		return fmt.Errorf("failed to parse GIR file: %v", err)
	}

	p.gir = gir
	p.debugInfo = &DebugInfo{}

	// Process functions from the GIR
	functionInfo, debug := p.extractVipsFunctions(gir)
	p.functionInfo = functionInfo
	p.debugInfo = debug

	return nil
}

// extractVipsFunctions extracts vips functions from the GIR data
func (p *VipsGIRParser) extractVipsFunctions(gir *girparser.GIR) ([]VipsFunctionInfo, *DebugInfo) {
	var functions []VipsFunctionInfo
	debugInfo := &DebugInfo{}

	// Process top-level functions
	log.Printf("Processing %d top-level functions", len(gir.Namespace.Functions))
	for _, fn := range gir.Namespace.Functions {
		if function := p.processVipsFunction(fn, debugInfo); function != nil {
			functions = append(functions, *function)
			debugInfo.FoundFunctionNames = append(debugInfo.FoundFunctionNames, fn.Name)
		}
	}

	// Process class methods/functions
	for _, class := range gir.Namespace.Classes {
		log.Printf("Processing class %s with %d methods and %d functions",
			class.Name, len(class.Methods), len(class.Functions))

		for _, fn := range class.Methods {
			if function := p.processVipsFunction(fn, debugInfo); function != nil {
				functions = append(functions, *function)
				debugInfo.FoundFunctionNames = append(debugInfo.FoundFunctionNames, fn.Name)
			}
		}
		for _, fn := range class.Functions {
			if function := p.processVipsFunction(fn, debugInfo); function != nil {
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
			if function := p.processVipsFunction(fn, debugInfo); function != nil {
				functions = append(functions, *function)
				debugInfo.FoundFunctionNames = append(debugInfo.FoundFunctionNames, fn.Name)
			}
		}
		for _, fn := range iface.Functions {
			if function := p.processVipsFunction(fn, debugInfo); function != nil {
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
			if function := p.processVipsFunction(fn, debugInfo); function != nil {
				functions = append(functions, *function)
				debugInfo.FoundFunctionNames = append(debugInfo.FoundFunctionNames, fn.Name)
			}
		}
		for _, fn := range record.Functions {
			if function := p.processVipsFunction(fn, debugInfo); function != nil {
				functions = append(functions, *function)
				debugInfo.FoundFunctionNames = append(debugInfo.FoundFunctionNames, fn.Name)
			}
		}
	}

	log.Printf("Successfully processed %d functions", len(functions))
	return functions, debugInfo
}

// processVipsFunction processes a Function to VipsFunctionInfo without skipping non-introspectable functions
func (p *VipsGIRParser) processVipsFunction(fn girparser.Function, debugInfo *DebugInfo) *VipsFunctionInfo {
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

	debugInfo.ProcessedFunctions++

	info := &VipsFunctionInfo{
		Name:        fn.Name,
		CIdentifier: fn.CIdentifier,
		ReturnType:  formatReturnType(fn.ReturnValue),
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
		if param.Direction == "out" || strings.HasSuffix(param.Type.CType, "**") {
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

// processVipsParam converts a Parameter to VipsParamInfo
func processVipsParam(param girparser.Parameter) VipsParamInfo {
	paramInfo := VipsParamInfo{
		Name:       param.Name,
		CType:      param.Type.CType,
		IsOptional: param.Optional,
		IsVarArgs:  param.VarArgs,
		IsOutput:   param.Direction == "out",
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
func (p *VipsGIRParser) ConvertToVipsgenOperations() []vipsgen.Operation {
	var operations []vipsgen.Operation

	for _, fn := range p.functionInfo {
		// Skip functions that don't match our vips_ pattern
		if !strings.HasPrefix(fn.CIdentifier, "vips_") {
			continue
		}

		// Create a new operation
		op := vipsgen.Operation{
			Name:        fn.Name,
			GoName:      formatGoFunctionName(fn.Name),
			Description: fmt.Sprintf("Wrapper for %s", fn.CIdentifier),
			Category:    vipsgen.DetermineCategory(fn.Name),
		}

		// Process arguments
		for _, param := range fn.Params {
			// Skip varargs placeholder
			if param.Name == "..." {
				continue
			}

			arg := vipsgen.Argument{
				Name:        param.Name,
				GoName:      formatGoIdentifier(param.Name),
				Type:        extractTypeFromCType(param.CType),
				GoType:      mapCTypeToGoType(param.CType, param.IsOutput),
				CType:       param.CType,
				Description: fmt.Sprintf("%s parameter", param.Name),
				Required:    !param.IsOptional,
				IsInput:     !param.IsOutput,
				IsOutput:    param.IsOutput,
				IsEnum:      isEnumType(param.CType),
				Flags:       determineFlags(param.IsOutput, !param.IsOptional),
			}

			// Determine enum type if applicable
			if arg.IsEnum {
				arg.EnumType = extractEnumType(param.CType)
			}

			op.Arguments = append(op.Arguments, arg)

			// Categorize arguments
			if arg.IsInput {
				if arg.Required {
					op.RequiredInputs = append(op.RequiredInputs, arg)
					if arg.Type == "VipsImage" {
						op.HasImageInput = true
					}
				} else {
					op.OptionalInputs = append(op.OptionalInputs, arg)
				}
			} else if arg.IsOutput {
				op.Outputs = append(op.Outputs, arg)
				if arg.Type == "VipsImage" {
					op.HasImageOutput = true
				}
			}
		}

		operations = append(operations, op)
	}

	return operations
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

func mapCTypeToGoType(cType string, isOutput bool) string {
	baseType := extractTypeFromCType(cType)

	// Handle special cases based on the base type
	switch baseType {
	case "VipsImage":
		return "*C.VipsImage"
	case "gint", "int":
		return "int"
	case "gdouble", "double":
		return "float64"
	case "gfloat", "float":
		return "float32"
	case "gboolean":
		return "bool"
	case "gchararray", "char", "gchar":
		return "string"
	case "void":
		if strings.HasSuffix(cType, "*") {
			// void* is often used for arrays
			return "[]interface{}"
		}
		return "interface{}"
	}

	// Handle enum types
	if isEnumType(cType) {
		return extractEnumType(cType)
	}

	// Handle array types
	if strings.Contains(cType, "[]") || strings.HasSuffix(cType, "*") && !isOutput {
		switch baseType {
		case "gint", "int":
			return "[]int"
		case "gdouble", "double":
			return "[]float64"
		case "gchararray", "char", "gchar":
			return "[]string"
		default:
			return "[]interface{}"
		}
	}

	// Default case for unknown types
	return "interface{}"
}

func isEnumType(cType string) bool {
	enumPrefixes := []string{
		"VipsBlendMode",
		"VipsAccess",
		"VipsExtend",
		"VipsAngle",
		"VipsInterpretation",
		"VipsDirection",
		"VipsOperationMorphology",
		"VipsForeignSubsample",
		"VipsForeignTiffCompression",
		"VipsForeignTiffPredictor",
	}

	baseType := extractTypeFromCType(cType)
	for _, prefix := range enumPrefixes {
		if baseType == prefix {
			return true
		}
	}
	return false
}

func extractEnumType(cType string) string {
	baseType := extractTypeFromCType(cType)

	// Map VipsEnum -> Enum
	if strings.HasPrefix(baseType, "Vips") {
		return strings.TrimPrefix(baseType, "Vips")
	}

	return baseType
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

func formatGoFunctionName(name string) string {
	// Convert operation names to match existing Go function style
	// e.g., "rotate" -> "vipsRotate", "extract_area" -> "vipsExtractArea"
	parts := strings.Split(name, "_")

	// Convert each part to title case
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[0:1]) + part[1:]
		}
	}

	// Join with vips prefix
	return "vips" + strings.Join(parts, "")
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
