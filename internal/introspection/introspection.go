package introspection

// #cgo pkg-config: vips
// #include "introspection.h"
import "C"
import (
	"encoding/json"
	"fmt"
	"github.com/cshum/vipsgen/internal/generator"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unsafe"
)

type enumTypeName struct {
	CName  string
	GoName string
}

// VipsFunctionInfo holds information needed to generate a wrapper function
type VipsFunctionInfo struct {
	Name           string
	CIdentifier    string
	ReturnType     string
	Category       string
	HasOutParam    bool
	OutParamIndex  int
	HasVarArgs     bool
	Description    string
	OriginalDoc    string
	Params         []VipsParamInfo
	RequiredParams []VipsParamInfo // Non-optional params
	OptionalParams []VipsParamInfo // Optional params that can be passed as named args
}

// VipsParamInfo represents a parameter for a vips function
type VipsParamInfo struct {
	Name       string
	CType      string
	IsOutput   bool
	IsOptional bool
	IsArray    bool
	ArrayType  string
	IsVarArgs  bool
}

// DebugInfo stores debug information during parsing
type DebugInfo struct {
	ProcessedFunctions         int
	FoundFunctionNames         []string
	MissingCIdentifierIncluded int
}

// Define a more base list of common enum types to look for in libvips
var baseEnumTypeNames []enumTypeName

var excludedEnumTypeNames = map[string]bool{"VipsForeignPngFilter": true}

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
	discoveredEnumTypes map[string]string
	enumTypeNames       []enumTypeName
	// Parsed function info
	functionInfo []VipsFunctionInfo
	// Debug info from parsing
	debugInfo            *DebugInfo
	discoveredImageTypes map[string]generator.ImageTypeInfo
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
	discoveredTypes := make(map[string]string)
	for _, enum := range baseEnumTypeNames {
		discoveredTypes[enum.CName] = enum.GoName
	}

	return &Introspection{
		discoveredEnumTypes:  discoveredTypes,
		discoveredImageTypes: map[string]generator.ImageTypeInfo{},
		enumTypeNames:        baseEnumTypeNames,
	}
}

// DiscoverOperations uses GObject introspection to discover all available operations
func (v *Introspection) DiscoverOperations() []generator.Operation {
	var nOps C.int
	opsPtr := C.get_all_operations(&nOps)
	if opsPtr == nil || nOps == 0 {
		return nil
	}
	defer C.free_operation_info(opsPtr, nOps)

	// Convert C array to Go slice
	opsSlice := (*[1 << 30]C.OperationInfo)(unsafe.Pointer(opsPtr))[:nOps:nOps]
	var operations []generator.Operation

	for i := 0; i < int(nOps); i++ {
		op := opsSlice[i]
		name := C.GoString(op.name)

		// Skip deprecated operations
		if (op.flags & C.VIPS_OPERATION_DEPRECATED) != 0 {
			continue
		}

		// Get detailed operation information
		opName := C.CString(name)
		details := C.get_operation_details(opName)
		C.free(unsafe.Pointer(opName))

		// Create the Go operation structure
		goOp := generator.Operation{
			Name:               name,
			GoName:             FormatGoFunctionName(name),
			Description:        C.GoString(op.description),
			HasImageInput:      int(details.has_image_input) != 0,
			HasImageOutput:     int(details.has_image_output) != 0,
			HasOneImageOutput:  int(details.has_one_image_output) != 0,
			HasBufferInput:     int(details.has_buffer_input) != 0,
			HasBufferOutput:    int(details.has_buffer_output) != 0,
			HasArrayImageInput: int(details.has_array_image_input) != 0,
			Category:           C.GoString(details.category),
			ImageTypeString:    v.DetermineImageTypeStringFromOperation(name),
		}

		if details.category != nil {
			C.free(unsafe.Pointer(details.category))
		}

		v.DiscoverEnumsFromOperation(name)

		// Get all arguments
		args, err := v.GetOperationArguments(name)
		if err == nil {

			// Categorize arguments
			for _, arg := range args {
				if arg.Type == "VipsForeignFlags" {
					continue
				}
				if arg.IsInput {
					if arg.Required {
						goOp.Arguments = append(goOp.Arguments, arg)
						goOp.RequiredInputs = append(goOp.RequiredInputs, arg)
					} else {
						goOp.OptionalInputs = append(goOp.OptionalInputs, arg)
					}
				} else if arg.IsOutput {
					goOp.Outputs = append(goOp.Outputs, arg)
					goOp.Arguments = append(goOp.Arguments, arg)
				}
			}
		}

		operations = append(operations, goOp)
	}

	// Debug: Write the parsed GIR to a JSON file
	jsonData, err := json.MarshalIndent(operations, "", "  ")
	if err != nil {
		log.Printf("Warning: failed to marshal operations to JSON: %v", err)
	} else {
		err = os.WriteFile("debug_operations.json", jsonData, 0644)
		if err != nil {
			log.Printf("Warning: failed to write debug_operations.json: %v", err)
		} else {
			log.Println("Wrote introspected operations to debug_operations.json")
		}
	}

	return operations
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

// DiscoverEnumsFromOperation discover enums from an operation
func (v *Introspection) DiscoverEnumsFromOperation(opName string) {
	// Create operation instance
	cName := C.CString(opName)
	defer C.free(unsafe.Pointer(cName))

	op := C.vips_operation_new(cName)
	if op == nil {
		return
	}
	defer C.g_object_unref(C.gpointer(op))

	// Get the GObject class
	gclass := C.get_object_class(unsafe.Pointer(op))

	// Get all properties
	var nProps C.guint
	props := C.g_object_class_list_properties(gclass, &nProps)
	defer C.g_free(C.gpointer(props))

	// Convert to slice for easier handling
	propsSlice := (*[1 << 30]*C.GParamSpec)(unsafe.Pointer(props))[:nProps:nProps]

	for i := 0; i < int(nProps); i++ {
		pspec := propsSlice[i]

		// Skip properties with NULL name (safety check)
		if pspec.name == nil {
			continue
		}

		// Get argument class and instance
		var argClass *C.VipsArgumentClass
		var argInstance *C.VipsArgumentInstance

		// Convert Go string to C string
		goName := C.GoString(pspec.name)
		cArgName := C.CString(goName)

		found := C.vips_object_get_argument(
			(*C.VipsObject)(unsafe.Pointer(op)),
			cArgName,
			&pspec,
			&argClass,
			&argInstance,
		)
		C.free(unsafe.Pointer(cArgName))

		if found != 0 || argClass == nil {
			continue
		}

		// Check if it's an enum
		if C.g_type_is_a(pspec.value_type, C.G_TYPE_ENUM) != 0 {
			enumTypeName := C.GoString(C.g_type_name(pspec.value_type))

			// Add this enum type to our list
			goEnumName := GetGoEnumName(enumTypeName)
			v.AddEnumType(enumTypeName, goEnumName)
		}

	}
}

// FilterOperations filters operations based on availability in the current libvips installation,
// excluded operations list, and deduplicates by Go function name
func (v *Introspection) FilterOperations(operations []generator.Operation) []generator.Operation {
	// Filter out excluded operations and deduplicate by Go function name
	seenFunctions := make(map[string]bool)
	var filteredOps []generator.Operation
	var notAvailableCount, excludedCount, duplicateCount int

	for _, op := range operations {
		// Check if operation can be instantiated in current libvips
		if !v.checkOperationExists(op.Name) {
			notAvailableCount++
			continue
		}
		if strings.Contains(op.Name, "_source") || strings.Contains(op.Name, "_target") ||
			strings.Contains(op.Name, "_mime") {
			fmt.Printf("Excluding operation: %s \n", op.Name)
			excludedCount++
			continue
		}

		// Check if operation is explicitly excluded
		if generator.ExcludedOperations[op.Name] {
			fmt.Printf("Excluding operation: %s (in ExcludedOperations list)\n", op.Name)
			excludedCount++
			continue
		}

		// Check if operation is excluded by config
		if config, ok := generator.OperationConfigs[op.Name]; ok && config.SkipGen {
			fmt.Printf("Skipping operation (configured in OperationConfigs): %s\n", op.Name)
			excludedCount++
			continue
		}

		// Check for duplicate Go function names
		if seenFunctions[op.GoName] {
			fmt.Printf("Skipping duplicate function: %s (from operation: %s)\n", op.GoName, op.Name)
			duplicateCount++
			continue
		}
		seenFunctions[op.GoName] = true

		filteredOps = append(filteredOps, op)
	}

	fmt.Printf("Filtered operations: %d excluded, %d duplicates, %d remaining\n",
		excludedCount, duplicateCount, len(filteredOps))

	return filteredOps
}

// GetEnumTypes retrieves all enum types from libvips
func (v *Introspection) GetEnumTypes() []generator.EnumTypeInfo {
	var enumTypes []generator.EnumTypeInfo

	for _, typeName := range v.enumTypeNames {
		if excludedEnumTypeNames[typeName.CName] {
			fmt.Printf("Excluded enum type: %s -> %s\n", typeName.CName, typeName.GoName)
			continue
		}
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
func (v *Introspection) getEnumType(cName, goName string) (generator.EnumTypeInfo, error) {

	enumType := generator.EnumTypeInfo{
		CName:  cName,
		GoName: goName,
		Values: []generator.EnumValueInfo{},
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
		goValueName := FormatEnumValueName(goName, name)

		// For "Foreign" types, we want to strip the "Foreign" prefix from the enum values
		if isForeignType && strings.HasPrefix(goValueName, "Foreign") {
			goValueName = strings.TrimPrefix(goValueName, "Foreign")
		}

		enumType.Values = append(enumType.Values, generator.EnumValueInfo{
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
	cNameLower := strings.ToLower(cName)
	if excludedEnumTypeNames[cName] {
		fmt.Printf("Excluded enum type: %s -> %s\n", cName, goName)
		return
	}
	if _, exists := v.discoveredEnumTypes[cNameLower]; !exists {
		// Add to our enum type list for later processing
		v.enumTypeNames = append(v.enumTypeNames, struct {
			CName  string
			GoName string
		}{
			CName:  cName,
			GoName: goName,
		})
		v.discoveredEnumTypes[cNameLower] = goName
		fmt.Printf("Discovered enum type: %s -> %s\n", cName, goName)
	}
}

func (v *Introspection) GetGoEnumName(typeName string) string {
	if name, exists := v.discoveredEnumTypes[strings.ToLower(typeName)]; exists {
		return name
	}
	return GetGoEnumName(typeName)
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
func (v *Introspection) DiscoverImageTypes() []generator.ImageTypeInfo {
	// Some image types are always defined, even if not supported
	imageTypes := []generator.ImageTypeInfo{
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
			imageType := generator.ImageTypeInfo{
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
		imageTypes = append(imageTypes, generator.ImageTypeInfo{
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

func (v *Introspection) isEnumType(cType string) bool {
	return v.discoveredEnumTypes[strings.ToLower(cType)] != ""
}

// determineFlags calculates the flags for an argument
func determineFlags(isOutput bool, isRequired bool) int {
	if isOutput && isRequired {
		return 35 // VIPS_ARGUMENT_REQUIRED | VIPS_ARGUMENT_OUTPUT
	} else if isOutput && !isRequired {
		return 33 // VIPS_ARGUMENT_OPTIONAL | VIPS_ARGUMENT_OUTPUT
	} else if !isOutput && isRequired {
		return 19 // VIPS_ARGUMENT_REQUIRED | VIPS_ARGUMENT_INPUT
	} else {
		return 17 // VIPS_ARGUMENT_OPTIONAL | VIPS_ARGUMENT_INPUT
	}
}

// FormatGoIdentifier formats a name to a go identifier
func FormatGoIdentifier(name string) string {
	s := generator.SnakeToCamel(FormatIdentifier(name))

	// first letter lower case
	if len(s) == 0 {
		return s
	}

	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// FormatIdentifier formats a name to an identifier
func FormatIdentifier(name string) string {
	// Handle Go keywords
	switch name {
	case "type", "func", "map", "range", "select", "case", "default":
		return name + "_"
	}

	// Handle special cases
	name = strings.Replace(name, "-", "_", -1)

	return name
}

// FormatEnumValueName converts a C enum name to a Go name
func FormatEnumValueName(typeName, valueName string) string {
	// Convert to CamelCase
	camelValue := generator.SnakeToCamel(strings.ToLower(valueName))

	// Check if the value already contains "Vips" + typeName or "VipsForeign" + typeName
	lowerCamelValue := strings.ToLower(camelValue)
	lowerTypeName := strings.ToLower(typeName)

	if strings.HasPrefix(lowerCamelValue, "vips"+lowerTypeName) ||
		strings.HasPrefix(lowerCamelValue, "vipsforeign"+lowerTypeName) {
		return GetGoEnumName(camelValue)
	}

	// Otherwise, prepend the type name
	return typeName + GetGoEnumName(camelValue)
}

// GetGoEnumName converts a C enum type name to a Go type name
func GetGoEnumName(cName string) string {
	// Strip "Vips" prefix if present
	if strings.HasPrefix(cName, "Vips") {
		cName = cName[4:]
	}

	// Also strip "Foreign" prefix if present in both the original name
	// and after removing "Vips" prefix
	if strings.HasPrefix(cName, "Foreign") {
		cName = cName[7:]
	}

	return cName
}

// FormatGoFunctionName formats an operation name to a Go function name
func FormatGoFunctionName(name string) string {
	// Convert operation names to match existing Go function style
	// e.g., "extract_area" -> "ExtractArea"
	parts := strings.Split(name, "_")

	// Convert each part to title case
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[0:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}
