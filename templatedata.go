package vipsgen

import "strings"

// NewTemplateData creates a new TemplateData structure with all needed information
func NewTemplateData(
	operations []Operation,
	enumTypes []EnumTypeInfo,
	imageTypes []ImageTypeInfo,
	supportedSavers map[string]bool,
) *TemplateData {
	// Create map for quick enum type lookups
	enumTypeMap := make(map[string]bool)
	for _, op := range operations {
		for _, arg := range op.RequiredInputs {
			if arg.IsEnum {
				enumTypeMap[arg.GoType] = true
			}
		}
	}

	// Filter operations that have Image as first argument and return Image
	var imageOps []Operation
	for _, op := range operations {
		if len(op.RequiredInputs) > 0 && op.RequiredInputs[0].Type == "VipsImage" && op.HasImageOutput {
			imageOps = append(imageOps, op)
		}
	}

	// Build list of supported savers for templates
	var saversList []SupportedSaverInfo
	for typeName, supported := range supportedSavers {
		if supported && strings.HasPrefix(typeName, "ImageType") {
			saversList = append(saversList, SupportedSaverInfo{
				EnumName: typeName,
				TypeName: strings.TrimPrefix(typeName, "ImageType"),
			})
		}
	}

	return &TemplateData{
		Operations:       operations,
		OperationConfigs: OperationConfigs,
		EnumTypes:        enumTypes,
		ImageTypes:       imageTypes,
		EnumTypeMap:      enumTypeMap,
		ImageOperations:  imageOps,

		// Specific saver flags for templates that expect them
		HasJpegSaver:      supportedSavers["HasJpegSaver"],
		HasPngSaver:       supportedSavers["HasPngSaver"],
		HasWebpSaver:      supportedSavers["HasWebpSaver"],
		HasTiffSaver:      supportedSavers["HasTiffSaver"],
		HasHeifSaver:      supportedSavers["HasHeifSaver"],
		HasLegacyGifSaver: supportedSavers["HasLegacyGifSaver"],
		HasCgifSaver:      supportedSavers["HasCgifSaver"],
		HasAvifSaver:      supportedSavers["HasAvifSaver"],
		HasJp2kSaver:      supportedSavers["HasJp2kSaver"],
		SupportedSavers:   saversList,
	}
}
