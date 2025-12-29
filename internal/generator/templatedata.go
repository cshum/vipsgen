package generator

import (
	"github.com/cshum/vipsgen/internal/introspection"
)

// TemplateData holds all data needed by any template
type TemplateData struct {
	VipsVersion string
	Operations  []introspection.Operation
	EnumTypes   []introspection.EnumTypeInfo
	ImageTypes  []introspection.ImageTypeInfo
	IncludeTest bool
}

// NewTemplateData creates a new TemplateData structure with all needed information
func NewTemplateData(
	vipsVersion string,
	operations []introspection.Operation,
	enumTypes []introspection.EnumTypeInfo,
	imageTypes []introspection.ImageTypeInfo,
	includeTest bool,
) *TemplateData {
	return &TemplateData{
		VipsVersion: vipsVersion,
		Operations:  operations,
		EnumTypes:   enumTypes,
		ImageTypes:  imageTypes,
		IncludeTest: includeTest,
	}
}
