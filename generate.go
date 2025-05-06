package vipsgen

import (
	"fmt"
	"github.com/cshum/vipsgen/templateloader"
	"os"
	"path/filepath"
)

// Generate generates all code files from templates using the unified template data
func Generate(
	templateLoader templateloader.TemplateLoader,
	templateData *TemplateData,
	outputDir string,
) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Map of template files to output files
	templateMapping := map[string]string{
		"vips.go.tmpl":    filepath.Join(outputDir, "vips.go"),
		"vips.h.tmpl":     filepath.Join(outputDir, "vips.h"),
		"vips.c.tmpl":     filepath.Join(outputDir, "vips.c"),
		"image.go.tmpl":   filepath.Join(outputDir, "image.go"),
		"types.go.tmpl":   filepath.Join(outputDir, "types.go"),
		"foreign.go.tmpl": filepath.Join(outputDir, "foreign.go"),
		"foreign.h.tmpl":  filepath.Join(outputDir, "foreign.h"),
		"foreign.c.tmpl":  filepath.Join(outputDir, "foreign.c"),
	}

	// Generate all files
	generatedFiles := []string{}
	for templateFile, outputFile := range templateMapping {
		if err := templateLoader.GenerateFile(templateFile, outputFile, templateData); err != nil {
			return fmt.Errorf("failed to generate %s: %v", outputFile, err)
		}
		generatedFiles = append(generatedFiles, outputFile)
	}

	// Process static files - this simply copies them with the .tmpl extension removed
	if err := templateLoader.ProcessStaticFiles(outputDir); err != nil {
		return fmt.Errorf("failed to process static files: %v", err)
	}

	fmt.Printf("\nSuccessfully generated files from templates: %d\n", len(generatedFiles))
	for _, file := range generatedFiles {
		fmt.Printf("  - %s\n", file)
	}
	fmt.Println("\nAdditional static files were also copied to the output directory.")

	return nil
}
