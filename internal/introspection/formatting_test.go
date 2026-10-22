package introspection

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestFormatIdentifier pins the three renames that keep a libvips argument name from
// colliding with Go: the buffer shorthand, the keywords, and the hyphens.
func TestFormatIdentifier(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"buffer", "buf"},
		{"type", "type_"},
		{"func", "func_"},
		{"map", "map_"},
		{"range", "range_"},
		{"select", "select_"},
		{"case", "case_"},
		{"default", "default_"},
		{"in-out", "in_out"},
		{"filename", "filename"},
	} {
		if got := formatIdentifier(c.in); got != c.want {
			t.Errorf("formatIdentifier(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestFormatGoIdentifier pins the shape a method argument takes: CamelCase from snake case,
// with the first rune lowered, and an empty name left empty.
func TestFormatGoIdentifier(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"in", "in"},
		{"page_height", "pageHeight"},
		{"input_profile", "inputProfile"},
		{"", ""},
	} {
		if got := formatGoIdentifier(c.in); got != c.want {
			t.Errorf("formatGoIdentifier(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestAddAsterisk pins that a pointer type is not given a second asterisk.
func TestAddAsterisk(t *testing.T) {
	if got := addAsterisk("VipsImage"); got != "VipsImage*" {
		t.Errorf("got %q", got)
	}
	if got := addAsterisk("VipsImage*"); got != "VipsImage*" {
		t.Errorf("an existing pointer was doubled: %q", got)
	}
}

// TestAddOutputPointer pins that only an output argument gains the asterisk.
func TestAddOutputPointer(t *testing.T) {
	if got := addOutputPointer("VipsImage", true); got != "VipsImage*" {
		t.Errorf("an output should be a pointer, got %q", got)
	}
	if got := addOutputPointer("VipsImage", false); got != "VipsImage" {
		t.Errorf("an input should not be, got %q", got)
	}
}

// TestDebugJson writes the dump the -debug flag produces, so the flag's own path is
// exercised rather than only the code that skips it.
func TestDebugJson(t *testing.T) {
	path := filepath.Join(t.TempDir(), "debug.json")
	debugJson(map[string]int{"operations": 327}, path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no debug file written: %v", err)
	}
	var payload map[string]int
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("the dump is not JSON: %v", err)
	}
	if payload["operations"] != 327 {
		t.Errorf("payload = %v", payload)
	}
}

// TestHandleSpecialCases pins the capabilities that are inferred rather than discovered:
// GIF saving through ImageMagick, and formats that claim a loader the build does not have.
// Each assertion follows the same check the code makes, so it holds on any build.
func TestHandleSpecialCases(t *testing.T) {
	v := NewIntrospection(false)

	t.Run("gif saver comes from magick", func(t *testing.T) {
		formats := map[string]*ImageTypeInfo{
			"gif": {TypeName: "gif", HasLoader: true, HasSaver: false},
		}
		v.handleSpecialCases(formats)

		wantSaver := v.checkOperationExists("magicksave") || v.checkOperationExists("magicksave_buffer")
		if formats["gif"].HasSaver != wantSaver {
			t.Errorf("gif saver = %v, want %v", formats["gif"].HasSaver, wantSaver)
		}
	})

	t.Run("a loader that does not exist is cleared", func(t *testing.T) {
		formats := map[string]*ImageTypeInfo{
			"notaformat": {TypeName: "notaformat", HasLoader: true},
		}
		v.handleSpecialCases(formats)

		if formats["notaformat"].HasLoader {
			t.Error("a format with no loader operation should not claim one")
		}
	})
}
