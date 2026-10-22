package generator

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cshum/vipsgen/v2/internal/introspection"
	"github.com/cshum/vipsgen/v2/internal/templates"
)

// TestGenerateEndToEnd runs the real pipeline the command runs: introspect the libvips this
// test links against, build the template data, and render the embedded templates into a
// temporary directory. The unit tests here use stub loaders, so without this nothing
// executes the template functions that produce every method, options struct and C wrapper.
func TestGenerateEndToEnd(t *testing.T) {
	dir := t.TempDir()

	vips := introspection.NewIntrospection(false)
	operations := vips.DiscoverOperations()
	if len(operations) < 100 {
		t.Fatalf("introspection returned %d operations", len(operations))
	}

	data := NewTemplateData(
		vips.GetVipsVersion(),
		operations,
		vips.DiscoverEnumTypes(),
		vips.DiscoverImageTypes(),
		true,
	)

	loader := NewFSTemplateLoader(templates.Templates, GetTemplateFuncMap())
	if err := Generate(loader, data, dir); err != nil {
		t.Fatalf("generate: %v", err)
	}

	for _, name := range []string{
		"image.go", "vips.go", "vips.c", "types.go", "util.go", "callback.go", "connection.go",
		"operations_test.go", "image_test.go", "util_test.go",
	} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s was not generated: %v", name, err)
		}
	}

	// Every Go file has to parse, which is what makes the next regen safe to trust.
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		if _, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.AllErrors); err != nil {
			t.Errorf("%s does not parse: %v", filepath.Base(path), err)
		}
	}

	// The generated methods have to cover the operations that were introspected.
	image, err := os.ReadFile(filepath.Join(dir, "image.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(image)
	missing := 0
	for _, op := range operations {
		if !strings.Contains(source, "func (r *Image) "+op.GoName+"(") &&
			!strings.Contains(source, "func New"+op.GoName+"(") &&
			!strings.Contains(source, "func "+op.GoName+"(") {
			missing++
			if missing < 5 {
				t.Errorf("%s has no method or constructor in image.go", op.Name)
			}
		}
	}
	if missing > 0 {
		t.Errorf("%d of %d operations produced no declaration", missing, len(operations))
	}
}
