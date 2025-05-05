package binding

// #cgo pkg-config: vips
// #include <vips/vips.h>
// #include <stdlib.h>
//
// typedef struct {
//     char **names;
//     int count;
//     int capacity;
// } OperationList;
//
// typedef struct {
//     char *name;
//     int value;
//     char *nick;
// } EnumValueInfo;
//
// static void* collect_operations(void *object_class, void *a, void *b) {
//     OperationList *list = (OperationList *)a;
//     VipsObjectClass *vobject_class = VIPS_OBJECT_CLASS(object_class);
//
//     if (vobject_class && vobject_class->nickname &&
//         G_TYPE_CHECK_CLASS_TYPE(vobject_class, VIPS_TYPE_OPERATION)) {
//         const char *nickname = vobject_class->nickname;
//
//         if (list->count >= list->capacity) {
//             list->capacity *= 2;
//             list->names = realloc(list->names, list->capacity * sizeof(char*));
//         }
//         list->names[list->count] = strdup(nickname);
//         list->count++;
//     }
//
//     return NULL;
// }
//
// // This function discovers all operations by directly querying GType system
// static char** get_all_operation_names(int *count) {
//     OperationList list = {
//         .names = malloc(200 * sizeof(char*)),
//         .count = 0,
//         .capacity = 200
//     };
//
//     // Get all types derived from VipsOperation
//     GType base_type = VIPS_TYPE_OPERATION;
//     guint n_children = 0;
//     GType *children = g_type_children(base_type, &n_children);
//
//     // Process each child type
//     for (guint i = 0; i < n_children; i++) {
//         // Only process non-abstract types
//         if (!G_TYPE_IS_ABSTRACT(children[i])) {
//             // Get the class to access the nickname
//             VipsObjectClass *class = VIPS_OBJECT_CLASS(g_type_class_ref(children[i]));
//
//             if (class && class->nickname) {
//                 // Check if we can actually instantiate this operation
//                 VipsOperation *op = VIPS_OPERATION(g_object_new(children[i], NULL));
//                 if (op) {
//                     // Expand array if needed
//                     if (list.count >= list.capacity) {
//                         list.capacity *= 2;
//                         list.names = realloc(list.names, list.capacity * sizeof(char*));
//                     }
//
//                     // Add the operation name
//                     list.names[list.count++] = strdup(class->nickname);
//                     g_object_unref(op);
//                 }
//             }
//
//             g_type_class_unref(class);
//         }
//
//         // Some operations might have their own child types
//         guint n_grandchildren = 0;
//         GType *grandchildren = g_type_children(children[i], &n_grandchildren);
//
//         for (guint j = 0; j < n_grandchildren; j++) {
//             if (!G_TYPE_IS_ABSTRACT(grandchildren[j])) {
//                 VipsObjectClass *class = VIPS_OBJECT_CLASS(g_type_class_ref(grandchildren[j]));
//
//                 if (class && class->nickname) {
//                     VipsOperation *op = VIPS_OPERATION(g_object_new(grandchildren[j], NULL));
//                     if (op) {
//                         if (list.count >= list.capacity) {
//                             list.capacity *= 2;
//                             list.names = realloc(list.names, list.capacity * sizeof(char*));
//                         }
//
//                         list.names[list.count++] = strdup(class->nickname);
//                         g_object_unref(op);
//                     }
//                 }
//
//                 g_type_class_unref(class);
//             }
//         }
//
//         g_free(grandchildren);
//     }
//
//     g_free(children);
//
//     *count = list.count;
//     return list.names;
// }
//
// static void free_operation_names(char **names, int count) {
//     for (int i = 0; i < count; i++) {
//         free(names[i]);
//     }
//     free(names);
// }
//
// static EnumValueInfo* get_enum_values(const char *enum_type_name, int *count) {
//     GType type = g_type_from_name(enum_type_name);
//     if (type == 0) {
//         *count = 0;
//         return NULL;
//     }
//
//     // Get the enum class
//     GEnumClass *enum_class = G_ENUM_CLASS(g_type_class_ref(type));
//     if (!enum_class) {
//         *count = 0;
//         return NULL;
//     }
//
//     // Allocate space for values
//     *count = enum_class->n_values;
//     if (*count <= 0 || *count > 100) { // Sanity check
//         g_type_class_unref(enum_class);
//         *count = 0;
//         return NULL;
//     }
//
//     EnumValueInfo *values = malloc(*count * sizeof(EnumValueInfo));
//     if (!values) {
//         g_type_class_unref(enum_class);
//         *count = 0;
//         return NULL;
//     }
//
//     // Copy the values with NULL checks
//     for (int i = 0; i < *count; i++) {
//         // Check for NULL pointers that might cause segfault
//         if (enum_class->values[i].value_name) {
//             values[i].name = strdup(enum_class->values[i].value_name);
//         } else {
//             values[i].name = strdup("UNKNOWN");
//         }
//
//         values[i].value = enum_class->values[i].value;
//
//         if (enum_class->values[i].value_nick) {
//             values[i].nick = strdup(enum_class->values[i].value_nick);
//         } else {
//             values[i].nick = strdup("");
//         }
//     }
//
//     // Unref the class
//     g_type_class_unref(enum_class);
//     return values;
// }
//
// // Check if a type exists
// static int type_exists(const char *type_name) {
//     GType type = g_type_from_name(type_name);
//     return type != 0 ? 1 : 0;
// }
//
// // Free enum value info
// static void free_enum_values(EnumValueInfo *values, int count) {
//     for (int i = 0; i < count; i++) {
//         free(values[i].name);
//         free(values[i].nick);
//     }
//     free(values);
// }
//
// static GObjectClass* get_object_class(void* obj) {
//     return G_OBJECT_GET_CLASS(obj);
// }
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

// Track enum types we've discovered

// List of enum types to look for in libvips
var enumTypeNames []struct {
	CName  string
	GoName string
}

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

// VipsIntrospection provides discovery and analysis of libvips operations
// through reflection of the C library's type system, extracting operation
// metadata, argument details, and supported enum types.
type VipsIntrospection struct {
	discoveredEnumTypes map[string]bool
}

// NewVipsIntrospection creates a new VipsIntrospection instance for analyzing libvips
// operations, initializing the libvips library in the process.
func NewVipsIntrospection() *VipsIntrospection {
	// Initialize libvips
	if C.vips_init(C.CString("vipsgen")) != 0 {
		log.Fatal("Failed to initialize libvips")
	}
	defer C.vips_shutdown()

	return &VipsIntrospection{
		discoveredEnumTypes: make(map[string]bool),
	}
}

// GetAllOperationNames retrieves names of all available operations from libvips
func (v *VipsIntrospection) GetAllOperationNames() []string {
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
func (v *VipsIntrospection) IntrospectOperations() []vipsgen.Operation {
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
func (v *VipsIntrospection) IntrospectOperation(name string) vipsgen.Operation {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	// Create operation
	vop := C.vips_operation_new(cName)
	if vop == nil {
		return vipsgen.Operation{}
	}
	defer C.g_object_unref(C.gpointer(vop))

	// Check if operation has custom config
	var config vipsgen.OperationConfig
	if cfg, ok := vipsgen.OperationConfigs[name]; ok {
		config = cfg
	}

	// Determine category based on operation name patterns
	category := vipsgen.DetermineCategory(name)

	// Get operation details
	operation := vipsgen.Operation{
		Name:        name,
		GoName:      vipsgen.FormatGoFunctionName(name),
		Description: v.getOperationDescription(vop),
		Flags:       int(C.vips_operation_get_flags(vop)),
		Category:    category,
	}

	// Get arguments
	args := v.getOperationArguments(vop)

	// If we have a custom config, apply it
	if config.OptionsParam != "" {
		// Add the options parameter as an optional input
		args = append(args, vipsgen.Argument{
			Name:        config.OptionsParam,
			GoName:      "options",
			Type:        "gchararray",
			GoType:      "string",
			CType:       "const char*",
			Description: "Operation options string",
			Required:    false,
			IsInput:     true,
			IsOutput:    false,
		})
	}

	// Categorize arguments and check for image inputs
	hasImageInput := false
	for _, arg := range args {
		if arg.IsInput {
			if arg.Required {
				operation.RequiredInputs = append(operation.RequiredInputs, arg)
				if arg.Type == "VipsImage" {
					hasImageInput = true
				}
			} else {
				operation.OptionalInputs = append(operation.OptionalInputs, arg)
			}
		} else if arg.IsOutput {
			operation.Outputs = append(operation.Outputs, arg)
			if arg.Type == "VipsImage" {
				operation.HasImageOutput = true
			}
		}
	}
	operation.HasImageInput = hasImageInput
	operation.Arguments = args

	return operation
}

// FilterOperations filters out operations that should be excluded and deduplicates
func (v *VipsIntrospection) FilterOperations(operations []vipsgen.Operation) []vipsgen.Operation {
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
func (v *VipsIntrospection) GetEnumTypes() []vipsgen.EnumTypeInfo {
	var enumTypes []vipsgen.EnumTypeInfo

	for _, typeName := range enumTypeNames {
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
func (v *VipsIntrospection) getEnumType(cName, goName string) (vipsgen.EnumTypeInfo, error) {
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

	for i := 0; i < safeCount; i++ {
		val := valueSlice[i]
		name := C.GoString(val.name)
		nick := C.GoString(val.nick)

		// Process name for Go usage
		goValueName := vipsgen.FormatEnumValueName(goName, name)

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
func (v *VipsIntrospection) AddEnumType(cName, goName string) {
	if _, exists := v.discoveredEnumTypes[cName]; !exists {
		// Add to our enum type list for later processing
		enumTypeNames = append(enumTypeNames, struct {
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

// ExtractImageTypes extracts image type information from operations
func (v *VipsIntrospection) ExtractImageTypes(operations []vipsgen.Operation) []vipsgen.ImageTypeInfo {
	typeMap := make(map[string]bool)

	// Extract from loader/saver operations
	for _, op := range operations {
		if strings.HasSuffix(op.Name, "load") || strings.HasSuffix(op.Name, "load_buffer") {
			typeName := strings.TrimSuffix(strings.TrimSuffix(op.Name, "load_buffer"), "load")
			if typeName != "" {
				typeMap[typeName] = true
			}
		}
	}

	// Create sorted list with predefined order
	imageTypes := []vipsgen.ImageTypeInfo{
		{TypeName: "unknown", EnumName: "ImageTypeUnknown", Order: 0},
	}

	// Common image formats in order
	commonTypes := []string{"gif", "jpeg", "magick", "pdf", "png", "svg", "tiff", "webp", "heif", "bmp", "avif", "jp2k"}

	for i, typeName := range commonTypes {
		if typeMap[typeName] {
			imageTypes = append(imageTypes, vipsgen.ImageTypeInfo{
				TypeName: typeName,
				EnumName: fmt.Sprintf("ImageType%s", strings.Title(typeName)),
				MimeType: v.getMimeType(typeName),
				Order:    i + 1,
			})
		}
	}

	// Add any additional types found
	for typeName := range typeMap {
		found := false
		for _, ct := range commonTypes {
			if ct == typeName {
				found = true
				break
			}
		}
		if !found {
			imageTypes = append(imageTypes, vipsgen.ImageTypeInfo{
				TypeName: typeName,
				EnumName: fmt.Sprintf("ImageType%s", strings.Title(typeName)),
				Order:    len(imageTypes),
			})
		}
	}

	return imageTypes
}

// getMimeType returns the MIME type for a given image format
func (v *VipsIntrospection) getMimeType(typeName string) string {
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

// getOperationDescription gets the description of an operation
func (v *VipsIntrospection) getOperationDescription(op *C.VipsOperation) string {
	obj := (*C.VipsObject)(unsafe.Pointer(op))
	if obj.description != nil {
		return C.GoString(obj.description)
	}
	return ""
}

// getOperationArguments gets the arguments of an operation
func (v *VipsIntrospection) getOperationArguments(op *C.VipsOperation) []vipsgen.Argument {
	var args []vipsgen.Argument

	// Get the GObject class
	gclass := C.get_object_class(unsafe.Pointer(op))

	// Get all properties
	var nProps C.guint
	props := C.g_object_class_list_properties(gclass, &nProps)
	defer C.g_free(C.gpointer(props))

	// Convert to slice for easier handling
	propsSlice := (*[1 << 30]*C.GParamSpec)(unsafe.Pointer(props))[:nProps:nProps]

	// Get VipsArgumentClass for each property
	for i := 0; i < int(nProps); i++ {
		pspec := propsSlice[i]

		// Get argument class
		var argClass *C.VipsArgumentClass
		var argInstance *C.VipsArgumentInstance

		// Convert Go string to C string
		cName := C.CString(C.GoString(pspec.name))
		defer C.free(unsafe.Pointer(cName))

		found := C.vips_object_get_argument(
			(*C.VipsObject)(unsafe.Pointer(op)),
			cName,
			&pspec,
			&argClass,
			&argInstance,
		)

		if found != 0 {
			continue // Skip if not found
		}

		if argClass == nil {
			continue
		}

		// Create argument
		goName := C.GoString(pspec.name)
		arg := vipsgen.Argument{
			Name:        goName,
			GoName:      vipsgen.FormatGoIdentifier(goName),
			Type:        v.getParamType(pspec),
			GoType:      v.getGoType(pspec),
			CType:       v.getCType(pspec),
			Description: v.getParamDescription(pspec),
			Required:    (argClass.flags & C.VIPS_ARGUMENT_REQUIRED) != 0,
			IsInput:     (argClass.flags & C.VIPS_ARGUMENT_INPUT) != 0,
			IsOutput:    (argClass.flags & C.VIPS_ARGUMENT_OUTPUT) != 0,
			Flags:       int(argClass.flags),
		}

		// Check if it's an enum
		if C.g_type_is_a(pspec.value_type, C.G_TYPE_ENUM) != 0 {
			arg.IsEnum = true
			enumTypeName := C.GoString(C.g_type_name(pspec.value_type))
			arg.EnumType = enumTypeName

			// Add this enum type to our list
			goEnumName := vipsgen.GetGoEnumName(enumTypeName)
			v.AddEnumType(enumTypeName, goEnumName)
		}

		args = append(args, arg)
	}

	return args
}

// getParamType returns the type of a parameter
func (v *VipsIntrospection) getParamType(pspec *C.GParamSpec) string {
	gtype := pspec.value_type
	typeName := C.GoString(C.g_type_name(gtype))
	return typeName
}

// getGoType maps VIPS types to Go types
func (v *VipsIntrospection) getGoType(pspec *C.GParamSpec) string {
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
func (v *VipsIntrospection) getCType(pspec *C.GParamSpec) string {
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
func (v *VipsIntrospection) getParamDescription(pspec *C.GParamSpec) string {
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
func (v *VipsIntrospection) DiscoverImageTypes() []vipsgen.ImageTypeInfo {
	// Some image types are always defined, even if not supported
	imageTypes := []vipsgen.ImageTypeInfo{
		{TypeName: "unknown", EnumName: "ImageTypeUnknown", MimeType: "", Order: 0},
	}

	// Standard image formats to check for
	standardTypes := []struct {
		TypeName string
		MimeType string
	}{
		{"gif", "image/gif"},
		{"jpeg", "image/jpeg"},
		{"magick", ""},
		{"pdf", "application/pdf"},
		{"png", "image/png"},
		{"svg", "image/svg+xml"},
		{"tiff", "image/tiff"},
		{"webp", "image/webp"},
		{"heif", "image/heif"},
		{"bmp", "image/bmp"},
		{"avif", "image/avif"},
		{"jp2k", "image/jp2"},
	}

	// Check which image types are supported for loading or saving
	for i, typeInfo := range standardTypes {
		// Format enum name to maintain compatibility with existing code
		enumName := "ImageType" + strings.Title(typeInfo.TypeName)

		// Check if this format is supported by libvips
		loaderName := typeInfo.TypeName + "load"
		cLoader := C.CString(loaderName)
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
				Order:    i + 1, // Start after Unknown (0)
			})
		}
	}

	return imageTypes
}
