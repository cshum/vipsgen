package generator

import (
	"text/template"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// GetTemplateFuncMap Helper functions for templates
func GetTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"generateGoFunctionBody":             generateGoFunctionBody,
		"generateFunctionCallArgs":           generateFunctionCallArgs,
		"generateFunctionCall":               generateFunctionCall,
		"generateImageMethodBody":            generateImageMethodBody,
		"generateImageArgumentsComment":      generateImageArgumentsComment,
		"generateImageMethodParams":          generateImageMethodParams,
		"generateImageMethodReturnTypes":     generateImageMethodReturnTypes,
		"generateMethodParams":               generateMethodParams,
		"generateCreatorMethodBody":          generateCreatorMethodBody,
		"generateCFunctionDeclaration":       generateCFunctionDeclaration,
		"generateCFunctionImplementation":    generateCFunctionImplementation,
		"generateOptionalInputsStruct":       generateOptionalInputsStruct,
		"generateUtilFunctionCallArgs":       generateUtilFunctionCallArgs,
		"generateUtilityFunctionReturnTypes": generateUtilityFunctionReturnTypes,
		"getSupportedOptionalOutputs":        getSupportedOptionalOutputs,
		"hasWithOptionsVariant":              hasWithOptionsVariant,
		"hasImageType":                       hasImageType,
		"hasOperation":                       hasOperation,
	}
}

// hasImageType reports whether libvips exposed an image type, so templates can
// keep version-specific cases out of packages generated for older releases.
func hasImageType(types []introspection.ImageTypeInfo, name string) bool {
	for _, t := range types {
		if t.TypeName == name {
			return true
		}
	}
	return false
}

// hasOperation reports whether libvips exposed an operation, by Go name.
func hasOperation(ops []introspection.Operation, goName string) bool {
	for _, op := range ops {
		if op.GoName == goName {
			return true
		}
	}
	return false
}

// getSupportedOptionalOutputs returns optional outputs that are supported for capture
func getSupportedOptionalOutputs(op introspection.Operation) []introspection.Argument {
	var supported []introspection.Argument
	for _, arg := range op.OptionalOutputs {
		if arg.GoType == "int" || arg.GoType == "float64" || arg.GoType == "bool" {
			supported = append(supported, arg)
		}
	}
	return supported
}

// hasWithOptionsVariant determines if an operation should have a _with_options variant
func hasWithOptionsVariant(op introspection.Operation) bool {
	return len(op.OptionalInputs) > 0 || len(getSupportedOptionalOutputs(op)) > 0
}
