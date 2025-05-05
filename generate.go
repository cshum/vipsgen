package vipsgen

import "github.com/cshum/vipsgen/templateloader"

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
