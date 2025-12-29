package main

import (
	"flag"
	"log"

	"github.com/cshum/vipsgen/internal/generator"
	"github.com/cshum/vipsgen/internal/introspection"
	"github.com/cshum/vipsgen/internal/templates"
)

func main() {
	extractTemplates := flag.Bool("extract", false, "Extract embedded templates to a directory")
	extractDir := flag.String("extract-dir", "./templates", "Directory to extract templates to")
	outputDirFlag := flag.String("out", "./vips", "Output directory")
	templateDirFlag := flag.String("templates", "", "Template directory (uses embedded templates if not specified)")
	isDebug := flag.Bool("debug", false, "Enable debug json output")
	includeTest := flag.Bool("include-test", false, "Include test files in generated output")

	flag.Parse()

	// Extract templates and exit if requested
	if *extractTemplates {
		if err := generator.ExtractEmbeddedFS(templates.Templates, *extractDir); err != nil {
			log.Fatalf("Failed to extract templates: %v", err)
		}

		log.Printf("Templates and static files extracted to: %s\n", *extractDir)
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
		log.Printf("Using templates from: %s\n", *templateDirFlag)
	} else {
		// Use embedded templates by default
		loader = generator.NewFSTemplateLoader(templates.Templates, funcMap)
		log.Printf("Using embedded templates\n")
	}

	// Determine output directory
	if *outputDirFlag != "" {
		outputDir = *outputDirFlag
	} else if flag.NArg() > 0 {
		outputDir = flag.Arg(0)
	} else {
		outputDir = "./vips"
	}

	// Create operation manager for C-based introspection
	vipsIntrospection := introspection.NewIntrospection(*isDebug)

	// Get libvips version from introspection
	vipsVersion := vipsIntrospection.GetVipsVersion()

	// Extract image types from operations
	imageTypes := vipsIntrospection.DiscoverImageTypes()

	// Convert GIR data to vipsgen.Operation format
	operations := vipsIntrospection.DiscoverOperations()
	log.Printf("Extracted %d operations from GObject Introspection\n", len(operations))

	// Get enum types
	enumTypes := vipsIntrospection.DiscoverEnumTypes()
	log.Printf("Discovered %d enum types\n", len(enumTypes))

	// Create unified template data
	templateData := generator.NewTemplateData(vipsVersion, operations, enumTypes, imageTypes, *includeTest)

	// Generate all code using the unified template data approach
	if err := generator.Generate(loader, templateData, outputDir); err != nil {
		log.Fatalf("Failed to generate code: %v", err)
	}
}
