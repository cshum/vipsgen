package introspection

// #include "introspection.h"
import "C"
import (
	"log"
	"regexp"
	"sort"
	"strings"
	"unsafe"
)

// ImageTypeInfo represents information about an image type
type ImageTypeInfo struct {
	TypeName  string // Short name (e.g., "gif")
	EnumName  string // Go enum name (e.g., "ImageTypeGIF")
	EnumValue string // String value for the enum (e.g., "gif")
	MimeType  string // MIME type (e.g., "image/gif")
	Order     int    // Position in the enum
	HasLoader bool
	HasSaver  bool
}

// Well-known MIME types for image formats
var knownMimeTypes = map[string]string{
	"gif":       "image/gif",
	"jpeg":      "image/jpeg",
	"jpg":       "image/jpeg",
	"png":       "image/png",
	"webp":      "image/webp",
	"tiff":      "image/tiff",
	"tif":       "image/tiff",
	"bmp":       "image/bmp",
	"svg":       "image/svg+xml",
	"heif":      "image/heif",
	"heic":      "image/heic",
	"avif":      "image/avif",
	"pdf":       "application/pdf",
	"jp2":       "image/jp2",
	"jp2k":      "image/jp2",
	"j2k":       "image/jp2",
	"jxl":       "image/jxl",
	"exr":       "image/x-exr",
	"openexr":   "image/openexr",
	"fits":      "image/fits",
	"ppm":       "image/x-portable-pixmap",
	"pgm":       "image/x-portable-graymap",
	"pbm":       "image/x-portable-bitmap",
	"pnm":       "image/x-portable-anymap",
	"dz":        "image/x-deepzoom",
	"vips":      "image/vnd.libvips",
	"mat":       "application/x-matlab-data",
	"nii":       "application/x-nifti",
	"nifti":     "application/x-nifti",
	"analyze":   "application/x-analyze",
	"openslide": "application/x-openslide",
	"matlab":    "application/x-matlab-data",
	"csv":       "text/csv",
	"matrix":    "application/x-matrix",
	"rad":       "image/rad",
	"raw":       "image/raw",
}

// Base image types that should always be included in the enum
var baseImageTypes = []string{
	"jpeg", "gif", "png", "webp", "heif", "svg",
	"tiff", "jp2k", "avif", "pdf", "bmp", "magick",
}

// Regular expressions to match load/save operations
var loadRegex = regexp.MustCompile(`^([a-zA-Z0-9_]+?)(?:load|load_buffer|load_source)(?:_(.+))?$`)
var saveRegex = regexp.MustCompile(`^([a-zA-Z0-9_]+?)(?:save|save_buffer|save_target)(?:_(.+))?$`)

// DiscoverImageTypes discovers supported image types by scanning available operations
func (v *Introspection) DiscoverImageTypes() []ImageTypeInfo {
	log.Printf("Discovering image types from available operations...")

	// Always include unknown type first
	imageTypes := []ImageTypeInfo{
		{TypeName: "unknown", EnumName: "ImageTypeUnknown", EnumValue: "unknown", MimeType: "", Order: 0},
	}

	// Initialize discoveredFormats with base types
	discoveredFormats := make(map[string]*ImageTypeInfo)
	for _, typeName := range baseImageTypes {
		discoveredFormats[typeName] = &ImageTypeInfo{
			TypeName:  typeName,
			EnumName:  "ImageType" + strings.Title(typeName),
			EnumValue: typeName,
			MimeType:  getMimeType(typeName),
			HasLoader: false,
			HasSaver:  false,
		}
	}

	// Get all operations
	var nOps C.int
	opsPtr := C.get_all_operations(&nOps)
	if opsPtr == nil || nOps == 0 {
		log.Printf("Warning: No operations found, using base types only")
		// Still return base types even if no operations found
		v.addBaseTypesToResult(discoveredFormats, &imageTypes)
		return imageTypes
	}
	defer C.free_operation_info(opsPtr, nOps)

	opsSlice := (*[1 << 30]C.OperationInfo)(unsafe.Pointer(opsPtr))[:nOps:nOps]

	// First pass: collect all operations by format
	formatOperations := make(map[string]map[string]bool) // format -> operation -> exists

	for i := 0; i < int(nOps); i++ {
		cOp := opsSlice[i]
		opName := C.GoString(cOp.name)

		// Skip deprecated operations
		if (cOp.flags & C.VIPS_OPERATION_DEPRECATED) != 0 {
			continue
		}

		var formatName string

		// Check if this is a loader operation
		if matches := loadRegex.FindStringSubmatch(opName); matches != nil {
			formatName = normalizeFormatName(matches[1])
			if formatName != "" {
				if formatOperations[formatName] == nil {
					formatOperations[formatName] = make(map[string]bool)
				}
				formatOperations[formatName][opName] = true
			}
		}

		// Check if this is a saver operation
		if matches := saveRegex.FindStringSubmatch(opName); matches != nil {
			formatName = normalizeFormatName(matches[1])
			if formatName != "" {
				if formatOperations[formatName] == nil {
					formatOperations[formatName] = make(map[string]bool)
				}
				formatOperations[formatName][opName] = true
			}
		}
	}

	// Second pass: analyze what operations each format has
	for formatName, operations := range formatOperations {
		hasLoader := false
		hasSaver := false

		// Check for loader variants
		loaderVariants := []string{
			formatName + "load",
			formatName + "load_buffer",
			formatName + "load_source",
		}
		var foundLoaders []string
		for _, variant := range loaderVariants {
			if operations[variant] {
				hasLoader = true
				foundLoaders = append(foundLoaders, variant)
			}
		}

		// Check for saver variants
		saverVariants := []string{
			formatName + "save",
			formatName + "save_buffer",
			formatName + "save_target",
		}
		var foundSavers []string
		for _, variant := range saverVariants {
			if operations[variant] {
				hasSaver = true
				foundSavers = append(foundSavers, variant)
			}
		}

		// Update existing format or add new one
		if existing, exists := discoveredFormats[formatName]; exists {
			// Update base type with discovered capabilities
			existing.HasLoader = hasLoader
			existing.HasSaver = hasSaver
			if v.isDebug && (hasLoader || hasSaver) {
				log.Printf("Image type %s: loaders=%v, savers=%v", formatName, foundLoaders, foundSavers)
			}
		} else if hasLoader || hasSaver {
			// Add new discovered format not in base types
			discoveredFormats[formatName] = &ImageTypeInfo{
				TypeName:  formatName,
				EnumName:  "ImageType" + strings.Title(formatName),
				EnumValue: formatName,
				MimeType:  getMimeType(formatName),
				HasLoader: hasLoader,
				HasSaver:  hasSaver,
			}
			if v.isDebug {
				log.Printf("Image type %s: loaders=%v, savers=%v", formatName, foundLoaders, foundSavers)
			}
		}
	}

	// Handle special cases and post-processing
	v.handleSpecialCases(discoveredFormats)

	// Add all discovered formats to result
	v.addBaseTypesToResult(discoveredFormats, &imageTypes)

	if v.isDebug {
		debugJson(imageTypes, "debug_image_types.json")
	}

	log.Printf("Discovered %d image types total", len(imageTypes))
	return imageTypes
}

// addBaseTypesToResult adds all discovered formats to the result with proper ordering
func (v *Introspection) addBaseTypesToResult(discoveredFormats map[string]*ImageTypeInfo, imageTypes *[]ImageTypeInfo) {
	// Add base types first
	currentOrder := 1
	for _, typeName := range baseImageTypes {
		if format, exists := discoveredFormats[typeName]; exists {
			format.Order = currentOrder
			*imageTypes = append(*imageTypes, *format)
			v.discoveredImageTypes[typeName] = *format
			currentOrder++

			// Remove from map so we don't add it again
			delete(discoveredFormats, typeName)
		}
	}

	// Add any remaining discovered formats (not in base types) in alphabetical order
	var extraFormats []string
	for formatName := range discoveredFormats {
		extraFormats = append(extraFormats, formatName)
	}
	sort.Strings(extraFormats)

	for _, formatName := range extraFormats {
		format := discoveredFormats[formatName]
		format.Order = currentOrder
		*imageTypes = append(*imageTypes, *format)
		v.discoveredImageTypes[formatName] = *format
		currentOrder++
	}
}

// normalizeFormatName handles special cases and aliases in format names
func normalizeFormatName(formatName string) string {
	// Convert to lowercase for consistency first
	formatName = strings.ToLower(formatName)

	// Filter out invalid/non-format operations
	if formatName == "" ||
		strings.HasSuffix(formatName, "_") ||
		formatName == "profile" ||
		formatName == "foreign" ||
		formatName == "icc" ||
		formatName == "colourspace" ||
		formatName == "colorspace" {
		return "" // Skip these operations
	}

	// Handle common aliases and special cases
	switch formatName {
	case "jpg":
		return "jpeg"
	case "tif":
		return "tiff"
	case "j2k", "jp2":
		return "jp2k"
	case "openslide":
		return "openslide"
	case "matlab":
		return "mat"
	case "nifti":
		return "nii"
	}

	// Remove common prefixes that don't indicate format
	if strings.HasPrefix(formatName, "foreign") {
		return strings.TrimPrefix(formatName, "foreign")
	}

	return formatName
}

// getMimeType returns the MIME type for a given format name
func getMimeType(formatName string) string {
	if mimeType, exists := knownMimeTypes[strings.ToLower(formatName)]; exists {
		return mimeType
	}
	return ""
}

// handleSpecialCases handles special processing for certain image formats
func (v *Introspection) handleSpecialCases(discoveredFormats map[string]*ImageTypeInfo) {
	// Handle AVIF as a special case of HEIF with AV1 compression
	if heifFormat, hasHeif := discoveredFormats["heif"]; hasHeif {
		if v.checkEnumValueExists("VipsForeignHeifCompression", "VIPS_FOREIGN_HEIF_COMPRESSION_AV1") {
			if avifFormat, hasAvif := discoveredFormats["avif"]; hasAvif {
				// Update existing AVIF format with HEIF capabilities
				avifFormat.HasLoader = heifFormat.HasLoader
				avifFormat.HasSaver = heifFormat.HasSaver
				log.Printf("Updated AVIF support based on HEIF with AV1 compression")
			} else {
				// Create AVIF format based on HEIF
				discoveredFormats["avif"] = &ImageTypeInfo{
					TypeName:  "avif",
					EnumName:  "ImageTypeAvif",
					EnumValue: "avif",
					MimeType:  "image/avif",
					HasLoader: heifFormat.HasLoader,
					HasSaver:  heifFormat.HasSaver,
				}
				log.Printf("Added AVIF support based on HEIF with AV1 compression")
			}
		}
	}

	// Handle legacy GIF support via ImageMagick
	if gifFormat, hasGif := discoveredFormats["gif"]; hasGif && !gifFormat.HasSaver {
		if v.checkOperationExists("magicksave") || v.checkOperationExists("magicksave_buffer") {
			gifFormat.HasSaver = true
			log.Printf("Added legacy GIF save support via ImageMagick")
		}
	}

	// Verify format support by double-checking operations exist
	for formatName, format := range discoveredFormats {
		if format.HasLoader {
			loaderExists := v.checkOperationExists(formatName+"load") ||
				v.checkOperationExists(formatName+"load_buffer") ||
				v.checkOperationExists(formatName+"load_source")
			if !loaderExists {
				format.HasLoader = false
				if v.isDebug {
					log.Printf("Warning: Loader for %s not actually available", formatName)
				}
			}
		}

		if format.HasSaver {
			saverExists := v.checkOperationExists(formatName+"save") ||
				v.checkOperationExists(formatName+"save_buffer") ||
				v.checkOperationExists(formatName+"save_target")
			if !saverExists {
				format.HasSaver = false
				if v.isDebug {
					log.Printf("Warning: Saver for %s not actually available", formatName)
				}
			}
		}
	}
}

// determineImageTypeStringFromOperation determines the appropriate ImageType
// constant for a given operation name using the discovered image types
func (v *Introspection) determineImageTypeStringFromOperation(opName string) string {
	// Extract format from operation name
	var format string

	// Try different operation name patterns
	patterns := []string{
		`^([a-zA-Z0-9_]+?)load`,
		`^([a-zA-Z0-9_]+?)save`,
	}

	for _, pattern := range patterns {
		if matched, err := regexp.MatchString(pattern, opName); err == nil && matched {
			re := regexp.MustCompile(pattern)
			if matches := re.FindStringSubmatch(opName); len(matches) > 1 {
				format = normalizeFormatName(matches[1])
				break
			}
		}
	}

	// If we found a format, look it up in the discovered image types
	if format != "" {
		if imageType, exists := v.discoveredImageTypes[format]; exists {
			return imageType.EnumName
		}
	}

	// Default fallback
	return "ImageTypeUnknown"
}
