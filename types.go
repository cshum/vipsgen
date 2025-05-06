package vipsgen

import "text/template"

// TemplateLoader is an interface for loading and generating files from templates
type TemplateLoader interface {
	// LoadTemplate loads a template by name
	LoadTemplate(name string) (*template.Template, error)

	// ListTemplateFiles returns a list of all template files
	ListTemplateFiles() ([]string, error)

	// GenerateFile generates a file using a template and data
	GenerateFile(templateName, outputFile string, data interface{}) error

	// ProcessStaticFiles processes all static files (copies them without template processing)
	ProcessStaticFiles(outputDir string) error
}

// Operation represents a libvips operation
type Operation struct {
	Name           string
	GoName         string
	Description    string
	Flags          int
	Arguments      []Argument
	RequiredInputs []Argument
	OptionalInputs []Argument
	Outputs        []Argument
	HasImageOutput bool
	Category       string // arithmetic, conversion, etc
	HasImageInput  bool   // Does this operation take a VipsImage as input?
}

// Argument represents an argument to a libvips operation
type Argument struct {
	Name        string
	GoName      string
	Type        string
	GoType      string
	CType       string
	Description string
	Required    bool
	IsInput     bool
	IsOutput    bool
	Flags       int
	IsEnum      bool
	EnumType    string
}

// EnumValue represents a value in a libvips enum
type EnumValue struct {
	Name        string
	Value       int
	Nickname    string
	Description string
	GoName      string // The Go-friendly name
}

// EnumType represents a libvips enum type
type EnumType struct {
	Name        string // Original C name (e.g., VipsInterpretation)
	GoName      string // Go name (e.g., Interpretation)
	Values      []EnumValue
	Description string
}

// EnumTypeInfo holds information about a vips enum type
type EnumTypeInfo struct {
	CName       string // Original C name (e.g. VipsInterpretation)
	GoName      string // Go name (e.g. Interpretation)
	Description string
	Values      []EnumValueInfo
}

// EnumValueInfo holds information about an enum value
type EnumValueInfo struct {
	CName       string // C name
	GoName      string // Go name
	Value       int    // Numeric value
	Description string
}

// ImageTypeInfo represents information about an image type
type ImageTypeInfo struct {
	TypeName string // Short name (e.g., "gif")
	EnumName string // Go enum name (e.g., "ImageTypeGIF")
	MimeType string // MIME type (e.g., "image/gif")
	Order    int    // Position in the enum
}

// OperationConfig holds custom configuration for specific operations
type OperationConfig struct {
	SkipGen        bool   // Don't generate this operation
	CustomWrapper  bool   // Needs custom C wrapper implementation
	OptionsParam   string // Name of the options parameter if any
	NeedsMultiPage bool   // Operation needs multi-page variant
}

// SupportedSaverInfo holds information about supported image savers
type SupportedSaverInfo struct {
	EnumName string
	TypeName string
}

// TemplateData holds all data needed by any template
type TemplateData struct {
	Operations       []Operation
	OperationConfigs map[string]OperationConfig
	EnumTypes        []EnumTypeInfo
	ImageTypes       []ImageTypeInfo
	EnumTypeMap      map[string]bool // For quick lookups

	// Image operations (filtered operations with VipsImage as first arg and output)
	ImageOperations []Operation

	HasJpegSaver      bool
	HasPngSaver       bool
	HasWebpSaver      bool
	HasTiffSaver      bool
	HasHeifSaver      bool
	HasLegacyGifSaver bool
	HasCgifSaver      bool
	HasAvifSaver      bool
	HasJp2kSaver      bool
	SupportedSavers   []SupportedSaverInfo
}
