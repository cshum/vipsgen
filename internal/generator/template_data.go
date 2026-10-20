package generator

import (
	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// enumValueNames maps an enum type's Go name and numeric value to the Go constant
// name, so a generated comment can say "default IntentLast" instead of "default 7".
// It is filled in once per run by NewTemplateData.
var enumValueNames = map[string]map[int]string{}

// TemplateData holds all data needed by any template
type TemplateData struct {
	VipsVersion string
	Operations  []introspection.Operation
	EnumTypes   []introspection.EnumTypeInfo
	ImageTypes  []introspection.ImageTypeInfo
	IncludeTest bool
	Coverage    []CoverageCase
}

// NewTemplateData creates a new TemplateData structure with all needed information
func NewTemplateData(
	vipsVersion string,
	operations []introspection.Operation,
	enumTypes []introspection.EnumTypeInfo,
	imageTypes []introspection.ImageTypeInfo,
	includeTest bool,
) *TemplateData {
	enumValueNames = map[string]map[int]string{}
	for _, et := range enumTypes {
		values := make(map[int]string, len(et.Values))
		for _, v := range et.Values {
			values[v.Value] = v.GoName
		}
		enumValueNames[et.GoName] = values
	}

	return &TemplateData{
		VipsVersion: vipsVersion,
		Operations:  operations,
		EnumTypes:   enumTypes,
		ImageTypes:  imageTypes,
		IncludeTest: includeTest,
		Coverage:    BuildOperationCoverage(operations),
	}
}
