package introspection

import (
	"fmt"
	"github.com/cshum/vipsgen/internal/generator"
	"regexp"
	"strings"
)

func (v *Introspection) extractOptionalArgsFromDoc(opName, doc string) (optionalArgs []generator.Argument) {
	if doc == "" {
		return
	}
	fmt.Printf("extractOptionalArgsFromDoc %s\n", opName)

	// Find the optional arguments section
	lines := strings.Split(doc, "\n")
	inOptionalArgs := false
	var optArgLines []string

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check for "Optional arguments:" section header
		if strings.HasPrefix(trimmedLine, "Optional arguments:") {
			inOptionalArgs = true
			fmt.Printf("Found optional args section at line %d\n", i)
			continue
		}

		// If we're in the optional args section
		if inOptionalArgs {
			// Check if we've reached the end of the section
			// This is typically a blank line or paragraph start that isn't a bullet point
			if trimmedLine == "" {
				// Only exit if we've collected at least one argument
				if len(optArgLines) > 0 {
					// Check if the next non-empty line doesn't start with * or @
					// This helps detect the end of the bullet point list
					foundNextNonBullet := false
					for j := i + 1; j < len(lines); j++ {
						nextLine := strings.TrimSpace(lines[j])
						if nextLine != "" {
							if !strings.HasPrefix(nextLine, "*") && !strings.HasPrefix(nextLine, "@") {
								foundNextNonBullet = true
							}
							break
						}
					}

					if foundNextNonBullet {
						fmt.Printf("End of optional args section at line %d\n", i)
						break
					}
				}
			} else if !strings.HasPrefix(trimmedLine, "*") && !strings.HasPrefix(trimmedLine, "@") &&
				!strings.HasPrefix(trimmedLine, "-") && len(optArgLines) > 0 {
				// We've hit a non-bullet point line after collecting some bullet points
				// This is likely the end of the optional args section
				fmt.Printf("End of optional args section at non-bullet line %d: %s\n", i, trimmedLine)
				break
			}

			// Collect bullet points
			if strings.HasPrefix(trimmedLine, "*") {
				optArgLines = append(optArgLines, trimmedLine)
				fmt.Printf("Found optional arg line: %s\n", trimmedLine)
			}
		}
	}

	fmt.Printf("Found %d optional argument lines\n", len(optArgLines))

	// Parse each optional argument line
	for _, line := range optArgLines {
		// Clean up the line
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)

		fmt.Printf("Processing line: %s\n", line)

		// Extract argument name - looking at the pattern in the PNG doc
		// Pattern: @name: %type, description
		var argName string

		// Try different patterns to extract argument name
		patterns := []struct {
			regex *regexp.Regexp
			name  string
		}{
			{regexp.MustCompile(`@([a-zA-Z0-9_]+)\s*:`), "@ prefix"},
			// Fix for backticks regex - using string literals with escaped backticks
			{regexp.MustCompile("`([a-zA-Z0-9_]+)`\\s*:"), "backticks"},
			{regexp.MustCompile(`^([a-zA-Z0-9_]+)\s*:`), "plain name"},
		}

		for _, pattern := range patterns {
			matches := pattern.regex.FindStringSubmatch(line)
			if len(matches) >= 2 {
				argName = matches[1]
				fmt.Printf("Found arg name with %s: %s\n", pattern.name, argName)
				break
			}
		}

		if argName == "" {
			fmt.Println("Could not extract argument name from line:", line)
			continue
		}

		// Extract type and description
		var argType, description string

		// Split by colon to get the type+description part
		parts := strings.SplitN(line, ":", 2)
		if len(parts) >= 2 {
			typePart := strings.TrimSpace(parts[1])
			fmt.Printf("Type part: %s\n", typePart)

			// Try different patterns to extract type
			typePatterns := []struct {
				regex *regexp.Regexp
				group int
				name  string
			}{
				{regexp.MustCompile(`%(g?[a-zA-Z0-9_]+)`), 1, "% prefix"},
				{regexp.MustCompile(`#([A-Za-z0-9_]+)`), 1, "# prefix"},
				// Fix for backticks regex
				{regexp.MustCompile("`([^`]+)`"), 1, "backticks"},
			}

			for _, pattern := range typePatterns {
				matches := pattern.regex.FindStringSubmatch(typePart)
				if len(matches) > pattern.group {
					argType = matches[pattern.group]
					fmt.Printf("Found type with %s: %s\n", pattern.name, argType)
					break
				}
			}

			// If we still don't have a type, try one more approach - take the first word
			if argType == "" {
				// Split by comma and take the first part
				firstPart := strings.Split(typePart, ",")[0]
				// Take the first word as the type
				words := strings.Fields(firstPart)
				if len(words) > 0 {
					argType = words[0]
					fmt.Printf("Extracted type as first word: %s\n", argType)
				}
			}

			// Extract description - everything after the type
			if strings.Contains(typePart, ",") {
				descParts := strings.SplitN(typePart, ",", 2)
				if len(descParts) > 1 {
					description = strings.TrimSpace(descParts[1])
					fmt.Printf("Found description after comma: %s\n", description)
				}
			}
		}

		// Determine Go type based on arg type
		goType := v.determineGoTypeFromDocType(argType)
		baseType := determineBaseTypeFromDoc(argType)
		cType := determineCTypeFromDoc(argType)
		isEnum := v.isEnumType(argType)

		// Special case for png filter
		if argName == "filter" && strings.Contains(opName, "pngsave") {
			argType = "VipsForeignPngFilter"
			goType = "PngFilter"
			baseType = "VipsForeignPngFilter"
			cType = "VipsForeignPngFilter"
			isEnum = true
		}

		fmt.Printf("Final type determination: goType=%s, baseType=%s, cType=%s, isEnum=%v\n",
			goType, baseType, cType, isEnum)

		// Create the argument
		arg := generator.Argument{
			Name:        argName,
			GoName:      FormatGoIdentifier(argName),
			Type:        baseType,
			GoType:      goType,
			CType:       cType,
			Description: description,
			Required:    false,
			IsInput:     true,
			IsOutput:    false,
			IsEnum:      isEnum,
			Flags:       17, // VIPS_ARGUMENT_OPTIONAL | VIPS_ARGUMENT_INPUT
		}

		// Add to optional args list
		optionalArgs = append(optionalArgs, arg)
		fmt.Printf("Added argument: %s (%s)\n", arg.Name, arg.GoType)
	}

	fmt.Printf("Extracted %d optional arguments\n", len(optionalArgs))
	return optionalArgs
}

// determineGoTypeFromDocType maps documentation type hints to Go types
func (v *Introspection) determineGoTypeFromDocType(docType string) string {
	docType = strings.ToLower(docType)

	switch {
	case strings.Contains(docType, "gboolean") ||
		strings.Contains(docType, "true") ||
		strings.Contains(docType, "false"):
		return "bool"
	case strings.Contains(docType, "gint") ||
		strings.Contains(docType, "int"):
		return "int"
	case strings.Contains(docType, "gdouble") ||
		strings.Contains(docType, "double") ||
		strings.Contains(docType, "float"):
		return "float64"
	case strings.Contains(docType, "string") ||
		strings.Contains(docType, "utf8") ||
		strings.Contains(docType, "char"):
		return "string"
	case strings.Contains(docType, "vipsimage"):
		return "*C.VipsImage"
	}

	// Check if it's a known enum type
	if v.isEnumType(docType) {
		return v.GetGoEnumName(docType)
	}

	// Default
	return "interface{}"
}

// determineBaseTypeFromDoc determines a C type name from documentation type hints
func determineBaseTypeFromDoc(docType string) string {
	docType = strings.ToLower(docType)

	// Check if it's an enum type with # prefix (e.g., #VipsForeignPpmFormat)
	if strings.HasPrefix(docType, "#") {
		// Extract the actual type name without the # prefix
		enumTypeName := strings.TrimPrefix(docType, "#")
		return strings.Title(enumTypeName) // Return with proper capitalization
	}

	switch {
	case strings.Contains(docType, "gboolean"):
		return "gboolean"
	case strings.Contains(docType, "gint") || strings.Contains(docType, "int"):
		return "gint"
	case strings.Contains(docType, "gdouble") || strings.Contains(docType, "double"):
		return "gdouble"
	case strings.Contains(docType, "gfloat") || strings.Contains(docType, "float"):
		return "gfloat"
	case strings.Contains(docType, "vipsfailon"):
		return "VipsFailOn"
	case strings.Contains(docType, "vipsalign"):
		return "VipsAlign"
	case strings.Contains(docType, "vipsdirection"):
		return "VipsDirection"
	case strings.Contains(docType, "foreignppm") || strings.Contains(docType, "ppmformat"):
		return "VipsForeignPpmFormat"
	}

	// Default
	return "gpointer"
}

// determineCTypeFromDoc determines a C type from documentation type hints
func determineCTypeFromDoc(docType string) string {
	docType = strings.ToLower(docType)

	switch {
	case strings.Contains(docType, "gboolean"):
		return "gboolean"
	case strings.Contains(docType, "gint") || strings.Contains(docType, "int"):
		return "int"
	case strings.Contains(docType, "gdouble") || strings.Contains(docType, "double"):
		return "double"
	case strings.Contains(docType, "gfloat") || strings.Contains(docType, "float"):
		return "float"
	case strings.Contains(docType, "string") || strings.Contains(docType, "utf8") ||
		strings.Contains(docType, "char"):
		return "const char*"
	case strings.Contains(docType, "vipsimage"):
		return "VipsImage*"
	}

	// Default for enum types and others
	return "int"
}

// determineEnumTypeFromDoc determines the enum type name from doc type
func (v *Introspection) determineEnumTypeFromDoc(docType string) string {
	docType = strings.ToLower(docType)

	// First check against discovered enum types
	for enumName, goName := range v.discoveredEnumTypes {
		lowerEnumName := strings.ToLower(enumName)
		if strings.Contains(docType, lowerEnumName) {
			return goName
		}
	}

	// Fallback to known mappings
	switch {
	case strings.Contains(docType, "vipsfailon"):
		return "FailOn"
	case strings.Contains(docType, "vipsalign"):
		return "Align"
	case strings.Contains(docType, "vipsdirection"):
		return "Direction"
	case strings.Contains(docType, "vipsinteresting"):
		return "Interesting"
	}

	// Default
	return ""
}
