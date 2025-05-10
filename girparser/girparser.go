package girparser

import (
	"encoding/xml"
	"fmt"
	"io"
)

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

// ParseGIR parses GIR data from an io.Reader
func ParseGIR(r io.Reader) (*GIR, error) {
	var gir GIR
	decoder := xml.NewDecoder(r)
	err := decoder.Decode(&gir)
	if err != nil {
		return nil, fmt.Errorf("failed to decode GIR XML: %w", err)
	}
	return &gir, nil
}
