package vipsgen

import (
	"fmt"
	"github.com/cshum/vipsgen/templateloader"
	"path/filepath"
	"strings"
)

// OperationsFile generates the Go file with operations
func OperationsFile(templateLoader templateloader.TemplateLoader, filename string, operations []Operation) error {
	// Collect all enum types used in operations
	enumTypes := make(map[string]bool)
	for _, op := range operations {
		for _, arg := range op.RequiredInputs {
			if arg.IsEnum {
				enumTypes[arg.GoType] = true
			}
		}
	}

	data := struct {
		Operations []Operation
		Config     map[string]OperationConfig
		EnumTypes  map[string]bool
	}{
		Operations: operations,
		Config:     OperationConfigs,
		EnumTypes:  enumTypes,
	}

	return templateLoader.GenerateFile("vips.go.tmpl", filename, data)
}

// HeaderFile generates the C header file
func HeaderFile(templateLoader templateloader.TemplateLoader, filename string, operations []Operation) error {
	data := struct {
		Operations []Operation
	}{
		Operations: operations,
	}

	return templateLoader.GenerateFile("vips.h.tmpl", filename, data)
}

// SourceFile generates the C source file
func SourceFile(templateLoader templateloader.TemplateLoader, filename string, operations []Operation) error {
	data := struct {
		Operations []Operation
		Config     map[string]OperationConfig
	}{
		Operations: operations,
		Config:     OperationConfigs,
	}

	return templateLoader.GenerateFile("vips.c.tmpl", filename, data)
}

// ImageFile generates the combined image file with methods
func ImageFile(templateLoader templateloader.TemplateLoader, filename string, imageTypes []ImageTypeInfo, operations []Operation) error {
	// Filter operations that have Image as first argument and return Image
	var imageOps []Operation
	for _, op := range operations {
		if len(op.RequiredInputs) > 0 && op.RequiredInputs[0].Type == "VipsImage" && op.HasImageOutput {
			imageOps = append(imageOps, op)
		}
	}

	data := struct {
		ImageTypes []ImageTypeInfo
		Operations []Operation
	}{
		ImageTypes: imageTypes,
		Operations: imageOps,
	}

	return templateLoader.GenerateFile("image.go.tmpl", filename, data)
}

// TypesFile generates the types file with enums
func TypesFile(templateLoader templateloader.TemplateLoader, filename string, enumTypes []EnumTypeInfo, imageTypes []ImageTypeInfo) error {
	data := struct {
		EnumTypes  []EnumTypeInfo
		ImageTypes []ImageTypeInfo
	}{
		EnumTypes:  enumTypes,
		ImageTypes: imageTypes,
	}

	return templateLoader.GenerateFile("types.go.tmpl", filename, data)
}

// ForeignFiles generates all foreign functionality files
func ForeignFiles(templateLoader templateloader.TemplateLoader, outputDir string, supportedSavers map[string]bool) error {
	data := struct {
		HasJpegSaver      bool
		HasPngSaver       bool
		HasWebpSaver      bool
		HasTiffSaver      bool
		HasHeifSaver      bool
		HasLegacyGifSaver bool
		HasCgifSaver      bool
		HasAvifSaver      bool
		HasJp2kSaver      bool
		SupportedSavers   []struct {
			EnumName string
			TypeName string
		}
	}{
		HasJpegSaver:      supportedSavers["HasJpegSaver"],
		HasPngSaver:       supportedSavers["HasPngSaver"],
		HasWebpSaver:      supportedSavers["HasWebpSaver"],
		HasTiffSaver:      supportedSavers["HasTiffSaver"],
		HasHeifSaver:      supportedSavers["HasHeifSaver"],
		HasLegacyGifSaver: supportedSavers["HasLegacyGifSaver"],
		HasCgifSaver:      supportedSavers["HasCgifSaver"],
		HasAvifSaver:      supportedSavers["HasAvifSaver"],
		HasJp2kSaver:      supportedSavers["HasJp2kSaver"],
	}

	// Build list of supported savers for the ImageSaverSupport map
	for typeName, supported := range supportedSavers {
		if supported && strings.HasPrefix(typeName, "ImageType") {
			data.SupportedSavers = append(data.SupportedSavers, struct {
				EnumName string
				TypeName string
			}{
				EnumName: typeName,
				TypeName: strings.TrimPrefix(typeName, "ImageType"),
			})
		}
	}

	// Generate all three foreign files
	foreignGoFile := filepath.Join(outputDir, "foreign.go")
	foreignHFile := filepath.Join(outputDir, "foreign.h")
	foreignCFile := filepath.Join(outputDir, "foreign.c")

	// Generate Go file
	if err := templateLoader.GenerateFile("foreign.go.tmpl", foreignGoFile, data); err != nil {
		return fmt.Errorf("failed to generate foreign.go file: %v", err)
	}

	// Generate H file
	if err := templateLoader.GenerateFile("foreign.h.tmpl", foreignHFile, data); err != nil {
		return fmt.Errorf("failed to generate foreign.h file: %v", err)
	}

	// Generate C file
	if err := templateLoader.GenerateFile("foreign.c.tmpl", foreignCFile, data); err != nil {
		return fmt.Errorf("failed to generate foreign.c file: %v", err)
	}

	return nil
}
