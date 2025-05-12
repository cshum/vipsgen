package vipsgen

// OperationConfigs defines configuration for specific operations that need special handling
var OperationConfigs = map[string]OperationConfig{}

// ExcludedOperations defines operations that should be excluded from generation
var ExcludedOperations = map[string]bool{
	// Exclude custom implementations
	"jpegsave_buffer":   true,
	"jpegsave_mime":     true,
	"jpegsave":          true,
	"heifsave_buffer":   true,
	"heifsave_mime":     true,
	"heifsave":          true,
	"tiffsave_buffer":   true,
	"tiffsave_mime":     true,
	"tiffsave":          true,
	"webpsave_buffer":   true,
	"webpsave_mime":     true,
	"webpsave":          true,
	"gifsave_buffer":    true,
	"gifsave_mime":      true,
	"gifsave":           true,
	"jp2ksave_buffer":   true,
	"jp2ksave_mime":     true,
	"jp2ksave":          true,
	"pngsave_buffer":    true,
	"pngsave_mime":      true,
	"pngsave":           true,
	"magicksave_buffer": true,
	"magicksave_mime":   true,
	"magicksave":        true,
}
