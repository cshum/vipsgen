package introspection

// #include "introspection.h"
import "C"
import (
	"strings"
	"unsafe"
)

// ImageTypeInfo represents information about an image type
type ImageTypeInfo struct {
	TypeName string // Short name (e.g., "gif")
	EnumName string // Go enum name (e.g., "ImageTypeGIF")
	MimeType string // MIME type (e.g., "image/gif")
	Order    int    // Position in the enum
}

// getMimeType returns the MIME type for a given image format
func (v *Introspection) getMimeType(typeName string) string {
	mimeTypes := map[string]string{
		"gif":  "image/gif",
		"jpeg": "image/jpeg",
		"pdf":  "application/pdf",
		"png":  "image/png",
		"svg":  "image/svg+xml",
		"tiff": "image/tiff",
		"webp": "image/webp",
		"heif": "image/heif",
		"bmp":  "image/bmp",
		"avif": "image/avif",
		"jp2k": "image/jp2",
	}

	if mime, ok := mimeTypes[typeName]; ok {
		return mime
	}
	return ""
}

// DiscoverImageTypes discovers supported image types in libvips
func (v *Introspection) DiscoverImageTypes() []ImageTypeInfo {
	// Some image types are always defined, even if not supported
	imageTypes := []ImageTypeInfo{
		{TypeName: "unknown", EnumName: "ImageTypeUnknown", MimeType: "", Order: 0},
	}

	// Standard image formats to check for
	standardTypes := []struct {
		TypeName string
		MimeType string
		OpName   string // Optional custom operation to check
	}{
		{"gif", "image/gif", ""},
		{"jpeg", "image/jpeg", ""},
		{"magick", "", ""},
		{"pdf", "application/pdf", ""},
		{"png", "image/png", ""},
		{"svg", "image/svg+xml", ""},
		{"tiff", "image/tiff", ""},
		{"webp", "image/webp", ""},
		{"heif", "image/heif", ""},
		{"bmp", "image/bmp", ""},
		// The AVIF format needs special handling - see below
		{"jp2k", "image/jp2", ""},
	}

	// Track current order number - start after Unknown (0)
	currentOrder := 1

	// Check which image types are supported for loading or saving
	for _, typeInfo := range standardTypes {
		// Format enum name to maintain compatibility with existing code
		enumName := "ImageType" + strings.Title(typeInfo.TypeName)

		// Check if this format is supported by libvips
		opName := typeInfo.OpName
		if opName == "" {
			opName = typeInfo.TypeName + "load"
		}

		cLoader := C.CString(opName)
		loaderExists := int(C.vips_type_find(cachedCString("VipsOperation"), cLoader)) != 0
		C.free(unsafe.Pointer(cLoader))

		saverName := typeInfo.TypeName + "save"
		cSaver := C.CString(saverName)
		saverExists := int(C.vips_type_find(cachedCString("VipsOperation"), cSaver)) != 0
		C.free(unsafe.Pointer(cSaver))

		// If either loader or saver exists, this format is supported
		if loaderExists || saverExists {
			imageType := ImageTypeInfo{
				TypeName: typeInfo.TypeName,
				EnumName: enumName,
				MimeType: typeInfo.MimeType,
				Order:    currentOrder,
			}
			imageTypes = append(imageTypes, imageType)
			v.discoveredImageTypes[typeInfo.TypeName] = imageType
			currentOrder++
		}
	}

	// Special handling for AVIF - it uses heifsave with AV1 compression
	avifSupported := v.checkOperationExists("heifsave_buffer") &&
		v.checkEnumValueExists("VipsForeignHeifCompression", "VIPS_FOREIGN_HEIF_COMPRESSION_AV1")

	if avifSupported {
		// Add AVIF to the list with its proper order
		imageTypes = append(imageTypes, ImageTypeInfo{
			TypeName: "avif",
			EnumName: "ImageTypeAvif",
			MimeType: "image/avif",
			Order:    currentOrder,
		})
		currentOrder++
	}

	return imageTypes
}

// DiscoverSupportedSavers finds which image savers are supported in current libvips build
func (v *Introspection) DiscoverSupportedSavers() map[string]bool {
	// Check for supported savers by checking if their types are defined
	saverSupport := make(map[string]bool)

	// Define the savers we want to check for
	savers := []struct {
		OpName    string // Operation name to check for
		ImageType string // Corresponding Go ImageType name
		LegacyOp  string // Optional legacy operation name
		ShortName string // Short name without "save_buffer"
	}{
		{"jpegsave_buffer", "ImageTypeJpeg", "", "Jpeg"},
		{"pngsave_buffer", "ImageTypePng", "", "Png"},
		{"webpsave_buffer", "ImageTypeWebp", "", "Webp"},
		{"tiffsave_buffer", "ImageTypeTiff", "", "Tiff"},
		{"heifsave_buffer", "ImageTypeHeif", "", "Heif"},
		{"gifsave_buffer", "ImageTypeGif", "magicksave_buffer", "Gif"},
		{"jp2ksave_buffer", "ImageTypeJp2k", "", "Jp2k"},
	}

	// Check each saver
	for _, saver := range savers {
		hasMainSaver := v.checkOperationExists(saver.OpName)
		hasLegacySaver := saver.LegacyOp != "" && v.checkOperationExists(saver.LegacyOp)

		// Set flag based on correctly formatted saver name
		saverSupport["Has"+saver.ShortName+"Saver"] = hasMainSaver

		// For GIF, also track legacy saver separately
		if saver.OpName == "gifsave_buffer" {
			saverSupport["HasCgifSaver"] = hasMainSaver
			saverSupport["HasLegacyGifSaver"] = hasLegacySaver
		}

		// If either main or legacy saver exists, the format is supported
		if hasMainSaver || hasLegacySaver {
			saverSupport[saver.ImageType] = true
		}
	}

	// AVIF is a special case - it's saved using heifsave with compression=AV1
	avifSupported := v.checkOperationExists("heifsave_buffer") &&
		v.checkEnumValueExists("VipsForeignHeifCompression", "VIPS_FOREIGN_HEIF_COMPRESSION_AV1")

	saverSupport["HasAvifSaver"] = avifSupported
	if avifSupported {
		saverSupport["ImageTypeAvif"] = true
	}

	return saverSupport
}

// DetermineImageTypeStringFromOperation determines the appropriate ImageType
// constant for a given operation name using the discovered image types
func (v *Introspection) DetermineImageTypeStringFromOperation(opName string) string {
	var format string
	if strings.HasSuffix(opName, "load") || strings.HasSuffix(opName, "load_buffer") {
		parts := strings.Split(opName, "load")
		if len(parts) > 1 {
			format = parts[0]
		}
	} else if strings.HasSuffix(opName, "save") || strings.HasSuffix(opName, "save_buffer") {
		parts := strings.Split(opName, "save")
		if len(parts) > 1 {
			format = parts[0]
		}
	}
	// If we found a format, look it up in the available image types
	if format != "" {
		if imageType, exists := v.discoveredImageTypes[format]; exists {
			return imageType.EnumName
		}
	}
	// Default fallback
	return "ImageTypeUnknown"
}

// checkOperationExists checks if a libvips operation exists
func (v *Introspection) checkOperationExists(name string) bool {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	// Try to create the operation - if it succeeds, the operation exists
	vop := C.vips_operation_new(cName)
	if vop == nil {
		return false
	}

	// Clean up and return true
	C.g_object_unref(C.gpointer(vop))
	return true
}
