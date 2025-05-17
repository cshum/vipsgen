package generator

import (
	"github.com/cshum/vipsgen/internal/introspection"
	"text/template"
)

// TemplateLoader is an interface for loading and generating files from templates
type TemplateLoader interface {
	// LoadTemplate loads a template by name
	LoadTemplate(name string) (*template.Template, error)

	// ListFiles returns a list of all template files
	ListFiles() ([]string, error)

	// GenerateFile generates a file using a template and data
	GenerateFile(templateName, outputFile string, data interface{}) error
}

// TemplateData holds all data needed by any template
type TemplateData struct {
	Operations       []introspection.Operation
	OperationConfigs map[string]introspection.OperationConfig
	EnumTypes        []introspection.EnumTypeInfo
	ImageTypes       []introspection.ImageTypeInfo
	EnumTypeMap      map[string]bool // For quick lookups

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

// SupportedSaverInfo holds information about supported image savers
type SupportedSaverInfo struct {
	EnumName string
	TypeName string
}
