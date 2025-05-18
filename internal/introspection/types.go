package introspection

// Operation represents a libvips operation
type Operation struct {
	Name               string
	GoName             string
	Description        string
	Arguments          []Argument
	RequiredInputs     []Argument
	OptionalInputs     []Argument
	RequiredOutputs    []Argument
	OptionalOutputs    []Argument
	HasImageInput      bool
	HasImageOutput     bool
	HasOneImageOutput  bool
	HasBufferInput     bool
	HasBufferOutput    bool
	HasArrayImageInput bool
	ImageTypeString    string
	Category           string // arithmetic, conversion, etc
}

// OperationConfig holds custom configuration for specific operations
type OperationConfig struct {
	SkipGen        bool   // Don't generate this operation
	CustomWrapper  bool   // Needs custom C wrapper implementation
	OptionsParam   string // Name of the options parameter if any
	NeedsMultiPage bool   // Operation needs multi-page variant
}

// Argument represents an argument to a libvips operation
type Argument struct {
	Name         string
	GoName       string
	Type         string
	GoType       string
	CType        string
	Description  string
	Required     bool
	IsInput      bool
	IsNInput     bool
	IsOutput     bool
	IsImage      bool
	IsBuffer     bool
	IsArray      bool
	Flags        int
	IsEnum       bool
	EnumType     string
	NArrayFrom   string
	DefaultValue interface{}
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
