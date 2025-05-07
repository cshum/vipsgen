package girintrospection

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/cshum/vipsgen"
)

// Operation represents a libvips operation
type Operation struct {
	Name           string     `json:"name"`
	GoName         string     `json:"goName"`
	CIdentifier    string     `json:"cIdentifier"`
	Description    string     `json:"description"`
	Arguments      []Argument `json:"arguments"`
	RequiredInputs []Argument `json:"requiredInputs"`
	OptionalInputs []Argument `json:"optionalInputs"`
	Outputs        []Argument `json:"outputs"`
	HasImageOutput bool       `json:"hasImageOutput"`
	Category       string     `json:"category"`
	HasImageInput  bool       `json:"hasImageInput"`
}

// Argument represents an argument to a libvips operation
type Argument struct {
	Name        string `json:"name"`
	GoName      string `json:"goName"`
	Direction   string `json:"direction"`
	Type        string `json:"type"`
	GoType      string `json:"goType"`
	CType       string `json:"cType"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	IsInput     bool   `json:"isInput"`
	IsOutput    bool   `json:"isOutput"`
	Flags       int    `json:"flags"`
	IsEnum      bool   `json:"isEnum"`
	EnumType    string `json:"enumType,omitempty"`
}

// DebugElement is used to capture all XML without enforcing a structure
type DebugElement struct {
	XMLName  xml.Name
	Attrs    []xml.Attr      `xml:",any,attr"`
	Content  string          `xml:",chardata"`
	Children []*DebugElement `xml:",any"`
}

// GirParser is responsible for parsing GIR files and extracting operations
type GirParser struct {
	operations []Operation
}

// New creates a new GirParser
func New() *GirParser {
	return &GirParser{
		operations: []Operation{},
	}
}

// Parse parses a GIR file from a reader
func (p *GirParser) Parse(r io.Reader) error {
	// Parse the GIR file with debug structure
	var root DebugElement
	if err := xml.NewDecoder(r).Decode(&root); err != nil {
		return fmt.Errorf("failed to parse GIR file: %v", err)
	}

	// Find the Image class
	imageClass := findImageClass(&root)
	if imageClass == nil {
		return fmt.Errorf("Image class not found in GIR file")
	}

	// Extract operations from the Image class methods
	p.operations = extractOperationsFromImageClass(imageClass)
	return nil
}

// GetOperations returns all operations parsed from the GIR file
func (p *GirParser) GetOperations() []Operation {
	return p.operations
}

// FindOperation finds an operation by name
func (p *GirParser) FindOperation(name string) (Operation, bool) {
	for _, op := range p.operations {
		if op.Name == name {
			return op, true
		}
	}
	return Operation{}, false
}

// ConvertToVipsgenOperations converts the GIR operations to vipsgen.Operation format
func (p *GirParser) ConvertToVipsgenOperations() []vipsgen.Operation {
	var ops []vipsgen.Operation

	for _, girOp := range p.operations {
		// Convert arguments
		var args []vipsgen.Argument
		var requiredInputs []vipsgen.Argument
		var optionalInputs []vipsgen.Argument
		var outputs []vipsgen.Argument

		for _, girArg := range girOp.Arguments {
			arg := vipsgen.Argument{
				Name:        girArg.Name,
				GoName:      girArg.GoName,
				Type:        girArg.Type,
				GoType:      girArg.GoType,
				CType:       girArg.CType,
				Description: girArg.Description,
				Required:    girArg.Required,
				IsInput:     girArg.IsInput,
				IsOutput:    girArg.IsOutput,
				Flags:       girArg.Flags,
				IsEnum:      girArg.IsEnum,
				EnumType:    girArg.EnumType,
			}

			args = append(args, arg)

			if girArg.IsInput {
				if girArg.Required {
					requiredInputs = append(requiredInputs, arg)
				} else {
					optionalInputs = append(optionalInputs, arg)
				}
			} else if girArg.IsOutput {
				outputs = append(outputs, arg)
			}
		}

		// Create vipsgen.Operation
		op := vipsgen.Operation{
			Name:           girOp.Name,
			GoName:         girOp.GoName,
			Description:    girOp.Description,
			Arguments:      args,
			RequiredInputs: requiredInputs,
			OptionalInputs: optionalInputs,
			Outputs:        outputs,
			HasImageOutput: girOp.HasImageOutput,
			Category:       girOp.Category,
			HasImageInput:  girOp.HasImageInput,
		}

		ops = append(ops, op)
		b, _ := json.Marshal(op)
		fmt.Println(string(b))
	}

	return ops
}

// Find the Image class in the GIR file
func findImageClass(root *DebugElement) *DebugElement {
	// Find the namespace element
	var namespace *DebugElement
	for _, child := range root.Children {
		if child.XMLName.Local == "namespace" {
			namespace = child
			break
		}
	}

	if namespace == nil {
		return nil
	}

	// Find the Image class
	for _, child := range namespace.Children {
		if child.XMLName.Local == "class" {
			for _, attr := range child.Attrs {
				if attr.Name.Local == "name" && attr.Value == "Image" {
					return child
				}
			}
		}
	}

	return nil
}

// Extract operations from Image class methods
func extractOperationsFromImageClass(imageClass *DebugElement) []Operation {
	var operations []Operation

	// Get methods in the Image class
	for _, child := range imageClass.Children {
		if child.XMLName.Local == "method" {
			// Get method name
			methodName := ""
			for _, attr := range child.Attrs {
				if attr.Name.Local == "name" {
					methodName = attr.Value
					break
				}
			}

			// Skip if no name
			if methodName == "" {
				continue
			}

			// Skip methods that are not vips operations
			if methodName == "get_typeof" ||
				methodName == "get_width" ||
				methodName == "get_height" ||
				methodName == "get_bands" ||
				methodName == "get_format" ||
				methodName == "get_interpretation" ||
				methodName == "hasalpha" {
				continue
			}

			// Create C identifier from method name (vips_methodname)
			cIdentifier := "vips_" + methodName

			// Create an operation
			op := Operation{
				Name:        methodName,
				GoName:      formatGoFunctionName(methodName),
				CIdentifier: cIdentifier,
				Category:    determineCategory(methodName),
			}

			// Extract documentation
			for _, elem := range child.Children {
				if elem.XMLName.Local == "doc" {
					op.Description = processDocContent(elem)
					break
				}
			}

			// Extract parameters
			extractParameters(child, &op)

			// Skip operations with no parameters (these are likely utility functions)
			if len(op.Arguments) > 0 {
				// Process the operation (categorize arguments)
				processOperation(&op)

				operations = append(operations, op)
			}
		}
	}

	return operations
}

// Extract parameters from a method element
func extractParameters(methodElement *DebugElement, op *Operation) {
	// Find the parameters element
	var paramsElem *DebugElement
	for _, elem := range methodElement.Children {
		if elem.XMLName.Local == "parameters" {
			paramsElem = elem
			break
		}
	}

	if paramsElem == nil {
		return
	}

	// Add instance parameter (input image)
	for _, elem := range paramsElem.Children {
		if elem.XMLName.Local == "instance-parameter" {
			// Create an argument for the instance parameter (always an input image)
			arg := Argument{
				Name:      "in",
				GoName:    "in",
				Type:      "VipsImage",
				GoType:    "*C.VipsImage",
				CType:     "VipsImage*",
				Direction: "in",
				IsInput:   true,
				IsOutput:  false,
				Required:  true,
				Flags:     19, // VIPS_ARGUMENT_REQUIRED | VIPS_ARGUMENT_INPUT
			}

			// Extract description
			for _, child := range elem.Children {
				if child.XMLName.Local == "doc" {
					arg.Description = processDocContent(child)
					break
				}
			}

			op.Arguments = append(op.Arguments, arg)
			break
		}
	}

	// Add output image parameter if not present in parameters list
	// Most vips operations return a new image
	hasOutputImage := false

	// Add additional parameters
	for _, elem := range paramsElem.Children {
		if elem.XMLName.Local == "parameter" {
			// Skip varargs
			var isVarargs bool
			for _, child := range elem.Children {
				if child.XMLName.Local == "varargs" {
					isVarargs = true
					break
				}
			}
			if isVarargs {
				continue
			}

			// Get parameter name and direction
			var paramName string
			var paramDirection string
			for _, attr := range elem.Attrs {
				if attr.Name.Local == "name" {
					paramName = attr.Value
				} else if attr.Name.Local == "direction" {
					paramDirection = attr.Value
				}
			}

			// Skip if no name
			if paramName == "" {
				continue
			}

			// Check if this is an output image
			if paramName == "out" && paramDirection == "out" {
				hasOutputImage = true
			}

			// Create an argument
			arg := Argument{
				Name:      paramName,
				GoName:    formatGoIdentifier(paramName),
				Direction: paramDirection,
				Required:  true,
			}

			// Get type information
			var typeElement *DebugElement
			for _, child := range elem.Children {
				if child.XMLName.Local == "type" {
					typeElement = child
					break
				}
			}

			if typeElement != nil {
				// Get type name and C type
				var typeName string
				var cType string
				for _, attr := range typeElement.Attrs {
					if attr.Name.Local == "name" {
						typeName = attr.Value
					} else if attr.Name.Local == "c:type" {
						cType = attr.Value
					}
				}

				arg.Type = typeName
				arg.CType = cType

				// Determine Go type
				arg.GoType = mapGirTypeToGo(typeName, cType)

				// Check if it's an enum
				if strings.HasPrefix(typeName, "Vips") &&
					(strings.Contains(typeName, "Enum") || strings.Contains(cType, "enum")) {
					arg.IsEnum = true
					arg.EnumType = typeName
				}
			}

			// Set input/output flags
			arg.IsInput = paramDirection != "out"
			arg.IsOutput = paramDirection == "out"

			// Set flags
			if arg.IsOutput {
				arg.Flags = 35 // VIPS_ARGUMENT_REQUIRED | VIPS_ARGUMENT_OUTPUT
			} else {
				arg.Flags = 19 // VIPS_ARGUMENT_REQUIRED | VIPS_ARGUMENT_INPUT
			}

			// Get description
			for _, child := range elem.Children {
				if child.XMLName.Local == "doc" {
					arg.Description = processDocContent(child)
					break
				}
			}

			// Adjust C type for output parameters
			if arg.IsOutput {
				adjustOutputArgType(&arg)
			}

			op.Arguments = append(op.Arguments, arg)
		}
	}

	// Add output image parameter if not present and this is likely an operation that returns an image
	if !hasOutputImage && shouldHaveOutputImage(op.Name) {
		outArg := Argument{
			Name:        "out",
			GoName:      "out",
			Type:        "VipsImage",
			GoType:      "*C.VipsImage",
			CType:       "VipsImage**",
			Direction:   "out",
			Description: "Output image",
			IsInput:     false,
			IsOutput:    true,
			Required:    true,
			Flags:       35, // VIPS_ARGUMENT_REQUIRED | VIPS_ARGUMENT_OUTPUT
		}

		op.Arguments = append(op.Arguments, outArg)
	}
}

// Determine if an operation should have an output image based on its name
func shouldHaveOutputImage(name string) bool {
	// Operations that typically return an image
	returnImageOps := []string{
		"add", "subtract", "multiply", "divide", "linear",
		"rotate", "affine", "flip", "crop", "extract", "embed",
		"resize", "reduce", "shrink", "thumbnail", "gaussblur",
		"sharpen", "colourspace", "bandjoin", "bandmean", "flatten",
		"cast", "copy", "morph", "hist_",
	}

	for _, prefix := range returnImageOps {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	// Operations that typically don't return an image
	nonImageOps := []string{
		"write", "save", "get_", "set_", "hasalpha", "invalidate", "minimise",
	}

	for _, prefix := range nonImageOps {
		if strings.HasPrefix(name, prefix) {
			return false
		}
	}

	// Default to true for most operations
	return true
}

// Adjust the C type for output parameters
func adjustOutputArgType(arg *Argument) {
	if arg.CType == "VipsImage*" {
		arg.CType = "VipsImage**"
	} else if strings.HasSuffix(arg.CType, "*") && !strings.HasSuffix(arg.CType, "**") {
		arg.CType = arg.CType + "*"
	} else if arg.CType == "int" || arg.CType == "gint" {
		arg.CType = "int*"
	} else if arg.CType == "double" || arg.CType == "gdouble" {
		arg.CType = "double*"
	} else if arg.CType == "gboolean" {
		arg.CType = "int*" // booleans represented as int in C
	}
}

// Process an operation to finalize its properties
func processOperation(op *Operation) {
	// Categorize arguments
	for _, arg := range op.Arguments {
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
}

// Process doc content
func processDocContent(elem *DebugElement) string {
	return ""
	//if elem == nil {
	//	return ""
	//}
	//
	//// Remove leading/trailing whitespace and normalize newlines
	//text := strings.TrimSpace(elem.Content)
	//text = strings.ReplaceAll(text, "\n", " ")
	//
	//// Remove multiple spaces
	//for strings.Contains(text, "  ") {
	//	text = strings.ReplaceAll(text, "  ", " ")
	//}
	//
	//return text
}

// Map GIR type to Go type
func mapGirTypeToGo(girType, cType string) string {
	switch girType {
	case "gint", "int":
		return "int"
	case "gdouble", "double":
		return "float64"
	case "gboolean":
		return "bool"
	case "utf8", "gchararray":
		return "string"
	case "Image":
		return "*C.VipsImage"
	default:
		// Handle enums and other special cases
		if strings.HasPrefix(girType, "Vips") {
			return getGoEnumName(girType)
		}
		// For unknown types, use the default mapping based on C type
		switch {
		case strings.Contains(cType, "double") && strings.Contains(cType, "*"):
			return "[]float64"
		case strings.Contains(cType, "int") && strings.Contains(cType, "*"):
			return "[]int"
		case strings.Contains(cType, "VipsArrayDouble"):
			return "[]float64"
		case strings.Contains(cType, "VipsArrayInt"):
			return "[]int"
		case strings.Contains(cType, "VipsBlob"):
			return "[]byte"
		default:
			return "interface{}"
		}
	}
}

// Get the Go enum name from a C enum type name
func getGoEnumName(cName string) string {
	// Strip "Vips" prefix if present
	if strings.HasPrefix(cName, "Vips") {
		cName = cName[4:]
	}

	// Also strip "Foreign" prefix if present
	if strings.HasPrefix(cName, "Foreign") {
		cName = cName[7:]
	}

	return cName
}

// Format a Go function name
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

// Format a Go identifier
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
		strings.HasPrefix(name, "linear") || strings.HasPrefix(name, "math") ||
		strings.HasPrefix(name, "abs") || strings.HasPrefix(name, "sign") ||
		strings.HasPrefix(name, "round") || strings.HasPrefix(name, "floor") ||
		strings.HasPrefix(name, "ceil") || strings.HasPrefix(name, "max") ||
		strings.HasPrefix(name, "min") || strings.HasPrefix(name, "avg") {
		return "arithmetic"
	}

	if strings.HasPrefix(name, "conv") || strings.HasPrefix(name, "sharpen") ||
		strings.HasPrefix(name, "gaussblur") || strings.HasPrefix(name, "sobel") ||
		strings.HasPrefix(name, "canny") {
		return "convolution"
	}

	if strings.HasPrefix(name, "resize") || strings.HasPrefix(name, "shrink") ||
		strings.HasPrefix(name, "reduce") || strings.HasPrefix(name, "thumbnail") ||
		strings.HasPrefix(name, "affine") || strings.HasPrefix(name, "similarity") {
		return "resample"
	}

	if strings.HasPrefix(name, "colourspace") || strings.HasPrefix(name, "icc") ||
		strings.HasPrefix(name, "Lab2XYZ") || strings.HasPrefix(name, "XYZ2Lab") ||
		strings.HasPrefix(name, "Lab2LCh") || strings.HasPrefix(name, "LCh2Lab") ||
		strings.HasPrefix(name, "sRGB2HSV") || strings.HasPrefix(name, "HSV2sRGB") {
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
		strings.HasPrefix(name, "bandjoin") || strings.HasPrefix(name, "bandmean") {
		return "conversion"
	}

	if strings.HasPrefix(name, "hist_") || strings.HasPrefix(name, "stdif") ||
		strings.HasPrefix(name, "percent") || strings.HasPrefix(name, "profile") {
		return "histogram"
	}

	if strings.HasPrefix(name, "morph") || strings.HasPrefix(name, "rank") ||
		strings.HasPrefix(name, "erode") || strings.HasPrefix(name, "dilate") {
		return "morphology"
	}

	if strings.HasPrefix(name, "draw_") || strings.HasPrefix(name, "text") {
		return "draw"
	}

	return "operation" // Default category
}
