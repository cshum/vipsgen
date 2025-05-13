package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Generate generates all code files from templates by scanning the template directory
func Generate(
	templateLoader TemplateLoader,
	templateData *TemplateData,
	outputDir string,
) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Get all template files
	templateFiles, err := templateLoader.ListFiles()
	if err != nil {
		return fmt.Errorf("failed to list template files: %v", err)
	}

	// Generate files from templates
	var generatedFiles []string

	for _, templateFile := range templateFiles {
		// Convert template name to output filename
		// For example: "vips.go.tmpl" -> "vips.go"
		outputFile := filepath.Join(outputDir, strings.TrimSuffix(filepath.Base(templateFile), ".tmpl"))

		// Generate file from template
		if err := templateLoader.GenerateFile(templateFile, outputFile, templateData); err != nil {
			return fmt.Errorf("failed to generate %s: %v", outputFile, err)
		}
		generatedFiles = append(generatedFiles, outputFile)
	}

	fmt.Printf("\nSuccessfully generated files from templates: %d\n", len(generatedFiles))
	for _, file := range generatedFiles {
		fmt.Printf("  - %s\n", file)
	}
	fmt.Println("\nAdditional static files were also copied to the output directory.")

	return nil
}
