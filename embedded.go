package vipsgen

import "embed"

//go:embed templates/*.tmpl Vips-8.0.gir
var EmbeddedTemplates embed.FS
