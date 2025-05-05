package main

import (
	"flag"
	"fmt"
	"github.com/cshum/vipsgen"
	"github.com/cshum/vipsgen/introspection"
	"github.com/cshum/vipsgen/templateloader"
	"log"
	"os"
	"path/filepath"
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

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Process static files - this simply copies them with the .tmpl extension removed
	if err := loader.ProcessStaticFiles(outputDir); err != nil {
		log.Fatalf("Failed to process static files: %v", err)
	}

	// Create operation manager
	vipsIntrospection := introspection.NewIntrospection()

	// Get all operations
	operations := vipsIntrospection.IntrospectOperations()
	fmt.Printf("Found %d operations with required inputs\n", len(operations))

	// Filter operations
	filteredOps := vipsIntrospection.FilterOperations(operations)
	fmt.Printf("Filtered to %d operations\n", len(filteredOps))

	// Group operations by category
	categories := make(map[string][]vipsgen.Operation)
	for _, op := range filteredOps {
		categories[op.Category] = append(categories[op.Category], op)
	}

	// Generate Go file with operations
	goFile := filepath.Join(outputDir, "vips.go")
	if err := vipsgen.OperationsFile(loader, goFile, filteredOps); err != nil {
		log.Fatalf("Failed to generate operations file: %v", err)
	}

	// Generate C header file
	hFile := filepath.Join(outputDir, "vips.h")
	if err := vipsgen.HeaderFile(loader, hFile, filteredOps); err != nil {
		log.Fatalf("Failed to generate header file: %v", err)
	}

	// Generate C source file
	cFile := filepath.Join(outputDir, "vips.c")
	if err := vipsgen.SourceFile(loader, cFile, filteredOps); err != nil {
		log.Fatalf("Failed to generate source file: %v", err)
	}

	// Extract image types from operations
	imageTypes := vipsIntrospection.DiscoverImageTypes()

	// Generate image file with methods
	imageFile := filepath.Join(outputDir, "image.go")
	if err := vipsgen.ImageFile(loader, imageFile, imageTypes, filteredOps); err != nil {
		log.Fatalf("Failed to generate image file: %v", err)
	}

	// Generate types file with enums
	typesFile := filepath.Join(outputDir, "types.go")
	enumTypes := vipsIntrospection.GetEnumTypes()
	if err := vipsgen.TypesFile(loader, typesFile, enumTypes, imageTypes); err != nil {
		log.Fatalf("Failed to generate types file: %v", err)
	}

	// List of generated files from templates
	generatedFiles := []string{goFile, hFile, cFile, imageFile, typesFile}

	// Discover supported savers
	supportedSavers := vipsIntrospection.DiscoverSupportedSavers()
	fmt.Printf("Discovered supported savers:\n")
	for name, supported := range supportedSavers {
		if supported {
			fmt.Printf("  - %s: supported\n", name)
		}
	}

	// Generate foreign support files (Go, H, and C)
	if err := vipsgen.ForeignFiles(loader, outputDir, supportedSavers); err != nil {
		log.Fatalf("Failed to generate foreign files: %v", err)
	}

	// Add the foreign files to the list of generated files
	foreignFiles := []string{
		filepath.Join(outputDir, "foreign.go"),
		filepath.Join(outputDir, "foreign.h"),
		filepath.Join(outputDir, "foreign.c"),
	}
	generatedFiles = append(generatedFiles, foreignFiles...)

	fmt.Printf("\nSuccessfully generated files from templates: %d\n", len(generatedFiles))
	for _, file := range generatedFiles {
		fmt.Printf("  - %s\n", file)
	}

	fmt.Println("\nAdditional static files were also copied to the output directory.")
}
