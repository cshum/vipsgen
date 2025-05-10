package girparser

import (
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
)

var vipsPattern = regexp.MustCompile(`^vips_.*`)

// GIR represents the root element of a GIR file
type GIR struct {
	XMLName   xml.Name  `xml:"repository"`
	Version   string    `xml:"version,attr"`
	Namespace Namespace `xml:"namespace"`
}

// Namespace represents a GIR namespace
type Namespace struct {
	Name          string      `xml:"name,attr"`
	Version       string      `xml:"version,attr"`
	SharedLibrary string      `xml:"shared-library,attr"`
	Functions     []Function  `xml:"function"`
	Classes       []Class     `xml:"class"`
	Interfaces    []Interface `xml:"interface"`
	Records       []Record    `xml:"record"`
}

// Function represents a function/method declaration
type Function struct {
	Name           string      `xml:"name,attr"`
	Introspectable string      `xml:"introspectable,attr"`
	CIdentifier    string      `xml:"c:identifier,attr"`
	ReturnValue    ReturnValue `xml:"return-value"`
	Parameters     []Parameter `xml:"parameters>parameter"`
	InstanceParam  *Parameter  `xml:"parameters>instance-parameter"`
	Doc            string      `xml:"doc"`
}

// Class represents a GObject class
type Class struct {
	Name      string     `xml:"name,attr"`
	Methods   []Function `xml:"method"`
	Functions []Function `xml:"function"`
}

// Interface represents a GObject interface
type Interface struct {
	Name      string     `xml:"name,attr"`
	Methods   []Function `xml:"method"`
	Functions []Function `xml:"function"`
}

// Record represents a C structure
type Record struct {
	Name      string     `xml:"name,attr"`
	Methods   []Function `xml:"method"`
	Functions []Function `xml:"function"`
}

// ReturnValue represents a function return value
type ReturnValue struct {
	Type Type `xml:"type"`
}

// Parameter represents a function parameter
type Parameter struct {
	Name      string `xml:"name,attr"`
	Direction string `xml:"direction,attr"`
	Optional  bool   `xml:"optional,attr"`
	Type      Type   `xml:"type"`
	VarArgs   bool   `xml:"varargs,attr"`
}

// Type represents a data type
type Type struct {
	Name  string `xml:"name,attr"`
	CType string `xml:"c:type,attr"`
}

// DebugInfo stores debug information during parsing
type DebugInfo struct {
	ProcessedFunctions         int
	FoundFunctionNames         []string
	MissingCIdentifierIncluded int
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

// VipsGIRParser adapts girparser functionality for vipsgen
type VipsGIRParser struct {
	// Original GIR data
	gir *GIR
	// Parsed function info
	functionInfo []VipsFunctionInfo
	// Debug info from parsing
	debugInfo *DebugInfo
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
	return &gir, debugInfo, nil
}
