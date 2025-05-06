package templateloader

import (
	"text/template"
)

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
