package main

import (
	"flag"
	"fmt"
	"github.com/cshum/vipsgen"
	"github.com/cshum/vipsgen/introspection"
	"github.com/cshum/vipsgen/templateloader"
	"log"
)

func main() {
	// Define flags
	extractTemplates := flag.Bool("extract", false, "Extract embedded templates to a directory")
	extractDir := flag.String("extract-dir", "./templates", "Directory to extract templates to")
	outputDirFlag := flag.String("out", "./out", "Output directory")
	templateDirFlag := flag.String("templates", "", "Template directory (uses embedded templates if not specified)")

	flag.Parse()

	// Extract templates and exit if requested
	if *extractTemplates {
		if err := templateloader.ExtractEmbeddedFilesystem(vipsgen.EmbeddedTemplates, *extractDir); err != nil {
			log.Fatalf("Failed to extract templates: %v", err)
		}

		fmt.Printf("Templates and static files extracted to: %s\n", *extractDir)
		return
	}

	var outputDir string
	var loader templateloader.TemplateLoader
	var funcMap = vipsgen.GetTemplateFuncMap()

	// Determine template source - use embedded by default, external if specified
	if *templateDirFlag != "" {
		// Use specified template directory
		var err error
		loader, err = templateloader.NewOSTemplateLoader(*templateDirFlag, funcMap)
		if err != nil {
			log.Fatalf("Failed to create template loader: %v", err)
		}
		fmt.Printf("Using templates from: %s\n", *templateDirFlag)
	} else {
		// Use embedded templates by default
		loader = templateloader.NewEmbeddedTemplateLoader(vipsgen.EmbeddedTemplates, funcMap)
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

	// Create operation manager
	vipsIntrospection := introspection.NewIntrospection()

	// Get all operations
	operations := vipsIntrospection.IntrospectOperations()
	fmt.Printf("Found %d operations with required inputs\n", len(operations))

	// Filter operations
	filteredOps := vipsIntrospection.FilterOperations(operations)
	fmt.Printf("Filtered to %d operations\n", len(filteredOps))

	// Extract image types from operations
	imageTypes := vipsIntrospection.DiscoverImageTypes()

	// Get enum types
	enumTypes := vipsIntrospection.GetEnumTypes()

	// Discover supported savers using the existing function
	supportedSavers := vipsIntrospection.DiscoverSupportedSavers()
	fmt.Printf("Discovered supported savers:\n")
	for name, supported := range supportedSavers {
		if supported {
			fmt.Printf("  - %s: supported\n", name)
		}
	}

	// Create unified template data
	templateData := vipsgen.NewTemplateData(filteredOps, enumTypes, imageTypes, supportedSavers)

	// Generate all code using the unified template data approach
	if err := vipsgen.Generate(loader, templateData, outputDir); err != nil {
		log.Fatalf("Failed to generate code: %v", err)
	}
}
