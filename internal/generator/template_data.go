package generator

import (
	"sort"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// enumValueNames maps an enum type's Go name and numeric value to the Go constant
// name, so a generated comment can say "default IntentLast" instead of "default 7".
// It is filled in once per run by NewTemplateData.
var enumValueNames = map[string]map[int]string{}

// TemplateData holds all data needed by any template
type TemplateData struct {
	VipsVersion    string
	Operations     []introspection.Operation
	EnumTypes      []introspection.EnumTypeInfo
	ImageTypes     []introspection.ImageTypeInfo
	IncludeTest    bool
	Coverage       []CoverageCase
	OptionDefaults []string
	// LoaderImageTypes is ImageTypes ordered longest name first, for the loader
	// prefix match: "matrixload" starts with both "matrix" and "mat", and only the
	// longer name is the truth.
	LoaderImageTypes []introspection.ImageTypeInfo
}

// longestNameFirst orders image types so that a loader name matches the most
// specific type before any type whose name is a prefix of it.
func longestNameFirst(types []introspection.ImageTypeInfo) []introspection.ImageTypeInfo {
	ordered := append([]introspection.ImageTypeInfo(nil), types...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return len(ordered[i].TypeName) > len(ordered[j].TypeName)
	})
	return ordered
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
		VipsVersion:      vipsVersion,
		Operations:       operations,
		EnumTypes:        enumTypes,
		ImageTypes:       imageTypes,
		IncludeTest:      includeTest,
		Coverage:         BuildOperationCoverage(operations),
		OptionDefaults:   BuildOptionDefaults(operations),
		LoaderImageTypes: longestNameFirst(imageTypes),
	}
}
