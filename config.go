package vipsgen

// OperationConfigs defines configuration for specific operations that need special handling
var OperationConfigs = map[string]OperationConfig{
	// Image loading operations
	"jpegload": {
		SkipGen:       false,
		CustomWrapper: true,
		OptionsParam:  "option_string",
	},
	"pngload": {
		SkipGen:       false,
		CustomWrapper: true,
		OptionsParam:  "option_string",
	},
	"webpload": {
		SkipGen:       false,
		CustomWrapper: true,
		OptionsParam:  "option_string",
	},
	"gifload": {
		SkipGen:       false,
		CustomWrapper: true,
		OptionsParam:  "option_string",
	},

	// Save operations - these often need custom handling
	"jpegsave": {
		SkipGen: true, // Use manual implementation
	},
	"pngsave": {
		SkipGen: true, // Use manual implementation
	},
	"webpsave": {
		SkipGen: true, // Use manual implementation
	},

	// Complex operations that need manual implementation
	"composite": {
		SkipGen: true,
	},
	"composite2": {
		SkipGen: true,
	},
}

// ExcludedOperations defines operations that should be excluded from generation
var ExcludedOperations = map[string]bool{
	// Internal or deprecated operations
	"cache":   true,
	"system":  true,
	"version": true,

	// Operations that require special handling
	"sequential": true, // Usually handled via loader options
	"tilecache":  true, // Usually handled internally
}
