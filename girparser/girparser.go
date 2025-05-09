// Package girparser provides functionality to parse GObject Introspection (GIR)
// files for libvips and extract data to generate C header functions
package girparser

import (
	"encoding/xml"
	"fmt"
	"github.com/cshum/vipsgen"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// GIR represents the root element of a GIR file
type GIR struct {
	XMLName   xml.Name  `xml:"repository"`
	Version   string    `xml:"version,attr"`
	Namespace Namespace `xml:"namespace"`
	Includes  []Include `xml:"include"`
}

// Include represents an included GIR dependency
type Include struct {
	Name    string `xml:"name,attr"`
	Version string `xml:"version,attr"`
}

// Namespace represents a GIR namespace
type Namespace struct {
	Name            string      `xml:"name,attr"`
	Version         string      `xml:"version,attr"`
	SharedLibrary   string      `xml:"shared-library,attr"`
	CIdentifierPref string      `xml:"c:identifier-prefixes,attr"`
	CSymbolPref     string      `xml:"c:symbol-prefixes,attr"`
	Aliases         []Alias     `xml:"alias"`
	Functions       []Function  `xml:"function"`
	Classes         []Class     `xml:"class"`
	Interfaces      []Interface `xml:"interface"`
	Records         []Record    `xml:"record"`
	Enums           []Enum      `xml:"enumeration"`
	Bitfields       []Enum      `xml:"bitfield"`
	Constants       []Constant  `xml:"constant"`
	Callbacks       []Callback  `xml:"callback"`
}

// Alias represents a typedef alias
type Alias struct {
	Name     string `xml:"name,attr"`
	CType    string `xml:"c:type,attr"`
	TypeName Type   `xml:"type"`
}

// Function represents a function/method declaration
type Function struct {
	Name           string `xml:"name,attr"`
	Introspectable string `xml:"introspectable,attr"`
	CIdentifier    string `xml:"c:identifier,attr"`
	SourcePosition struct {
		Filename string `xml:"filename,attr"`
		Line     int    `xml:"line,attr"`
	} `xml:"source-position"`
	ReturnValue   ReturnValue `xml:"return-value"`
	Parameters    []Parameter `xml:"parameters>parameter"`
	InstanceParam *Parameter  `xml:"parameters>instance-parameter"`
	Doc           string      `xml:"doc"`
	Throws        bool        `xml:"throws,attr"`
}

// Class represents a GObject class
type Class struct {
	Name           string     `xml:"name,attr"`
	CType          string     `xml:"c:type,attr"`
	Parent         string     `xml:"parent,attr"`
	Methods        []Function `xml:"method"`
	Constructors   []Function `xml:"constructor"`
	Functions      []Function `xml:"function"`
	VirtualMethods []Function `xml:"virtual-method"`
}

// Interface represents a GObject interface
type Interface struct {
	Name           string     `xml:"name,attr"`
	CType          string     `xml:"c:type,attr"`
	Methods        []Function `xml:"method"`
	Functions      []Function `xml:"function"`
	VirtualMethods []Function `xml:"virtual-method"`
}

// Record represents a C structure
type Record struct {
	Name         string     `xml:"name,attr"`
	CType        string     `xml:"c:type,attr"`
	Methods      []Function `xml:"method"`
	Constructors []Function `xml:"constructor"`
	Functions    []Function `xml:"function"`
}

// ReturnValue represents a function return value
type ReturnValue struct {
	TransferOwn string `xml:"transfer-ownership,attr"`
	Nullable    bool   `xml:"nullable,attr"`
	Type        Type   `xml:"type"`
	Array       *Array `xml:"array"`
}

// Parameter represents a function parameter
type Parameter struct {
	Name        string `xml:"name,attr"`
	Direction   string `xml:"direction,attr"`
	TransferOwn string `xml:"transfer-ownership,attr"`
	Nullable    bool   `xml:"nullable,attr"`
	Optional    bool   `xml:"optional,attr"`
	AllowNone   bool   `xml:"allow-none,attr"`
	Type        Type   `xml:"type"`
	Array       *Array `xml:"array"`
	VarArgs     bool   `xml:"varargs,attr"`
}

// Type represents a data type
type Type struct {
	Name  string `xml:"name,attr"`
	CType string `xml:"c:type,attr"`
}

// Array represents an array type
type Array struct {
	CType      string `xml:"c:type,attr"`
	Length     int    `xml:"length,attr"`
	ZeroTermin bool   `xml:"zero-terminated,attr"`
	FixedSize  int    `xml:"fixed-size,attr"`
	Type       Type   `xml:"type"`
}

// Enum represents an enumeration
type Enum struct {
	Name    string       `xml:"name,attr"`
	CType   string       `xml:"c:type,attr"`
	Members []EnumMember `xml:"member"`
}

// EnumMember represents an enum value
type EnumMember struct {
	Name        string `xml:"name,attr"`
	Value       string `xml:"value,attr"`
	CIdentifier string `xml:"c:identifier,attr"`
}

// Constant represents a C constant or #define
type Constant struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
	CType string `xml:"c:type,attr"`
	Type  Type   `xml:"type"`
}

// Callback represents a function pointer type
type Callback struct {
	Name        string      `xml:"name,attr"`
	CType       string      `xml:"c:type,attr"`
	ReturnValue ReturnValue `xml:"return-value"`
	Parameters  []Parameter `xml:"parameters>parameter"`
}

// DebugInfo stores debug information during parsing
type DebugInfo struct {
	FunctionsFound             int
	ClassMethodsFound          int
	InterfaceMethodsFound      int
	RecordMethodsFound         int
	IntrospectableFunctions    int
	NonIntrospectableFunctions int
	FunctionWithoutCIdentifier int
	ProcessedFunctions         int
	FoundFunctionNames         []string
	NonIntrospectableIncluded  int
	MissingCIdentifierIncluded int
}

// ParseGIRFile parses a GIR file at the given path
func ParseGIRFile(filePath string) (*GIR, *DebugInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open GIR file: %w", err)
	}
	defer file.Close()

	return ParseGIR(file)
}

// ParseGIR parses GIR data from an io.Reader
func ParseGIR(r io.Reader) (*GIR, *DebugInfo, error) {
	var gir GIR
	decoder := xml.NewDecoder(r)
	err := decoder.Decode(&gir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode GIR XML: %w", err)
	}

	debugInfo := &DebugInfo{}

	// Count the various types of functions
	debugInfo.FunctionsFound = len(gir.Namespace.Functions)

	for _, class := range gir.Namespace.Classes {
		debugInfo.ClassMethodsFound += len(class.Methods)
		debugInfo.ClassMethodsFound += len(class.Functions)
	}

	for _, iface := range gir.Namespace.Interfaces {
		debugInfo.InterfaceMethodsFound += len(iface.Methods)
		debugInfo.InterfaceMethodsFound += len(iface.Functions)
	}

	for _, record := range gir.Namespace.Records {
		debugInfo.RecordMethodsFound += len(record.Methods)
		debugInfo.RecordMethodsFound += len(record.Functions)
	}

	// Count introspectable vs non-introspectable functions
	for _, fn := range gir.Namespace.Functions {
		if fn.Introspectable == "0" {
			debugInfo.NonIntrospectableFunctions++
		} else {
			debugInfo.IntrospectableFunctions++
		}

		if fn.CIdentifier == "" {
			debugInfo.FunctionWithoutCIdentifier++
		}
	}

	return &gir, debugInfo, nil
}

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

// IncludeFunc is a filtering function to determine if a particular function should be included
type IncludeFunc func(string) bool

// GetVipsFunctions collects all the Vips operation functions from the GIR data
func GetVipsFunctions(gir *GIR, includeFilter IncludeFunc) ([]VipsFunctionInfo, *DebugInfo) {
	var functions []VipsFunctionInfo
	debugInfo := &DebugInfo{}

	// Process top-level functions
	log.Printf("Processing %d top-level functions", len(gir.Namespace.Functions))
	for _, fn := range gir.Namespace.Functions {
		if function := processVipsFunction(fn, includeFilter, debugInfo); function != nil {
			functions = append(functions, *function)
			debugInfo.FoundFunctionNames = append(debugInfo.FoundFunctionNames, fn.Name)
		}
	}

	// Process class methods/functions
	for _, class := range gir.Namespace.Classes {
		log.Printf("Processing class %s with %d methods and %d functions",
			class.Name, len(class.Methods), len(class.Functions))

		for _, fn := range class.Methods {
			if function := processVipsFunction(fn, includeFilter, debugInfo); function != nil {
				functions = append(functions, *function)
				debugInfo.FoundFunctionNames = append(debugInfo.FoundFunctionNames, fn.Name)
			}
		}
		for _, fn := range class.Functions {
			if function := processVipsFunction(fn, includeFilter, debugInfo); function != nil {
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
			if function := processVipsFunction(fn, includeFilter, debugInfo); function != nil {
				functions = append(functions, *function)
				debugInfo.FoundFunctionNames = append(debugInfo.FoundFunctionNames, fn.Name)
			}
		}
		for _, fn := range iface.Functions {
			if function := processVipsFunction(fn, includeFilter, debugInfo); function != nil {
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
			if function := processVipsFunction(fn, includeFilter, debugInfo); function != nil {
				functions = append(functions, *function)
				debugInfo.FoundFunctionNames = append(debugInfo.FoundFunctionNames, fn.Name)
			}
		}
		for _, fn := range record.Functions {
			if function := processVipsFunction(fn, includeFilter, debugInfo); function != nil {
				functions = append(functions, *function)
				debugInfo.FoundFunctionNames = append(debugInfo.FoundFunctionNames, fn.Name)
			}
		}
	}

	log.Printf("Successfully processed %d functions", len(functions))
	return functions, debugInfo
}

// processVipsFunction converts a Function to VipsFunctionInfo
func processVipsFunction(fn Function, includeFilter IncludeFunc, debugInfo *DebugInfo) *VipsFunctionInfo {
	// Special case: if this function is in our include filter, we'll process it
	// even if it's not introspectable or missing a C identifier
	specialCase := includeFilter != nil && includeFilter(fn.Name)

	// Skip introspectable="0" functions unless it's a special case
	if fn.Introspectable == "0" && !specialCase {
		log.Printf("Skipping non-introspectable function: %s", fn.Name)
		return nil
	} else if fn.Introspectable == "0" && specialCase {
		// Count functions that we included despite being non-introspectable
		debugInfo.NonIntrospectableIncluded++
		log.Printf("Including non-introspectable function due to filter match: %s", fn.Name)
	}

	// Skip functions with no C identifier (macros, etc.) unless it's a special case
	if fn.CIdentifier == "" {
		// For functions with no C identifier but matching our filter,
		// we'll try to generate a C identifier based on the vips_ prefix and function name
		if specialCase {
			fn.CIdentifier = "vips_" + fn.Name
			debugInfo.MissingCIdentifierIncluded++
			log.Printf("Generated C identifier for function %s: %s", fn.Name, fn.CIdentifier)
		} else {
			log.Printf("Skipping function without C identifier: %s", fn.Name)
			return nil
		}
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
func processVipsParam(param Parameter) VipsParamInfo {
	paramInfo := VipsParamInfo{
		Name:       param.Name,
		CType:      param.Type.CType,
		IsOptional: param.Optional,
		IsVarArgs:  param.VarArgs,
		IsOutput:   param.Direction == "out",
	}

	// Handle array parameters
	if param.Array != nil {
		paramInfo.IsArray = true
		paramInfo.ArrayType = param.Array.Type.CType
		paramInfo.CType = param.Array.CType
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
func formatReturnType(ret ReturnValue) string {
	if ret.Type.CType != "" {
		return ret.Type.CType
	}
	if ret.Type.Name == "none" {
		return "void"
	}
	return ret.Type.Name
}

// VipsCodeGenerator generates C code for vips functions
type VipsCodeGenerator struct {
	HeaderTemplate   *template.Template
	FunctionTemplate *template.Template
}

// NewVipsCodeGenerator creates a new code generator with templates
func NewVipsCodeGenerator() (*VipsCodeGenerator, error) {
	// Fixed header template that includes parameter types
	headerTmpl, err := template.New("header").Parse(`/* Generated wrapper functions for libvips
 * Generated from {{.Namespace.Name}}-{{.Namespace.Version}} GIR
 */

#ifndef VIPS_WRAPPER_H
#define VIPS_WRAPPER_H

#include <glib.h>
#include <vips/vips.h>

#ifdef __cplusplus
extern "C" {
#endif

{{range .Functions}}
/**
 * {{.Name}}:
 {{range .RequiredParams}}* @{{.Name}}: {{if .IsOutput}}Output {{end}}parameter
 {{end}}{{range .OptionalParams}}* @{{.Name}}: Optional parameter
 {{end}}* Returns: VIPS error code (0 for success)
 */
int {{.Name}}({{range $i, $p := .Params}}{{if $i}}, {{end}}{{$p.CType}} {{$p.Name}}{{end}}{{if .HasVarArgs}}{{if .Params}}, {{end}}...{{end}});
{{end}}

#ifdef __cplusplus
}
#endif

#endif /* VIPS_WRAPPER_H */
`)
	if err != nil {
		return nil, err
	}

	// Fixed function implementation template that includes parameter types
	funcTmpl, err := template.New("function").Parse(`{{range .Functions}}
/**
 * {{.Name}}:
 {{range .RequiredParams}}* @{{.Name}}: {{if .IsOutput}}Output {{end}}parameter
 {{end}}{{range .OptionalParams}}* @{{.Name}}: Optional parameter
 {{end}}* Returns: VIPS error code (0 for success)
 */
int {{.Name}}({{range $i, $p := .Params}}{{if $i}}, {{end}}{{$p.CType}} {{$p.Name}}{{end}}{{if .HasVarArgs}}{{if .Params}}, {{end}}...{{end}}) {
  return {{.CIdentifier}}({{range $i, $p := .RequiredParams}}{{if $i}}, {{end}}{{$p.Name}}{{end}}{{if and .RequiredParams .OptionalParams}}, {{end}}{{range $i, $p := .OptionalParams}}{{if $i}}, {{end}}"{{$p.Name}}", {{$p.Name}}{{end}}{{if or .HasVarArgs (not .OptionalParams)}}, NULL{{end}});
}
{{end}}
`)
	if err != nil {
		return nil, err
	}

	return &VipsCodeGenerator{
		HeaderTemplate:   headerTmpl,
		FunctionTemplate: funcTmpl,
	}, nil
}

// GenerateHeader generates a C header file with vips wrapper functions
func (g *VipsCodeGenerator) GenerateHeader(gir *GIR, functions []VipsFunctionInfo) (string, error) {
	var buf strings.Builder

	log.Printf("Generating header for %d functions", len(functions))

	err := g.HeaderTemplate.Execute(&buf, struct {
		Namespace *Namespace
		Functions []VipsFunctionInfo
	}{
		Namespace: &gir.Namespace,
		Functions: functions,
	})

	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// GenerateImplementation generates C implementation code for the wrapper functions
func (g *VipsCodeGenerator) GenerateImplementation(gir *GIR, functions []VipsFunctionInfo) (string, error) {
	var buf strings.Builder

	log.Printf("Generating implementation for %d functions", len(functions))

	buf.WriteString("/* Generated wrapper functions for libvips\n")
	buf.WriteString(fmt.Sprintf(" * Generated from %s-%s GIR\n", gir.Namespace.Name, gir.Namespace.Version))
	buf.WriteString(" */\n\n")
	buf.WriteString("#include <glib.h>\n")
	buf.WriteString("#include <vips/vips.h>\n\n")

	err := g.FunctionTemplate.Execute(&buf, struct {
		Namespace *Namespace
		Functions []VipsFunctionInfo
	}{
		Namespace: &gir.Namespace,
		Functions: functions,
	})

	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// FilterVipsFunctions filters functions to those matching certain criteria
func FilterVipsFunctions(functions []VipsFunctionInfo, criteria func(VipsFunctionInfo) bool) []VipsFunctionInfo {
	var filtered []VipsFunctionInfo

	log.Printf("Filtering %d functions", len(functions))

	for _, fn := range functions {
		if criteria(fn) {
			filtered = append(filtered, fn)
			log.Printf("Function matched filter: %s", fn.Name)
		}
	}

	log.Printf("After filtering: %d functions", len(filtered))

	return filtered
}

// DumpCIdentifiers dumps all CIdentifiers from the GIR file to help with debugging
func DumpCIdentifiers(gir *GIR) []string {
	var identifiers []string

	// Collect from top-level functions
	for _, fn := range gir.Namespace.Functions {
		if fn.CIdentifier != "" {
			identifiers = append(identifiers, fmt.Sprintf("%s -> %s", fn.Name, fn.CIdentifier))
		}
	}

	// Collect from class methods/functions
	for _, class := range gir.Namespace.Classes {
		for _, fn := range class.Methods {
			if fn.CIdentifier != "" {
				identifiers = append(identifiers, fmt.Sprintf("%s.%s -> %s", class.Name, fn.Name, fn.CIdentifier))
			}
		}
		for _, fn := range class.Functions {
			if fn.CIdentifier != "" {
				identifiers = append(identifiers, fmt.Sprintf("%s::%s -> %s", class.Name, fn.Name, fn.CIdentifier))
			}
		}
	}

	// Collect from interface methods/functions
	for _, iface := range gir.Namespace.Interfaces {
		for _, fn := range iface.Methods {
			if fn.CIdentifier != "" {
				identifiers = append(identifiers, fmt.Sprintf("%s.%s -> %s", iface.Name, fn.Name, fn.CIdentifier))
			}
		}
		for _, fn := range iface.Functions {
			if fn.CIdentifier != "" {
				identifiers = append(identifiers, fmt.Sprintf("%s::%s -> %s", iface.Name, fn.Name, fn.CIdentifier))
			}
		}
	}

	// Collect from record methods/functions
	for _, record := range gir.Namespace.Records {
		for _, fn := range record.Methods {
			if fn.CIdentifier != "" {
				identifiers = append(identifiers, fmt.Sprintf("%s.%s -> %s", record.Name, fn.Name, fn.CIdentifier))
			}
		}
		for _, fn := range record.Functions {
			if fn.CIdentifier != "" {
				identifiers = append(identifiers, fmt.Sprintf("%s::%s -> %s", record.Name, fn.Name, fn.CIdentifier))
			}
		}
	}

	return identifiers
}

// VipsGIRParser adapts girparser functionality for vipsgen
type VipsGIRParser struct {
	// Original GIR data
	gir *GIR
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
	gir, debugInfo, err := ParseGIR(r)
	if err != nil {
		return fmt.Errorf("failed to parse GIR file: %v", err)
	}

	p.gir = gir
	p.debugInfo = debugInfo

	// Set up vips-specific filter function
	includeFilter := func(name string) bool {
		// Custom filter to include operations like arrayjoin, composite2, etc.
		specialOps := map[string]bool{
			"arrayjoin":  true,
			"composite2": true,
			"linear":     true,
			"linear1":    true,
			"replicate":  true,
			"find_trim":  true,
			"affine":     true,
		}
		return specialOps[name]
	}

	// Get Vips functions using our filter
	p.functionInfo, _ = GetVipsFunctions(gir, includeFilter)
	return nil
}

// ConvertToVipsgenOperations converts girparser functions to vipsgen.Operation format
func (p *VipsGIRParser) ConvertToVipsgenOperations() []vipsgen.Operation {
	var operations []vipsgen.Operation

	for _, fn := range p.functionInfo {
		// Create a new operation
		op := vipsgen.Operation{
			Name:        fn.Name,
			GoName:      formatGoFunctionName(fn.Name),
			Description: fmt.Sprintf("Wrapper for %s", fn.CIdentifier),
			Category:    determineCategory(fn.Name),
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

// Extract the base type from a C type (e.g., "VipsImage*" -> "VipsImage")
func extractTypeFromCType(cType string) string {
	// Strip any pointer/array markers
	baseType := strings.TrimRight(cType, "*[]")

	// Map C types to vipsgen types
	switch baseType {
	case "VipsImage":
		return "VipsImage"
	case "int", "gint":
		return "gint"
	case "double", "gdouble":
		return "gdouble"
	case "float", "gfloat":
		return "gfloat"
	case "gboolean":
		return "gboolean"
	case "char", "gchar":
		return "gchararray"
	// Add enum types
	case "VipsBlendMode":
		return "VipsBlendMode"
	case "VipsAccess":
		return "VipsAccess"
	case "VipsExtend":
		return "VipsExtend"
	case "VipsAngle":
		return "VipsAngle"
	case "VipsInterpretation":
		return "VipsInterpretation"
	default:
		// For unknown types, use as is
		return baseType
	}
}

// Map C types to Go types for vipsgen
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

// Check if a type is likely an enum type
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

// Extract enum type name from C type
func extractEnumType(cType string) string {
	baseType := extractTypeFromCType(cType)

	// Map VipsEnum -> Enum
	if strings.HasPrefix(baseType, "Vips") {
		return strings.TrimPrefix(baseType, "Vips")
	}

	return baseType
}

// Determine flags for argument based on input/output and required/optional
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

// Format a Go function name from operation name
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

// Format a Go identifier from parameter name
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

// Determine the category of an operation based on its name
func determineCategory(name string) string {
	// Use prefixes to determine categories
	if strings.HasPrefix(name, "add") || strings.HasPrefix(name, "subtract") ||
		strings.HasPrefix(name, "multiply") || strings.HasPrefix(name, "divide") ||
		strings.HasPrefix(name, "linear") || strings.HasPrefix(name, "math") {
		return "arithmetic"
	}

	if strings.HasPrefix(name, "conv") || strings.HasPrefix(name, "sharpen") ||
		strings.HasPrefix(name, "gaussblur") {
		return "convolution"
	}

	if strings.HasPrefix(name, "resize") || strings.HasPrefix(name, "shrink") ||
		strings.HasPrefix(name, "reduce") || strings.HasPrefix(name, "thumbnail") ||
		strings.HasPrefix(name, "affine") {
		return "resample"
	}

	if strings.HasPrefix(name, "colourspace") || strings.HasPrefix(name, "icc") {
		return "colour"
	}

	if strings.HasSuffix(name, "load") {
		return "foreign_load"
	}

	if strings.HasSuffix(name, "save") || strings.HasSuffix(name, "save_buffer") {
		return "foreign_save"
	}

	if strings.HasPrefix(name, "flip") || strings.HasPrefix(name, "rot") ||
		strings.HasPrefix(name, "extract") || strings.HasPrefix(name, "embed") ||
		strings.HasPrefix(name, "crop") || strings.HasPrefix(name, "join") ||
		strings.HasPrefix(name, "bandjoin") || strings.HasPrefix(name, "arrayjoin") ||
		strings.HasPrefix(name, "replicate") {
		return "conversion"
	}

	if strings.HasPrefix(name, "find_trim") {
		return "conversion"
	}

	if strings.HasPrefix(name, "composite") {
		return "conversion"
	}

	return "operation" // Default category
}

// FindGIRFile searches for a GIR file in standard locations
func FindGIRFile(name string) (string, error) {
	// Implementation from original girparser
	locations := []string{
		"/usr/share/gir-1.0",
		"/usr/local/share/gir-1.0",
		"/opt/homebrew/share/gir-1.0", // macOS Homebrew location
	}

	// Check if XDG_DATA_DIRS is set
	if xdgDataDirs := os.Getenv("XDG_DATA_DIRS"); xdgDataDirs != "" {
		for _, dir := range strings.Split(xdgDataDirs, ":") {
			girDir := filepath.Join(dir, "gir-1.0")
			locations = append(locations, girDir)
		}
	}

	for _, loc := range locations {
		path := filepath.Join(loc, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("could not find GIR file %s in standard locations", name)
}

// GenerateVipsWrapperHeader generates a C header file with vips wrapper function declarations
func GenerateVipsWrapperHeader(functions []VipsFunctionInfo) (string, error) {
	// Define a template for the header
	headerTmpl, err := template.New("header").Parse(`/* Generated wrapper functions for libvips */

#ifndef VIPS_WRAPPER_H
#define VIPS_WRAPPER_H

#include <glib.h>
#include <vips/vips.h>

#ifdef __cplusplus
extern "C" {
#endif

{{range .}}
/**
 * {{.Name}}: Wrapper for {{.CIdentifier}}
 {{range .RequiredParams}}* @{{.Name}}: {{if .IsOutput}}Output {{end}}parameter
 {{end}}{{range .OptionalParams}}* @{{.Name}}: Optional parameter
 {{end}}* Returns: VIPS error code (0 for success)
 */
int {{.Name}}_wrapper({{range $i, $p := .Params}}{{if $i}}, {{end}}{{$p.CType}} {{$p.Name}}{{end}});
{{end}}

#ifdef __cplusplus
}
#endif

#endif /* VIPS_WRAPPER_H */
`)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	err = headerTmpl.Execute(&buf, functions)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// GenerateVipsWrapperImplementation generates C implementation for wrapper functions
func GenerateVipsWrapperImplementation(functions []VipsFunctionInfo) (string, error) {
	// Define a template for the implementation
	implTmpl, err := template.New("implementation").Parse(`/* Generated wrapper functions for libvips */

#include <glib.h>
#include <vips/vips.h>
#include "vips.h"

{{range .}}
/**
 * {{.Name}}_wrapper: Wrapper for {{.CIdentifier}}
 {{range .RequiredParams}}* @{{.Name}}: {{if .IsOutput}}Output {{end}}parameter
 {{end}}{{range .OptionalParams}}* @{{.Name}}: Optional parameter
 {{end}}* Returns: VIPS error code (0 for success)
 */
int {{.Name}}_wrapper({{range $i, $p := .Params}}{{if $i}}, {{end}}{{$p.CType}} {{$p.Name}}{{end}}) {
  {{if .HasOutParam}}return {{.CIdentifier}}({{range $i, $p := .RequiredParams}}{{if $i}}, {{end}}{{if eq $p.Name "out"}}*out{{else}}{{$p.Name}}{{end}}{{end}}{{if and .RequiredParams .OptionalParams}}, {{end}}{{range $i, $p := .OptionalParams}}{{if $i}}, {{end}}"{{$p.Name}}", {{$p.Name}}{{end}}, NULL);
  {{else}}return {{.CIdentifier}}({{range $i, $p := .RequiredParams}}{{if $i}}, {{end}}{{$p.Name}}{{end}}{{if and .RequiredParams .OptionalParams}}, {{end}}{{range $i, $p := .OptionalParams}}{{if $i}}, {{end}}"{{$p.Name}}", {{$p.Name}}{{end}}, NULL);
  {{end}}
}
{{end}}
`)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	err = implTmpl.Execute(&buf, functions)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
