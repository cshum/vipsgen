package generator

import (
	"github.com/cshum/vipsgen/internal/introspection"
	"strings"
)

// NewTemplateData creates a new TemplateData structure with all needed information
func NewTemplateData(
	operations []introspection.Operation,
	enumTypes []introspection.EnumTypeInfo,
	imageTypes []introspection.ImageTypeInfo,
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
		OperationConfigs: introspection.OperationConfigs,
		EnumTypes:        enumTypes,
		ImageTypes:       imageTypes,
		EnumTypeMap:      enumTypeMap,

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
