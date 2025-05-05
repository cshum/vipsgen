package vipsgen

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl statics/*.tmpl
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

// ProcessStaticTemplates copies all files from the statics directory to the output directory,
// removing the .tmpl extension in the process
func ProcessStaticTemplates(embeddedFS embed.FS, outputDir, templateDir string) error {
	// Determine the source of static files
	var staticTemplates []string
	var err error

	if templateDir != "" {
		// If using external template directory, read from there
		staticDir := filepath.Join(templateDir, "statics")
		if _, err := os.Stat(staticDir); !os.IsNotExist(err) {
			// Read directory contents
			entries, err := os.ReadDir(staticDir)
			if err != nil {
				return fmt.Errorf("failed to read static directory: %v", err)
			}

			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tmpl") {
					staticTemplates = append(staticTemplates, entry.Name())
				}
			}
		}
	} else {
		// If using embedded templates, read from the embedded filesystem
		entries, err := fs.ReadDir(embeddedFS, "statics")
		if err != nil {
			return fmt.Errorf("failed to read embedded statics directory: %v", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tmpl") {
				staticTemplates = append(staticTemplates, entry.Name())
			}
		}
	}

	// Process each static template file
	for _, filename := range staticTemplates {
		var content []byte

		if templateDir != "" {
			// Try loading from external template directory first
			sourcePath := filepath.Join(templateDir, "statics", filename)
			content, err = os.ReadFile(sourcePath)
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to read static template file %s: %v", sourcePath, err)
			}
		}

		// If content is still nil, try loading from embedded FS
		if content == nil {
			content, err = embeddedFS.ReadFile(filepath.Join("statics", filename))
			if err != nil {
				return fmt.Errorf("failed to read embedded static template file %s: %v", filename, err)
			}
		}

		// Create output file (without .tmpl extension)
		outputFilename := strings.TrimSuffix(filename, ".tmpl")
		outputPath := filepath.Join(outputDir, outputFilename)
		outputDir := filepath.Dir(outputPath)

		// Create directory if it doesn't exist
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory for static template file: %v", err)
		}

		// Write the file
		if err := os.WriteFile(outputPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write static template file %s: %v", outputPath, err)
		}

		fmt.Printf("  - Copied static file: %s\n", outputFilename)
	}

	return nil
}

// ExtractStaticTemplates extracts all files from statics directory to the destination
func ExtractStaticTemplates(embeddedFS embed.FS, destDir string) error {
	// Create the statics directory in the destination
	staticDir := filepath.Join(destDir, "statics")
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		return fmt.Errorf("failed to create statics directory: %v", err)
	}

	// Read all entries in the statics directory
	entries, err := fs.ReadDir(embeddedFS, "statics")
	if err != nil {
		return fmt.Errorf("failed to read embedded statics directory: %v", err)
	}

	// Extract each static template file
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Get the file content
		content, err := embeddedFS.ReadFile(filepath.Join("statics", entry.Name()))
		if err != nil {
			return fmt.Errorf("failed to read embedded static template file %s: %v", entry.Name(), err)
		}

		// Create destination file path
		destPath := filepath.Join(staticDir, entry.Name())

		// Write the file
		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write static template file to %s: %v", destPath, err)
		}

		fmt.Printf("  - Extracted static template file: %s\n", entry.Name())
	}

	return nil
}
