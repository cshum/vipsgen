package vipsgen

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// FSTemplateLoader loads templates from any fs.FS implementation
type FSTemplateLoader struct {
	fs         fs.FS
	funcMap    template.FuncMap
	staticRoot string
	tmplRoot   string
}

// NewFSTemplateLoader creates a new template loader from any fs.FS implementation
func NewFSTemplateLoader(filesystem fs.FS, funcMap template.FuncMap, staticRoot, tmplRoot string) TemplateLoader {
	return &FSTemplateLoader{
		fs:         filesystem,
		funcMap:    funcMap,
		staticRoot: staticRoot,
		tmplRoot:   tmplRoot,
	}
}

// NewEmbeddedTemplateLoader creates a template loader from an embedded filesystem
func NewEmbeddedTemplateLoader(embeddedFS fs.FS, funcMap template.FuncMap) TemplateLoader {
	return &FSTemplateLoader{
		fs:         embeddedFS,
		funcMap:    funcMap,
		staticRoot: "statics",
		tmplRoot:   "templates",
	}
}

// NewOSTemplateLoader creates a template loader from the OS filesystem
func NewOSTemplateLoader(rootDir string, funcMap template.FuncMap) (TemplateLoader, error) {
	// Check if template directory exists
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("template directory does not exist: %s", rootDir)
	}

	return &FSTemplateLoader{
		fs:         os.DirFS(rootDir),
		funcMap:    funcMap,
		staticRoot: "statics",
		tmplRoot:   "templates",
	}, nil
}

// LoadTemplate loads a template from the filesystem
func (t *FSTemplateLoader) LoadTemplate(name string) (*template.Template, error) {
	var templatePath string

	// For static templates, check in statics directory
	if strings.HasPrefix(name, t.staticRoot+"/") {
		// Keep the path as is for static templates
		templatePath = name
	} else {
		// For regular templates, look in templates directory
		templatePath = filepath.Join(t.tmplRoot, name)
	}

	// Read template content
	content, err := fs.ReadFile(t.fs, templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template %s: %v", templatePath, err)
	}

	// Parse template
	tmpl, err := template.New(name).Funcs(t.funcMap).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %v", err)
	}

	return tmpl, nil
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

// ListTemplateFiles returns a list of all template files
func (t *FSTemplateLoader) ListTemplateFiles() ([]string, error) {
	var templateFiles []string

	// Walk template directory
	err := fs.WalkDir(t.fs, t.tmplRoot, func(path string, d fs.DirEntry, err error) error {
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
			relPath, err := filepath.Rel(t.tmplRoot, path)
			if err != nil {
				return fmt.Errorf("failed to get relative path: %v", err)
			}
			templateFiles = append(templateFiles, relPath)
		}

		return nil
	})

	// Handle the case where templates directory doesn't exist
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to list template files: %v", err)
	}

	return templateFiles, nil
}

// copyStaticFile copies a static file without template processing
func (t *FSTemplateLoader) copyStaticFile(staticPath, outputPath string) error {
	// Open the static file
	file, err := t.fs.Open(staticPath)
	if err != nil {
		return fmt.Errorf("failed to open static file %s: %v", staticPath, err)
	}
	defer file.Close()

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Create the output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	// Copy the content
	_, err = io.Copy(outFile, file)
	if err != nil {
		return fmt.Errorf("failed to copy static file: %v", err)
	}

	return nil
}

// ProcessStaticFiles processes all static files (copies them without template processing)
func (t *FSTemplateLoader) ProcessStaticFiles(outputDir string) error {
	// Walk the statics directory
	err := fs.WalkDir(t.fs, t.staticRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process .tmpl files
		if !strings.HasSuffix(d.Name(), ".tmpl") {
			return nil
		}

		// Get the relative path from the statics root
		relPath, err := filepath.Rel(t.staticRoot, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %v", err)
		}

		// Create the output path (without .tmpl extension)
		outputPath := filepath.Join(outputDir, strings.TrimSuffix(relPath, ".tmpl"))

		// Copy the file
		if err := t.copyStaticFile(path, outputPath); err != nil {
			return err
		}

		fmt.Printf("  - Copied static file: %s\n", strings.TrimSuffix(relPath, ".tmpl"))
		return nil
	})

	// It's okay if the statics directory doesn't exist
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to process static files: %v", err)
	}

	return nil
}

// ExtractEmbeddedFilesystem extracts an embedded filesystem to a directory
func ExtractEmbeddedFilesystem(filesystem fs.FS, destDir string) error {
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
