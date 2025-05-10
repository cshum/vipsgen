package introspection

// #cgo pkg-config: vips
// #include "introspection.h"
import "C"
import (
	"fmt"
	"github.com/cshum/vipsgen"
	"log"
	"sort"
	"strings"
	"sync"
	"unsafe"
)

type enumTypeName struct {
	CName  string
	GoName string
}

// Define a more base list of common enum types to look for in libvips
var baseEnumTypeNames []enumTypeName

var cStringsCache sync.Map

// cachedCString returns a cached C string
func cachedCString(str string) *C.char {
	if cstr, ok := cStringsCache.Load(str); ok {
		return cstr.(*C.char)
	}
	cstr := C.CString(str)
	cStringsCache.Store(str, cstr)
	return cstr
}

// Introspection provides discovery and analysis of libvips operations
// through reflection of the C library's type system, extracting operation
// metadata, argument details, and supported enum types.
type Introspection struct {
	discoveredEnumTypes map[string]bool
	enumTypeNames       []enumTypeName
}

// NewIntrospection creates a new Introspection instance for analyzing libvips
// operations, initializing the libvips library in the process.
func NewIntrospection() *Introspection {
	// Initialize libvips
	if C.vips_init(C.CString("vipsgen")) != 0 {
		log.Fatal("Failed to initialize libvips")
	}
	defer C.vips_shutdown()

	// Initialize map with known enum types
	discoveredTypes := make(map[string]bool)
	for _, enum := range baseEnumTypeNames {
		discoveredTypes[enum.CName] = true
	}

	return &Introspection{
		discoveredEnumTypes: discoveredTypes,
		enumTypeNames:       baseEnumTypeNames,
	}
}

// GetAllOperationNames retrieves names of all available operations from libvips
func (v *Introspection) GetAllOperationNames() []string {
	var count C.int
	cNames := C.get_all_operation_names(&count)
	defer C.free_operation_names(cNames, count)

	names := make([]string, int(count))

	// Convert C array to Go slice
	cNamesSlice := (*[1 << 30]*C.char)(unsafe.Pointer(cNames))[:count:count]
	for i, cName := range cNamesSlice {
		names[i] = C.GoString(cName)
	}

	sort.Strings(names)
	return names
}

// IntrospectOperations discovers and analyzes all libvips operations
func (v *Introspection) IntrospectOperations() []vipsgen.Operation {
	var operations []vipsgen.Operation

	// Get all operation names
	operationNames := v.GetAllOperationNames()

	for _, name := range operationNames {
		op := v.IntrospectOperation(name)
		if op.Name != "" && len(op.RequiredInputs) > 0 { // Only include operations with required inputs
			operations = append(operations, op)
		}
	}

	return operations
}

// IntrospectOperation analyzes a single libvips operation
func (v *Introspection) IntrospectOperation(name string) vipsgen.Operation {
	// Create operation
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	vop := C.vips_operation_new(cName)
	if vop == nil {
		return vipsgen.Operation{}
	}
	defer C.g_object_unref(C.gpointer(vop))

	// Determine category based on operation name patterns
	category := vipsgen.DetermineCategory(name)

	// Get operation details
	operation := vipsgen.Operation{
		Name:     name,
		GoName:   vipsgen.FormatGoFunctionName(name),
		Flags:    int(C.vips_operation_get_flags(vop)),
		Category: category,
	}

	return operation
}

// FilterOperations filters out operations that should be excluded and deduplicates
func (v *Introspection) FilterOperations(operations []vipsgen.Operation) []vipsgen.Operation {
	// Filter out excluded operations and deduplicate by Go function name
	seenFunctions := make(map[string]bool)
	var filteredOps []vipsgen.Operation
	for _, op := range operations {
		if vipsgen.ExcludedOperations[op.Name] {
			fmt.Printf("Excluding operation: %s\n", op.Name)
			continue
		}

		if config, ok := vipsgen.OperationConfigs[op.Name]; ok && config.SkipGen {
			fmt.Printf("Skipping operation (configured): %s\n", op.Name)
			continue
		}

		// Check for duplicate Go function names
		if seenFunctions[op.GoName] {
			fmt.Printf("Skipping duplicate function: %s (from operation: %s)\n", op.GoName, op.Name)
			continue
		}
		seenFunctions[op.GoName] = true

		filteredOps = append(filteredOps, op)
	}

	return filteredOps
}

// GetEnumTypes retrieves all enum types from libvips
func (v *Introspection) GetEnumTypes() []vipsgen.EnumTypeInfo {
	var enumTypes []vipsgen.EnumTypeInfo

	for _, typeName := range v.enumTypeNames {
		// Check if the enum type exists first
		cTypeName := C.CString(typeName.CName)
		exists := C.type_exists(cTypeName)
		C.free(unsafe.Pointer(cTypeName))

		if exists == 0 {
			fmt.Printf("Warning: enum type %s not found in libvips\n", typeName.CName)
			continue
		}

		// Try to get the enum values
		enumInfo, err := v.getEnumType(typeName.CName, typeName.GoName)
		if err != nil {
			fmt.Printf("Warning: couldn't process enum type %s: %v\n", typeName.CName, err)
			continue
		}

		// Add successfully processed enum
		enumTypes = append(enumTypes, enumInfo)
	}

	return enumTypes
}

// getEnumType retrieves information about a specific enum type
func (v *Introspection) getEnumType(cName, goName string) (vipsgen.EnumTypeInfo, error) {
	enumType := vipsgen.EnumTypeInfo{
		CName:  cName,
		GoName: goName,
		Values: []vipsgen.EnumValueInfo{},
	}

	// Convert strings to C strings
	cTypeName := C.CString(cName)
	defer C.free(unsafe.Pointer(cTypeName))

	// Get enum values - check count first to ensure safe allocation
	var count C.int
	values := C.get_enum_values(cTypeName, &count)

	if values == nil || count <= 0 {
		return enumType, fmt.Errorf("no values found for enum type %s", cName)
	}

	// Process enum values safely
	defer C.free_enum_values(values, count)
	valueSlice := (*[1 << 30]C.EnumValueInfo)(unsafe.Pointer(values))

	// Only use the valid range
	safeCount := int(count)
	if safeCount > 100 { // Sanity check to avoid insane values
		safeCount = 100
	}

	// Check if we need to handle "VipsForeign" prefixes
	isForeignType := strings.HasPrefix(cName, "VipsForeign")

	for i := 0; i < safeCount; i++ {
		val := valueSlice[i]
		name := C.GoString(val.name)
		nick := C.GoString(val.nick)

		// Process name for Go usage
		goValueName := vipsgen.FormatEnumValueName(goName, name)

		// For "Foreign" types, we want to strip the "Foreign" prefix from the enum values
		if isForeignType && strings.HasPrefix(goValueName, "Foreign") {
			goValueName = strings.TrimPrefix(goValueName, "Foreign")
		}

		enumType.Values = append(enumType.Values, vipsgen.EnumValueInfo{
			CName:       name,
			GoName:      goValueName,
			Value:       int(val.value),
			Description: nick,
		})
	}

	return enumType, nil
}

// AddEnumType adds a newly discovered enum type
func (v *Introspection) AddEnumType(cName, goName string) {
	if _, exists := v.discoveredEnumTypes[cName]; !exists {
		// Process the Go name to remove "Foreign" prefix if needed
		// For example, change "ForeignTiffPredictor" to "TiffPredictor"
		if strings.HasPrefix(goName, "Foreign") {
			goName = strings.TrimPrefix(goName, "Foreign")
		}

		// Add to our enum type list for later processing
		v.enumTypeNames = append(v.enumTypeNames, struct {
			CName  string
			GoName string
		}{
			CName:  cName,
			GoName: goName,
		})
		v.discoveredEnumTypes[cName] = true
		fmt.Printf("Discovered enum type: %s -> %s\n", cName, goName)
	}
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

// getParamType returns the type of a parameter
func (v *Introspection) getParamType(pspec *C.GParamSpec) string {
	gtype := pspec.value_type
	typeName := C.GoString(C.g_type_name(gtype))
	return typeName
}

// getGoType maps VIPS types to Go types
func (v *Introspection) getGoType(pspec *C.GParamSpec) string {
	gtype := pspec.value_type
	typeName := C.GoString(C.g_type_name(gtype))

	// Map VIPS types to Go types
	switch typeName {
	case "VipsImage":
		return "*C.VipsImage"
	case "gboolean":
		return "bool"
	case "gint":
		return "int"
	case "gdouble":
		return "float64"
	case "gchararray":
		return "string"
	case "VipsArrayInt":
		return "[]int"
	case "VipsArrayDouble":
		return "[]float64"
	case "VipsArrayImage":
		return "[]*C.VipsImage"
	case "VipsBlob":
		return "[]byte"
	case "VipsInterpolate":
		return "*C.VipsInterpolate"
	case "VipsSource":
		return "*C.VipsSource"
	case "VipsTarget":
		return "*C.VipsTarget"
	default:
		// Check if it's an enum type
		if C.g_type_is_a(gtype, C.G_TYPE_ENUM) != 0 {
			// Convert VipsBlah to Blah for the Go enum type name
			// Common enums to map
			return vipsgen.GetGoEnumName(typeName)
		}
		// Check if it's a flags type
		if C.g_type_is_a(gtype, C.G_TYPE_FLAGS) != 0 {
			return "int"
		}
		return "interface{}"
	}
}

// getCType maps VIPS types to C types
func (v *Introspection) getCType(pspec *C.GParamSpec) string {
	gtype := pspec.value_type
	typeName := C.GoString(C.g_type_name(gtype))

	// Map VIPS types to C types
	switch typeName {
	case "VipsImage":
		return "VipsImage*"
	case "gboolean":
		return "int" // Use int for boolean in C wrapper
	case "gint":
		return "int"
	case "gdouble":
		return "double"
	case "gchararray":
		return "const char*"
	case "VipsArrayInt":
		return "VipsArrayInt*"
	case "VipsArrayDouble":
		return "VipsArrayDouble*"
	case "VipsArrayImage":
		return "VipsArrayImage*"
	case "VipsBlob":
		return "VipsBlob*"
	case "VipsInterpolate":
		return "VipsInterpolate*"
	case "VipsSource":
		return "VipsSource*"
	case "VipsTarget":
		return "VipsTarget*"
	default:
		// Check if it's an enum type
		if C.g_type_is_a(gtype, C.G_TYPE_ENUM) != 0 {
			return typeName
		}
		return "void*"
	}
}

// getParamDescription gets the description of a parameter
func (v *Introspection) getParamDescription(pspec *C.GParamSpec) string {
	// Try nick first, then blurb
	if pspec.flags&C.G_PARAM_READABLE != 0 && pspec._nick != nil {
		return C.GoString(pspec._nick)
	}
	if pspec._blurb != nil {
		return C.GoString(pspec._blurb)
	}
	return ""
}

// DiscoverImageTypes discovers supported image types in libvips
func (v *Introspection) DiscoverImageTypes() []vipsgen.ImageTypeInfo {
	// Some image types are always defined, even if not supported
	imageTypes := []vipsgen.ImageTypeInfo{
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
			imageTypes = append(imageTypes, vipsgen.ImageTypeInfo{
				TypeName: typeInfo.TypeName,
				EnumName: enumName,
				MimeType: typeInfo.MimeType,
				Order:    currentOrder,
			})
			currentOrder++
		}
	}

	// Special handling for AVIF - it uses heifsave with AV1 compression
	avifSupported := v.checkOperationExists("heifsave_buffer") &&
		v.checkEnumValueExists("VipsForeignHeifCompression", "VIPS_FOREIGN_HEIF_COMPRESSION_AV1")

	if avifSupported {
		// Add AVIF to the list with its proper order
		imageTypes = append(imageTypes, vipsgen.ImageTypeInfo{
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

// checkEnumValueExists checks if a specific enum value exists
func (v *Introspection) checkEnumValueExists(enumName, valueName string) bool {
	// First check if the enum type exists
	cEnumName := C.CString(enumName)
	defer C.free(unsafe.Pointer(cEnumName))

	if C.type_exists(cEnumName) == 0 {
		return false
	}

	// Get all enum values
	var count C.int
	values := C.get_enum_values(cEnumName, &count)

	if values == nil || count <= 0 {
		return false
	}

	defer C.free_enum_values(values, count)
	valueSlice := (*[1 << 30]C.EnumValueInfo)(unsafe.Pointer(values))

	// Look for the specific value
	safeCount := int(count)
	if safeCount > 100 { // Sanity check
		safeCount = 100
	}

	for i := 0; i < safeCount; i++ {
		val := valueSlice[i]
		name := C.GoString(val.name)

		if name == valueName {
			return true
		}
	}

	return false
}
