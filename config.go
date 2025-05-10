package vipsgen

// OperationConfigs defines configuration for specific operations that need special handling
var OperationConfigs = map[string]OperationConfig{}

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
