package vipsgen

import (
	"fmt"
	"strings"
)

// ImageMethodName converts vipsFooBar to FooBar for method names
func ImageMethodName(name string) string {
	if strings.HasPrefix(name, "vips") {
		return name[4:]
	}
	return name
}

// FormatImageMethodArgs formats arguments for image methods, skipping the first Image argument
func FormatImageMethodArgs(args []Argument) string {
	if len(args) <= 1 {
		return ""
	}
	var params []string
	for i, arg := range args {
		if i == 0 { // Skip the first Image argument
			continue
		}
		params = append(params, fmt.Sprintf("%s %s", arg.GoName, arg.GoType))
	}
	return strings.Join(params, ", ")
}

// FormatGoFunctionName formats an operation name to a Go function name
func FormatGoFunctionName(name string) string {
	// Convert operation names to match existing Go function style
	// e.g., "rotate" -> "vipsRotate", "extract_area" -> "vipsExtractArea"
	parts := strings.Split(name, "_")

	// Convert each part to title case
	for i, part := range parts {
		parts[i] = strings.Title(part)
	}

	// Join with vips prefix
	return "vips" + strings.Join(parts, "")
}

// FormatGoIdentifier formats a name to a Go identifier
func FormatGoIdentifier(name string) string {
	// Handle Go keywords
	switch name {
	case "type", "func", "map", "range", "select", "case", "default":
		return name + "_"
	}

	// Handle special cases
	name = strings.Replace(name, "-", "_", -1)

	return name
}

// GetGoEnumName converts a C enum type name to a Go type name
func GetGoEnumName(cName string) string {
	// Strip "Vips" prefix if present
	if strings.HasPrefix(cName, "Vips") {
		cName = cName[4:]
	}

	return cName
}

// SnakeToCamel converts a snake_case string to CamelCase
func SnakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		parts[i] = strings.Title(parts[i])
	}
	return strings.Join(parts, "")
}

// FormatEnumValueName converts a C enum name to a Go name
func FormatEnumValueName(typeName, valueName string) string {
	if strings.HasPrefix(strings.ToLower(SnakeToCamel(strings.ToLower(valueName))), "vips"+strings.ToLower(typeName)) {
		return GetGoEnumName(SnakeToCamel(strings.ToLower(valueName)))
	}
	return typeName + GetGoEnumName(SnakeToCamel(strings.ToLower(valueName)))
}

// DetermineCategory determines the category of an operation based on its name
func DetermineCategory(name string) string {
	// Use prefixes to determine categories
	if strings.HasPrefix(name, "add") || strings.HasPrefix(name, "subtract") ||
		strings.HasPrefix(name, "multiply") || strings.HasPrefix(name, "divide") ||
		strings.HasPrefix(name, "linear") || strings.HasPrefix(name, "math") ||
		strings.HasPrefix(name, "abs") || strings.HasPrefix(name, "sign") ||
		strings.HasPrefix(name, "round") || strings.HasPrefix(name, "floor") ||
		strings.HasPrefix(name, "ceil") || strings.HasPrefix(name, "max") ||
		strings.HasPrefix(name, "min") || strings.HasPrefix(name, "avg") {
		return "arithmetic"
	}

	if strings.HasPrefix(name, "conv") || strings.HasPrefix(name, "sharpen") ||
		strings.HasPrefix(name, "gaussblur") || strings.HasPrefix(name, "sobel") ||
		strings.HasPrefix(name, "canny") {
		return "convolution"
	}

	if strings.HasPrefix(name, "resize") || strings.HasPrefix(name, "shrink") ||
		strings.HasPrefix(name, "reduce") || strings.HasPrefix(name, "thumbnail") ||
		strings.HasPrefix(name, "affine") || strings.HasPrefix(name, "similarity") {
		return "resample"
	}

	if strings.HasPrefix(name, "colourspace") || strings.HasPrefix(name, "icc") ||
		strings.HasPrefix(name, "Lab2XYZ") || strings.HasPrefix(name, "XYZ2Lab") ||
		strings.HasPrefix(name, "Lab2LCh") || strings.HasPrefix(name, "LCh2Lab") ||
		strings.HasPrefix(name, "sRGB2HSV") || strings.HasPrefix(name, "HSV2sRGB") {
		return "colour"
	}

	if strings.HasSuffix(name, "load") {
		return "foreign_load"
	}

	if strings.HasSuffix(name, "save") || strings.HasSuffix(name, "save_buffer") {
		return "foreign_save"
	}

	if strings.HasPrefix(name, "flip") || strings.HasPrefix(name, "rot") ||
		strings.HasPrefix(name, "extract") || strings.HasPrefix(name, "embed") ||
		strings.HasPrefix(name, "crop") || strings.HasPrefix(name, "join") ||
		strings.HasPrefix(name, "bandjoin") || strings.HasPrefix(name, "bandmean") {
		return "conversion"
	}

	if strings.HasPrefix(name, "hist_") || strings.HasPrefix(name, "stdif") ||
		strings.HasPrefix(name, "percent") || strings.HasPrefix(name, "profile") {
		return "histogram"
	}

	if strings.HasPrefix(name, "morph") || strings.HasPrefix(name, "rank") ||
		strings.HasPrefix(name, "erode") || strings.HasPrefix(name, "dilate") {
		return "morphology"
	}

	if strings.HasPrefix(name, "draw_") || strings.HasPrefix(name, "text") {
		return "draw"
	}

	return "operation" // Default category
}
