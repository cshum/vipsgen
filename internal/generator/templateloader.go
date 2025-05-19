package generator

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// TemplateLoader is an interface for loading and generating files from templates
type TemplateLoader interface {
	// LoadTemplate loads a template by name
	LoadTemplate(name string) (*template.Template, error)

	// ListFiles returns a list of all template files
	ListFiles() ([]string, error)

	// GenerateFile generates a file using a template and data
	GenerateFile(templateName, outputFile string, data interface{}) error
}

// FSTemplateLoader loads templates from any fs.FS implementation
type FSTemplateLoader struct {
	fs      fs.FS
	funcMap template.FuncMap
}

// NewFSTemplateLoader creates a new template loader from any fs.FS implementation
func NewFSTemplateLoader(filesystem fs.FS, funcMap template.FuncMap) TemplateLoader {
	return &FSTemplateLoader{
		fs:      filesystem,
		funcMap: funcMap,
	}
}

// NewOSTemplateLoader creates a template loader from the OS filesystem
func NewOSTemplateLoader(rootDir string, funcMap template.FuncMap) (TemplateLoader, error) {
	// Check if template directory exists
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("template directory does not exist: %s", rootDir)
	}
	return &FSTemplateLoader{
		fs:      os.DirFS(rootDir),
		funcMap: funcMap,
	}, nil
}

// LoadTemplate loads a template from the filesystem
func (t *FSTemplateLoader) LoadTemplate(templatePath string) (*template.Template, error) {
	// Read template content
	content, err := fs.ReadFile(t.fs, templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template %s: %v", templatePath, err)
	}

	// Parse template
	tmpl, err := template.New(templatePath).Funcs(t.funcMap).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %v", err)
	}

	return tmpl, nil
}

// ListFiles returns a list of all template files
func (t *FSTemplateLoader) ListFiles() ([]string, error) {
	var templateFiles []string

	// Walk template directory
	err := fs.WalkDir(t.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only include .tmpl files
		if strings.HasSuffix(d.Name(), ".tmpl") {
			// Convert path to be relative to tmplRoot
			if err != nil {
				return fmt.Errorf("failed to get relative path: %v", err)
			}
			templateFiles = append(templateFiles, path)
		}

		return nil
	})

	// Handle the case where templates directory doesn't exist
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to list template files: %v", err)
	}

	return templateFiles, nil
}

// GenerateFile generates a file using a template and data
func (t *FSTemplateLoader) GenerateFile(templateName, outputFile string, data interface{}) error {
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

// ExtractEmbeddedFS extracts an embedded filesystem to a directory
func ExtractEmbeddedFS(filesystem fs.FS, destDir string) error {
	// Create the destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %v", err)
	}

	// Walk the filesystem recursively
	err := fs.WalkDir(filesystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory
		if path == "." {
			return nil
		}

		// Create directories as needed
		if d.IsDir() {
			dirPath := filepath.Join(destDir, path)
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %v", dirPath, err)
			}
			return nil
		}

		// Extract file
		content, err := fs.ReadFile(filesystem, path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %v", path, err)
		}

		// Create output file
		outPath := filepath.Join(destDir, path)
		if err := os.WriteFile(outPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %v", outPath, err)
		}

		fmt.Printf("  - Extracted: %s\n", path)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to extract filesystem: %v", err)
	}

	return nil
}
