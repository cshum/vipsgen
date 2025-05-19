package main

import (
	"flag"
	"fmt"
	"github.com/cshum/vipsgen/internal/generator"
	"github.com/cshum/vipsgen/internal/introspection"
	"github.com/cshum/vipsgen/internal/templates"
	"log"
)

func main() {
	// Define flags
	extractTemplates := flag.Bool("extract", false, "Extract embedded templates to a directory")
	extractDir := flag.String("extract-dir", "./templates", "Directory to extract templates to")
	outputDirFlag := flag.String("out", "./vips", "Output directory")
	templateDirFlag := flag.String("templates", "", "Template directory (uses embedded templates if not specified)")

	flag.Parse()

	// Extract templates and exit if requested
	if *extractTemplates {
		if err := generator.ExtractEmbeddedFS(templates.Templates, *extractDir); err != nil {
			log.Fatalf("Failed to extract templates: %v", err)
		}

		fmt.Printf("Templates and static files extracted to: %s\n", *extractDir)
		return
	}

	var outputDir string
	var loader generator.TemplateLoader
	var funcMap = generator.GetTemplateFuncMap()

	// Determine template source - use embedded by default, external if specified
	if *templateDirFlag != "" {
		// Use specified template directory
		var err error
		loader, err = generator.NewOSTemplateLoader(*templateDirFlag, funcMap)
		if err != nil {
			log.Fatalf("Failed to create template loader: %v", err)
		}
		fmt.Printf("Using templates from: %s\n", *templateDirFlag)
	} else {
		// Use embedded templates by default
		loader = generator.NewFSTemplateLoader(templates.Templates, funcMap)
		fmt.Println("Using embedded templates")
	}

	// Determine output directory
	if *outputDirFlag != "" {
		outputDir = *outputDirFlag
	} else if flag.NArg() > 0 {
		outputDir = flag.Arg(0)
	} else {
		outputDir = "./out"
	}

	// Create operation manager for C-based introspection
	vipsIntrospection := introspection.NewIntrospection()

	// Extract image types from operations
	imageTypes := vipsIntrospection.DiscoverImageTypes()

	// Discover supported savers
	supportedSavers := vipsIntrospection.DiscoverSupportedSavers()
	fmt.Printf("Discovered supported savers:\n")
	for name, supported := range supportedSavers {
		if supported {
			fmt.Printf("  - %s: supported\n", name)
		}
	}

	// Convert GIR data to vipsgen.Operation format
	operations := vipsIntrospection.DiscoverOperations()
	fmt.Printf("Extracted %d operations from GObject Introspection\n", len(operations))

	// Get enum types
	enumTypes := vipsIntrospection.GetEnumTypes()
	fmt.Printf("Discovered %d enum types\n", len(enumTypes))

	// Create unified template data
	templateData := generator.NewTemplateData(operations, enumTypes, imageTypes, supportedSavers)

	// Generate all code using the unified template data approach
	if err := generator.Generate(loader, templateData, outputDir); err != nil {
		log.Fatalf("Failed to generate code: %v", err)
	}
}
