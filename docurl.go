package vipsgen

import (
	"fmt"
	"strings"
)

var categoryToDocMap = map[string]string{
	"foreign": "VipsForeignSave",
}

func generateDocUrl(funcName string, sourceCategory string) string {
	// Look up the documentation category
	docCategory, exists := categoryToDocMap[sourceCategory]
	if !exists {
		// Default to the source category if no mapping exists
		docCategory = "libvips-" + sourceCategory
	}

	funcName = strings.ReplaceAll(funcName, "_", "-")

	// For most categories, the URL format seems to be:
	return fmt.Sprintf("https://www.libvips.org/API/current/%s.html#vips-%s", docCategory, funcName)
}
