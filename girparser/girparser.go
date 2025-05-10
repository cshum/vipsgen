package girparser

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// ParseGIR parses GIR data from an io.Reader
func ParseGIR(r io.Reader) (*GIR, error) {
	// Read the entire file
	xmlData, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read XML data: %w", err)
	}

	// Simple pre-processing to handle namespaces
	// Replace namespaced attributes with regular attributes
	xmlString := string(xmlData)
	xmlString = strings.ReplaceAll(xmlString, "c:type", "ctype")
	xmlString = strings.ReplaceAll(xmlString, "c:identifier", "cidentifier")

	// Parse the modified XML
	var gir GIR
	decoder := xml.NewDecoder(strings.NewReader(xmlString))
	err = decoder.Decode(&gir)
	if err != nil {
		return nil, fmt.Errorf("failed to decode GIR XML: %w", err)
	}

	// Debug: Write the parsed GIR to a JSON file
	jsonData, err := json.MarshalIndent(gir, "", "  ")
	if err != nil {
		log.Printf("Warning: failed to marshal GIR to JSON: %v", err)
	} else {
		err = os.WriteFile("debug_gir.json", jsonData, 0644)
		if err != nil {
			log.Printf("Warning: failed to write debug_gir.json: %v", err)
		} else {
			log.Println("Wrote parsed GIR structure to debug_gir.json")
		}
	}

	return &gir, nil
}

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
	Name           string         `xml:"name,attr"`
	Introspectable string         `xml:"introspectable,attr"`
	CIdentifier    string         `xml:"cidentifier,attr"` // Changed from c:identifier
	ReturnValue    ReturnValue    `xml:"return-value"`
	Parameters     []Parameter    `xml:"parameters>parameter"`
	InstanceParam  *Parameter     `xml:"parameters>instance-parameter"`
	Doc            string         `xml:"doc"`
	SourcePosition SourcePosition `xml:"source-position"`
	DocFilename    string         `xml:"-"` // Extract from Doc attributes
}

type SourcePosition struct {
	Filename string `xml:"filename,attr"`
	Line     int    `xml:"line,attr"`
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
	Name            string     `xml:"name,attr"`
	Direction       string     `xml:"direction,attr"`
	CallerAllocates string     `xml:"caller-allocates,attr"`
	Optional        bool       `xml:"optional,attr"`
	Type            Type       `xml:"type"`
	Array           *ArrayType `xml:"array"`
	VarArgs         bool       `xml:"varargs,attr"`
}

// ArrayType represents an array type in GIR
type ArrayType struct {
	Length         string `xml:"length,attr"`
	ZeroTerminated string `xml:"zero-terminated,attr"`
	CType          string `xml:"ctype,attr"` // Changed from c:type
	ElementType    Type   `xml:"type"`
}

// Type represents a data type
type Type struct {
	Name  string `xml:"name,attr"`
	CType string `xml:"ctype,attr"` // Changed from c:type
}
