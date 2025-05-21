package introspection

// #include "introspection.h"
import "C"
import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"
	"unicode"
	"unsafe"
)

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

// snakeToCamel converts a snake_case string to CamelCase
func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		parts[i] = strings.Title(parts[i])
	}
	return strings.Join(parts, "")
}

// formatGoIdentifier formats a name to a go identifier
func formatGoIdentifier(name string) string {
	s := snakeToCamel(formatIdentifier(name))

	// first letter lower case
	if len(s) == 0 {
		return s
	}

	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// formatIdentifier formats a name to an identifier
func formatIdentifier(name string) string {
	if name == "buffer" {
		return "buf"
	}
	// Handle Go keywords
	switch name {
	case "type", "func", "map", "range", "select", "case", "default":
		return name + "_"
	}
	// handle hyphens
	name = strings.Replace(name, "-", "_", -1)
	return name
}

// formatEnumValueName converts a C enum name to a Go name
func formatEnumValueName(typeName, valueName string) string {
	// Convert to CamelCase
	camelValue := snakeToCamel(strings.ToLower(valueName))

	// Check if the value already contains "Vips" + typeName or "VipsForeign" + typeName
	lowerCamelValue := strings.ToLower(camelValue)
	lowerTypeName := strings.ToLower(typeName)

	if strings.HasPrefix(lowerCamelValue, "vips"+lowerTypeName) ||
		strings.HasPrefix(lowerCamelValue, "vipsforeign"+lowerTypeName) {
		return getGoEnumName(camelValue)
	}

	// BandFormatFormat fix
	if typeName == "BandFormat" {
		typeName = "Band"
	}

	// Otherwise, prepend the type name
	return typeName + getGoEnumName(camelValue)
}

// getGoEnumName converts a C enum type name to a Go type name
func getGoEnumName(cName string) string {
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

// formatGoFunctionName formats an operation name to a Go function name
func formatGoFunctionName(name string) string {
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

// addAsterisk adds a * to the end of a type name if not already there
func addAsterisk(typeName string) string {
	if !strings.HasSuffix(typeName, "*") {
		return typeName + "*"
	}
	return typeName
}

// addOutputPointer adds an additional * for output parameters
func addOutputPointer(cType string, isOutput bool) string {
	if isOutput {
		return addAsterisk(cType)
	}
	return cType
}

// Helper function to check type names
func cTypeCheck(gtype C.GType, name string) bool {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	cTypeNamePtr := C.get_type_name(gtype)
	if cTypeNamePtr == nil {
		return false
	}

	cTypeName := C.GoString(cTypeNamePtr)
	return cTypeName == name
}

func debugJson(data any, filename string) {
	// Debug: Write the parsed GIR to a JSON file
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("Warning: failed to marshal Enum Types to JSON: %v", err)
	} else {
		err = os.WriteFile(filename, jsonData, 0644)
		if err != nil {
			log.Printf("Warning: failed to write %s: %v", filename, err)
		} else {
			log.Printf("Wrote introspected Enum Types to %s", filename)
		}
	}
}
