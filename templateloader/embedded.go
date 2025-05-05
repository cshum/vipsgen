package templateloader

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"
)

// EmbeddedTemplateLoader is a template loader that uses embedded templates
type EmbeddedTemplateLoader struct {
	embedded embed.FS
	funcMap  template.FuncMap
}

// NewEmbeddedTemplateLoader creates a new template loader with embedded templates
func NewEmbeddedTemplateLoader(embedded embed.FS, funcMap template.FuncMap) TemplateLoader {
	return &EmbeddedTemplateLoader{
		embedded: embedded,
		funcMap:  funcMap,
	}
}

// LoadTemplate loads a template from embedded FS
func (t *EmbeddedTemplateLoader) LoadTemplate(name string) (*template.Template, error) {
	templatePath := filepath.Join("templates", name)

	// Read template content
	content, err := t.embedded.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded template %s: %v", templatePath, err)
	}

	// Parse template
	tmpl, err := template.New(name).Funcs(t.funcMap).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %v", err)
	}

	return tmpl, nil
}

// GenerateFile generates a file using a template and data
func (t *EmbeddedTemplateLoader) GenerateFile(templateName, outputFile string, data interface{}) error {
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

// ExtractEmbeddedTemplates extracts embedded templates to a directory
func ExtractEmbeddedTemplates(embeddedTemplates embed.FS, destDir string) error {
	// Create the destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create template directory: %v", err)
	}

	// Walk through the embedded templates
	return fs.WalkDir(embeddedTemplates, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Read template content
		content, err := embeddedTemplates.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded template %s: %v", path, err)
		}

		// Create destination file path
		relativePath, err := filepath.Rel("templates", path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %v", err)
		}
		destPath := filepath.Join(destDir, relativePath)

		// Create destination file
		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write template to %s: %v", destPath, err)
		}

		return nil
	})
}
