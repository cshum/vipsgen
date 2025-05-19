package introspection

// #include "introspection.h"
import "C"
import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"unsafe"
)

// EnumTypeInfo holds information about a vips enum type
type EnumTypeInfo struct {
	CName       string // Original C name (e.g. VipsInterpretation)
	GoName      string // Go name (e.g. Interpretation)
	Description string
	Values      []EnumValueInfo
}

// EnumValueInfo holds information about an enum value
type EnumValueInfo struct {
	CName       string // C name
	GoName      string // Go name
	Value       int    // Numeric value
	Description string
}

type enumTypeName struct {
	CName  string
	GoName string
}

// DiscoverEnumTypes retrieves all enum types from libvips
func (v *Introspection) DiscoverEnumTypes() []EnumTypeInfo {
	var enumTypes []EnumTypeInfo

	// First scan for all operations
	var nOps C.int
	opsPtr := C.get_all_operations(&nOps)
	if opsPtr != nil && nOps > 0 {
		defer C.free_operation_info(opsPtr, nOps)
		opsSlice := (*[1 << 30]C.OperationInfo)(unsafe.Pointer(opsPtr))[:nOps:nOps]

		// Loop through each operation to discover enums
		for i := 0; i < int(nOps); i++ {
			cOp := opsSlice[i]
			name := C.GoString(cOp.name)

			// Skip deprecated operations
			if (cOp.flags & C.VIPS_OPERATION_DEPRECATED) != 0 {
				continue
			}
			// Discover enums from this operation
			v.discoverEnumsFromOperation(name)
		}
	}

	// Now process all the discovered enum types
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

	// Debug: Write the parsed GIR to a JSON file
	jsonData, err := json.MarshalIndent(enumTypes, "", "  ")
	if err != nil {
		log.Printf("Warning: failed to marshal Enum Types to JSON: %v", err)
	} else {
		err = os.WriteFile("debug_enums.json", jsonData, 0644)
		if err != nil {
			log.Printf("Warning: failed to write debug_enums.json: %v", err)
		} else {
			log.Println("Wrote introspected Enum Types to debug_enums.json")
		}
	}

	return enumTypes
}

// discoverEnumsFromOperation discover enums from an operation
func (v *Introspection) discoverEnumsFromOperation(opName string) {
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
			enumType := C.GoString(C.g_type_name(pspec.value_type))

			// Add this enum type to our list
			goEnumName := getGoEnumName(enumType)
			v.addEnumType(enumType, goEnumName)
		}

		// Also check for flag types (similar to enums but can be combined as bit flags)
		if C.g_type_is_a(pspec.value_type, C.G_TYPE_FLAGS) != 0 {
			flagTypeName := C.GoString(C.g_type_name(pspec.value_type))

			// Add this flag type to our list
			goFlagName := getGoEnumName(flagTypeName)
			v.addEnumType(flagTypeName, goFlagName)
		}
	}
}

// getEnumType retrieves information about a specific enum type
func (v *Introspection) getEnumType(cName, goName string) (EnumTypeInfo, error) {
	enumType := EnumTypeInfo{
		CName:  cName,
		GoName: goName,
		Values: []EnumValueInfo{},
	}

	// Convert strings to C strings
	cTypeName := C.CString(cName)
	defer C.free(unsafe.Pointer(cTypeName))

	// Determine if this is a flags type
	isFlags := 0
	if C.type_exists(cTypeName) != 0 {
		cType := C.g_type_from_name(cTypeName)
		if C.g_type_is_a(cType, C.G_TYPE_FLAGS) != 0 {
			isFlags = 1
		}
	}

	// Get enum values - check count first to ensure safe allocation
	var count C.int
	values := C.get_enum_or_flag_values(cTypeName, &count, C.int(isFlags))

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
		goValueName := formatEnumValueName(goName, name)

		// For "Foreign" types, we want to strip the "Foreign" prefix from the enum values
		if isForeignType && strings.HasPrefix(goValueName, "Foreign") {
			goValueName = strings.TrimPrefix(goValueName, "Foreign")
		}

		enumType.Values = append(enumType.Values, EnumValueInfo{
			CName:       name,
			GoName:      goValueName,
			Value:       int(val.value),
			Description: nick,
		})
	}

	return enumType, nil
}

// addEnumType adds a newly discovered enum type
func (v *Introspection) addEnumType(cName, goName string) {
	cNameLower := strings.ToLower(cName)
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

func (v *Introspection) getGoEnumName(typeName string) string {
	if name, exists := v.discoveredEnumTypes[strings.ToLower(typeName)]; exists {
		return name
	}
	return getGoEnumName(typeName)
}

// checkEnumValueExists checks if a specific enum value exists
func (v *Introspection) checkEnumValueExists(enumName, valueName string) bool {
	// First check if the enum type exists
	cEnumName := C.CString(enumName)
	defer C.free(unsafe.Pointer(cEnumName))

	if C.type_exists(cEnumName) == 0 {
		return false
	}

	// Determine if this is a flags type
	isFlags := 0
	cType := C.g_type_from_name(cEnumName)
	if C.g_type_is_a(cType, C.G_TYPE_FLAGS) != 0 {
		isFlags = 1
	}

	// Get all enum values
	var count C.int
	values := C.get_enum_or_flag_values(cEnumName, &count, C.int(isFlags))

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
