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
	applyEnumOverrides(enumTypes)
	return &TemplateData{
		VipsVersion: vipsVersion,
		Operations:  operations,
		EnumTypes:   enumTypes,
		ImageTypes:  imageTypes,
		IncludeTest: includeTest,
	}
}

// applyEnumOverrides post-processes discovered enum types to apply Go-side value
// overrides. This keeps special-case logic out of templates.
func applyEnumOverrides(enumTypes []introspection.EnumTypeInfo) {
	for i, et := range enumTypes {
		if et.GoName == "Keep" {
			for j, v := range et.Values {
				if v.GoName == "KeepNone" {
					// KeepNone is remapped to -1 in Go so that the zero value of a
					// Keep field (0) means "not set" and is safe in empty structs.
					// vipsgen_set_keep translates Go -1 back to C VIPS_FOREIGN_KEEP_NONE (0).
					enumTypes[i].Values[j].GoValue = "-1"
				}
			}
		}
	}
}
