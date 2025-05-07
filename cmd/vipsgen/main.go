package main

import (
	"flag"
	"fmt"
	"github.com/cshum/vipsgen"
	"github.com/cshum/vipsgen/girintrospection"
	"github.com/cshum/vipsgen/introspection"
	"io"
	"log"
	"os"
)

func main() {
	// Define flags
	extractTemplates := flag.Bool("extract", false, "Extract embedded templates to a directory")
	extractDir := flag.String("extract-dir", "./templates", "Directory to extract templates to")
	outputDirFlag := flag.String("out", "./out", "Output directory")
	templateDirFlag := flag.String("templates", "", "Template directory (uses embedded templates if not specified)")
	girFileFlag := flag.String("gir-file", "", "Path to GIR file (uses embedded GIR file if not specified)")

	flag.Parse()

	// Extract templates and exit if requested
	if *extractTemplates {
		if err := vipsgen.ExtractEmbeddedFilesystem(vipsgen.EmbeddedTemplates, *extractDir); err != nil {
			log.Fatalf("Failed to extract templates: %v", err)
		}

		fmt.Printf("Templates and static files extracted to: %s\n", *extractDir)
		return
	}

	var outputDir string
	var loader vipsgen.TemplateLoader
	var funcMap = vipsgen.GetTemplateFuncMap()

	// Determine template source - use embedded by default, external if specified
	if *templateDirFlag != "" {
		// Use specified template directory
		var err error
		loader, err = vipsgen.NewOSTemplateLoader(*templateDirFlag, funcMap)
		if err != nil {
			log.Fatalf("Failed to create template loader: %v", err)
		}
		fmt.Printf("Using templates from: %s\n", *templateDirFlag)
	} else {
		// Use embedded templates by default
		loader = vipsgen.NewEmbeddedTemplateLoader(vipsgen.EmbeddedTemplates, funcMap)
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

	// Get all operations from C-based introspection
	operations := vipsIntrospection.IntrospectOperations()
	fmt.Printf("Found %d operations with required inputs from C introspection\n", len(operations))

	// Filter operations
	filteredOps := vipsIntrospection.FilterOperations(operations)
	fmt.Printf("Filtered to %d operations\n", len(filteredOps))

	// Extract image types from operations
	imageTypes := vipsIntrospection.DiscoverImageTypes()

	// Get enum types
	enumTypes := vipsIntrospection.GetEnumTypes()

	// Discover supported savers
	supportedSavers := vipsIntrospection.DiscoverSupportedSavers()
	fmt.Printf("Discovered supported savers:\n")
	for name, supported := range supportedSavers {
		if supported {
			fmt.Printf("  - %s: supported\n", name)
		}
	}

	// Get GIR data
	var girFile io.Reader
	var err error

	// Determine GIR file source
	if *girFileFlag != "" {
		// Use specified GIR file
		fmt.Printf("Parsing GIR file: %s\n", *girFileFlag)
		girFile, err = os.Open(*girFileFlag)
		if err != nil {
			log.Fatalf("Failed to open GIR file: %v", err)
		}
		defer girFile.(io.Closer).Close()
	} else {
		// Use embedded GIR file
		fmt.Println("Using embedded GIR file")
		girFile, err = vipsgen.EmbeddedTemplates.Open("vips-8.0.gir")
		if err != nil {
			log.Fatalf("Failed to open embedded GIR file: %v", err)
		}
		defer girFile.(io.Closer).Close()
	}

	// Parse GIR file
	parser := girintrospection.New()
	if err := parser.Parse(girFile); err != nil {
		log.Fatalf("Failed to parse GIR file: %v", err)
	}

	// Get operations from GIR
	girOps := parser.ConvertToVipsgenOperations()
	fmt.Printf("Extracted %d operations from GIR file\n", len(girOps))

	// TODO: Create a combined template data that uses both C introspection and GIR data
	// For now, just use the C-based introspection operations

	// Create unified template data
	templateData := vipsgen.NewTemplateData(filteredOps, enumTypes, imageTypes, supportedSavers)

	// Generate all code using the unified template data approach
	if err := vipsgen.Generate(loader, templateData, outputDir); err != nil {
		log.Fatalf("Failed to generate code: %v", err)
	}
}
