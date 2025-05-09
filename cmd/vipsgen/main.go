package main

import (
	"flag"
	"fmt"
	"github.com/cshum/vipsgen"
	"github.com/cshum/vipsgen/girparser"
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
	useGIRParser := flag.Bool("use-gir-parser", true, "Use the GIR parser for operation discovery (recommended)")

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
	cIntrospectionOps := vipsIntrospection.IntrospectOperations()
	fmt.Printf("Found %d operations with required inputs from C introspection\n", len(cIntrospectionOps))

	// Filter operations
	filteredCOps := vipsIntrospection.FilterOperations(cIntrospectionOps)
	fmt.Printf("Filtered to %d operations from C introspection\n", len(filteredCOps))

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

	// Determine the operations to use
	var operations []vipsgen.Operation

	if *useGIRParser {
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

		// Use the new GIR parser
		parser := girparser.NewVipsGIRParser()
		if err := parser.Parse(girFile); err != nil {
			log.Fatalf("Failed to parse GIR file: %v", err)
		}

		// Convert GIR data to vipsgen.Operation format
		girOps := parser.ConvertToVipsgenOperations()
		fmt.Printf("Extracted %d operations from GIR file\n", len(girOps))

		// Create a map of GIR operations for lookup
		girOpMap := make(map[string]vipsgen.Operation)
		for _, op := range girOps {
			girOpMap[op.Name] = op
		}

		// Enhance C operations with GIR data
		var enhancedOps []vipsgen.Operation
		girOpsUsed := 0

		for _, cOp := range filteredCOps {
			// Check if we have GIR data for this operation
			if girOp, exists := girOpMap[cOp.Name]; exists {
				// Use GIR operation as basis but keep C introspection metadata if needed
				enhancedOp := girOp

				// If GIR operation is missing some metadata, use C introspection data
				if enhancedOp.Category == "" {
					enhancedOp.Category = cOp.Category
				}
				if !enhancedOp.HasImageInput && cOp.HasImageInput {
					enhancedOp.HasImageInput = true
				}
				if !enhancedOp.HasImageOutput && cOp.HasImageOutput {
					enhancedOp.HasImageOutput = true
				}

				enhancedOps = append(enhancedOps, enhancedOp)
				delete(girOpMap, cOp.Name) // Remove from map to track used operations
				girOpsUsed++
			} else {
				// No GIR data, use C operation as is
				enhancedOps = append(enhancedOps, cOp)
			}
		}

		// Add any remaining GIR operations (not found in C introspection)
		remainingGirOps := len(girOpMap)
		for _, op := range girOpMap {
			enhancedOps = append(enhancedOps, op)
		}

		fmt.Printf("Enhanced %d C operations with GIR data\n", girOpsUsed)
		fmt.Printf("Added %d operations found only in GIR\n", remainingGirOps)

		operations = enhancedOps
		fmt.Printf("Using %d operations in total\n", len(operations))
	} else {
		// Use only C introspection operations
		operations = filteredCOps
		fmt.Printf("Using %d operations from C introspection only\n", len(operations))
	}

	// Create unified template data
	templateData := vipsgen.NewTemplateData(operations, enumTypes, imageTypes, supportedSavers)

	// Generate all code using the unified template data approach
	if err := vipsgen.Generate(loader, templateData, outputDir); err != nil {
		log.Fatalf("Failed to generate code: %v", err)
	}
}

// This function is no longer used as we directly enhance operations in the main function
