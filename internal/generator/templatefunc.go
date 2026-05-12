package generator

import (
	"text/template"

	"github.com/cshum/vipsgen/internal/introspection"
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
	}
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
