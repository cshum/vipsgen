package templateloader

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// FileTemplateLoader loads and manages templates from a directory
type FileTemplateLoader struct {
	templateDir string
	funcMap     template.FuncMap
}

// NewTemplateLoader creates a new template loader
func NewTemplateLoader(templateDir string, funcMap template.FuncMap) (TemplateLoader, error) {
	// Check if template directory exists
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("template directory does not exist: %s", templateDir)
	}

	return &FileTemplateLoader{
		templateDir: templateDir,
		funcMap:     funcMap,
	}, nil
}

// LoadTemplate loads a template from the template directory
func (t *FileTemplateLoader) LoadTemplate(name string) (*template.Template, error) {
	templatePath := filepath.Join(t.templateDir, name)

	// Check if template file exists
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("template file does not exist: %s", templatePath)
	}

	// Load template content
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %v", err)
	}

	// Parse template
	tmpl, err := template.New(name).Funcs(t.funcMap).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %v", err)
	}

	return tmpl, nil
}

// GenerateFile generates a file using a template and data
func (t *FileTemplateLoader) GenerateFile(templateName, outputFile string, data interface{}) error {
	tmpl, err := t.LoadTemplate(templateName)
	if err != nil {
		return err
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Create output file
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	return nil
}
