package vipsgen

import "embed"

//go:embed templates/*.tmpl
var EmbeddedTemplates embed.FS
