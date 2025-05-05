package vipsgen

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var EmbeddedTemplates embed.FS

// GetTemplateFuncMap Helper functions for templates
func GetTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"formatArgs": func(args []Argument) string {
			var params []string
			for _, arg := range args {
				params = append(params, fmt.Sprintf("%s %s", arg.GoName, arg.GoType))
			}
			return strings.Join(params, ", ")
		},
		"hasVipsImageInput": func(args []Argument) bool {
			for _, arg := range args {
				if arg.Type == "VipsImage" {
					return true
				}
			}
			return false
		},
		"cArgList": func(args []Argument) string {
			var params []string
			for _, arg := range args {
				params = append(params, fmt.Sprintf("%s %s", arg.CType, arg.Name))
			}
			return strings.Join(params, ", ")
		},
		"callArgList": func(args []Argument) string {
			var params []string
			for _, arg := range args {
				params = append(params, arg.Name)
			}
			return strings.Join(params, ", ")
		},
		"imageMethodName":       ImageMethodName,
		"formatImageMethodArgs": FormatImageMethodArgs,
	}
}
