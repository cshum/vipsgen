package generator

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/cshum/vipsgen/v2/internal/introspection"
	"github.com/cshum/vipsgen/v2/internal/templates"
)

// TestCgoOptionType pins how an option field reaches C: the four Go types with a direct
// counterpart, an enum by its C name, and everything else by its C type or as int.
func TestCgoOptionType(t *testing.T) {
	for _, c := range []struct {
		opt  introspection.Argument
		want string
	}{
		{introspection.Argument{GoType: "string"}, "*C.char"},
		{introspection.Argument{GoType: "bool"}, "C.gboolean"},
		{introspection.Argument{GoType: "float64"}, "C.double"},
		{introspection.Argument{GoType: "float32"}, "C.float"},
		{introspection.Argument{GoType: "int"}, "C.int"},
		{introspection.Argument{GoType: "int64"}, "C.int"},
		{introspection.Argument{GoType: "int", IsEnum: true, CType: "VipsIntent"}, "C.VipsIntent"},
		{introspection.Argument{GoType: "int", IsFlags: true, CType: "VipsForeignFlags"}, "C.VipsForeignFlags"},
		{introspection.Argument{GoType: "VipsBlob", CType: "VipsBlob"}, "C.VipsBlob"},
	} {
		if got := cgoOptionType(c.opt); got != c.want {
			t.Errorf("%+v: got %q, want %q", c.opt, got, c.want)
		}
	}
}

// TestDescribeOptionDefaultValue pins what a doc comment says about a default: nothing when
// the zero value already is the default, otherwise the value, by enum name where one exists.
func TestDescribeOptionDefaultValue(t *testing.T) {
	enumValueNames["TestIntent"] = map[int]string{1: "TestIntentOne"}

	for _, c := range []struct {
		name string
		opt  introspection.Argument
		want string
	}{
		{"zero int is the default", introspection.Argument{GoType: "int", DefaultValue: 0}, ""},
		{"no default at all", introspection.Argument{GoType: "string"}, ""},
		{"true", introspection.Argument{GoType: "bool", DefaultValue: true}, "default true"},
		{"int", introspection.Argument{GoType: "int", DefaultValue: 75}, "default 75"},
		{"int64", introspection.Argument{GoType: "int64", DefaultValue: int64(3)}, "default 3"},
		{"float", introspection.Argument{GoType: "float64", DefaultValue: 1.5}, "default 1.5"},
		{
			"enum by name",
			introspection.Argument{GoType: "int", IsEnum: true, EnumType: "TestIntent", DefaultValue: 1},
			"default TestIntentOne",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := describeOptionDefaultValue(c.opt); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

// TestExtractEmbeddedFS writes the templates vipsgen embeds to a directory, which is what
// the -extract flag does for anyone editing them.
func TestExtractEmbeddedFS(t *testing.T) {
	dir := t.TempDir()
	if err := ExtractEmbeddedFS(templates.Templates, dir); err != nil {
		t.Fatalf("extract: %v", err)
	}

	written, err := os.ReadFile(filepath.Join(dir, "image.go.tmpl"))
	if err != nil {
		t.Fatalf("image.go.tmpl was not extracted: %v", err)
	}
	if len(written) == 0 {
		t.Error("image.go.tmpl is empty")
	}
	if _, err := os.Stat(filepath.Join(dir, "vips.c.tmpl")); err != nil {
		t.Errorf("vips.c.tmpl was not extracted: %v", err)
	}
}

// TestGenerateFunctionCall pins the C call a generated method makes: the image first, then
// every argument that is not an output and not the image itself.
func TestGenerateFunctionCall(t *testing.T) {
	op := coverageOp("resize", "Resize")
	op.Arguments = []introspection.Argument{
		{Name: "in", GoName: "in", GoType: "*C.VipsImage", IsInput: true},
		{Name: "scale", GoName: "scale", GoType: "float64", IsInput: true},
		{Name: "out", GoName: "out", GoType: "*C.VipsImage", IsOutput: true},
	}

	if got, want := generateFunctionCall(op), "Resize(r.image, scale)"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestExtractEmbeddedFSErrors pins the two ways extraction fails: a destination that cannot
// be created, and a template set it cannot read.
func TestExtractEmbeddedFSErrors(t *testing.T) {
	t.Run("destination is a file", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "not-a-dir")
		if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := ExtractEmbeddedFS(templates.Templates, file); err == nil {
			t.Error("expected an error when the destination is a file")
		}
	})

	t.Run("templates cannot be read", func(t *testing.T) {
		if err := ExtractEmbeddedFS(emptyFS{}, t.TempDir()); err == nil {
			t.Error("expected an error when the template set is empty")
		}
	})
}

// emptyFS has no files, which is what an extracted set looks like when nothing was embedded.
type emptyFS struct{}

func (emptyFS) Open(string) (fs.File, error) { return nil, fs.ErrNotExist }

// TestNewOSTemplateLoader pins the path a contributor uses when editing templates: a loader
// over a directory, which lists and renders what is there, and refuses a directory that is
// not.
func TestNewOSTemplateLoader(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "greeting.go.tmpl"), []byte("// {{.Name}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	loader, err := NewOSTemplateLoader(dir, GetTemplateFuncMap())
	if err != nil {
		t.Fatalf("loader: %v", err)
	}

	files, err := loader.ListFiles()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(files) != 1 || files[0] != "greeting.go.tmpl" {
		t.Errorf("list = %v", files)
	}

	tmpl, err := loader.LoadTemplate("greeting.go.tmpl")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if tmpl.Name() != "greeting.go.tmpl" {
		t.Errorf("template name = %q", tmpl.Name())
	}

	if _, err := NewOSTemplateLoader(filepath.Join(dir, "missing"), GetTemplateFuncMap()); err == nil {
		t.Error("a missing directory should be an error")
	}
}
