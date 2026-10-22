package generator

import (
	"strings"
	"testing"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// TestFixtureKind pins which loaders get a testdata fixture and how each takes the file:
// by name, as a Source, or as a slice - the last one being invisible to callArguments,
// which drops buffer arguments.
func TestFixtureKind(t *testing.T) {
	file := introspection.Operation{
		Name: "pdfload", GoName: "Pdfload", HasOneImageOutput: true,
		RequiredInputs: []introspection.Argument{
			{Name: "filename", GoName: "filename", GoType: "string", IsInput: true},
		},
	}
	if variant, name, ok := fixtureKind(file); !ok || variant != "file" || name != "sample.pdf" {
		t.Errorf("a filename loader should read sample.pdf by name, got %q %q %v", variant, name, ok)
	}

	source := introspection.Operation{
		Name: "svgload_source", GoName: "SvgloadSource", HasOneImageOutput: true,
		Arguments: []introspection.Argument{
			{Name: "source", GoName: "source", GoType: "*C.VipsSourceCustom", IsInput: true, IsSource: true},
		},
		RequiredInputs: []introspection.Argument{
			{Name: "source", GoName: "source", GoType: "*C.VipsSourceCustom", IsInput: true, IsSource: true},
		},
	}
	if variant, name, ok := fixtureKind(source); !ok || variant != "source" || name != "sample.svg" {
		t.Errorf("a source loader should read sample.svg through a Source, got %q %q %v", variant, name, ok)
	}

	buffer := introspection.Operation{
		Name: "pdfload_buffer", GoName: "PdfloadBuffer", HasOneImageOutput: true, HasBufferInput: true,
		RequiredInputs: []introspection.Argument{
			{Name: "buf", GoName: "buf", GoType: "[]byte", IsInput: true, IsBuffer: true},
			{Name: "len", GoName: "len", GoType: "int", IsInput: true},
		},
	}
	if variant, name, ok := fixtureKind(buffer); !ok || variant != "buffer" || name != "sample.pdf" {
		t.Errorf("a buffer loader should read sample.pdf from bytes, got %q %q %v", variant, name, ok)
	}

	unlisted := introspection.Operation{
		Name: "jpegload", GoName: "Jpegload", HasOneImageOutput: true,
		RequiredInputs: []introspection.Argument{
			{Name: "filename", GoName: "filename", GoType: "string", IsInput: true},
		},
	}
	if _, _, ok := fixtureKind(unlisted); ok {
		t.Error("a format with a saver needs no fixture")
	}
}

// TestFixtureLoaderBodySnapshots pins the two shapes: an image opened from the path, and
// bytes read out of it first.
func TestFixtureLoaderBodySnapshots(t *testing.T) {
	load := introspection.Operation{
		Name: "pdfload", GoName: "Pdfload", HasOneImageOutput: true,
		RequiredInputs: []introspection.Argument{
			{Name: "filename", GoName: "filename", GoType: "string", IsInput: true},
		},
	}

	tail := "\t\tif err != nil {\n" +
		"\t\t\tt.Fatalf(\"wrapper reported an error: %v\", err)\n" +
		"\t\t}\n" +
		"\t\tdefer loaded.Close()\n" +
		"\t\tif loaded.Width() <= 0 || loaded.Height() <= 0 {\n" +
		"\t\t\tt.Fatalf(\"pdfload: opened an empty image %dx%d\", loaded.Width(), loaded.Height())\n" +
		"\t\t}\n" +
		"\t\tif _, err := loaded.Avg(); err != nil {\n" +
		"\t\t\tt.Fatalf(\"pdfload: the fixture did not decode: %v\", err)\n" +
		"\t\t}\n"

	wantFile := "\t\tpath := testdataPath(t, \"sample.pdf\")\n" +
		"\t\tloaded, err := NewPdfload(path)\n" +
		tail

	if got := fixtureLoaderBody(load, "file", "sample.pdf", nilCoverageOptions()); got != wantFile {
		t.Errorf("unexpected file body\n got: %q\nwant: %q", got, wantFile)
	}

	bufferLoad := load
	bufferLoad.Name = "pdfload_buffer"
	bufferLoad.GoName = "PdfloadBuffer"
	bufferLoad.HasBufferInput = true

	wantBuffer := "\t\tpath := testdataPath(t, \"sample.pdf\")\n" +
		"\t\tdata, err := os.ReadFile(path)\n" +
		"\t\tif err != nil {\n" +
		"\t\t\tt.Fatalf(\"reading fixture: %v\", err)\n" +
		"\t\t}\n" +
		"\t\tloaded, err := NewPdfloadBuffer(data)\n" +
		strings.Replace(tail, "pdfload:", "pdfload_buffer:", 1)

	wantBuffer = strings.ReplaceAll(wantBuffer, "pdfload:", "pdfload_buffer:")
	if got := fixtureLoaderBody(bufferLoad, "buffer", "sample.pdf", nilCoverageOptions()); got != wantBuffer {
		t.Errorf("unexpected buffer body\n got: %q\nwant: %q", got, wantBuffer)
	}

	if !strings.Contains(fixtureLoaderBody(load, "source", "sample.pdf", nilCoverageOptions()), "coverageSource(t, data)") {
		t.Error("a source loader should hand the bytes to coverageSource")
	}
}
