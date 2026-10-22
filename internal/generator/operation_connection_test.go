package generator

import (
	"strings"
	"testing"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// TestByteSaverOptions pins that a saver which cannot infer its format names one, and that
// a saver that can is left alone.
func TestByteSaverOptions(t *testing.T) {
	magick := coverageOp("magicksave_buffer", "MagicksaveBuffer")
	magick.OptionalInputs = []introspection.Argument{
		{Name: "format", GoName: "Format", GoType: "string", IsInput: true},
	}
	if got, want := byteSaverOptions(magick), `&MagicksaveBufferOptions{Format: "png"}`; got != want {
		t.Errorf("a saver with a format option should name one:\n got %s\nwant %s", got, want)
	}

	png := coverageOp("pngsave_buffer", "PngsaveBuffer")
	png.OptionalInputs = []introspection.Argument{
		{Name: "compression", GoName: "Compression", GoType: "int", IsInput: true},
	}
	if got := byteSaverOptions(png); got != "nil" {
		t.Errorf("a saver the extension describes needs no format, got %s", got)
	}
}

// TestByteSaversCarryTheirOptions pins the index: the buffer saver for a format is recorded
// with the argument its call needs, so a loader body can call it without knowing why.
func TestByteSaversCarryTheirOptions(t *testing.T) {
	magick := coverageOp("magicksave_buffer", "MagicksaveBuffer")
	magick.HasThisImageInput = true
	magick.HasBufferOutput = true
	magick.OptionalInputs = []introspection.Argument{
		{Name: "format", GoName: "Format", GoType: "string", IsInput: true},
	}

	savers := coverageByteSavers([]introspection.Operation{magick})
	got, ok := savers["magick"]
	if !ok {
		t.Fatal("the buffer saver should be indexed under its format")
	}
	if got.GoName != "MagicksaveBuffer" || got.Options != `&MagicksaveBufferOptions{Format: "png"}` {
		t.Errorf("unexpected saver record: %+v", got)
	}
}

// TestEverySubtestChecksTheOperation pins the guard. The wrapper links the libvips function,
// so a build without the library does not link; but linking and loading can differ, and the
// missing symbol would be a fatal lookup on first call rather than an error to report.
func TestEverySubtestChecksTheOperation(t *testing.T) {
	op := coverageOp("avg", "Avg")
	op.HasThisImageInput = true
	op.HasOneImageOutput = true
	op.RequiredInputs = []introspection.Argument{imageArg("in")}
	op.Arguments = append([]introspection.Argument{imageArg("in")},
		introspection.Argument{Name: "out", GoName: "out", GoType: "float64", IsOutput: true})
	op.RequiredOutputs = []introspection.Argument{
		{Name: "out", GoName: "out", GoType: "float64", IsOutput: true},
	}

	cases := BuildOperationCoverage([]introspection.Operation{op})
	if len(cases) == 0 {
		t.Fatal("the operation should get a subtest")
	}
	want := "\t\tif !HasOperation(\"avg\") {\n\t\t\tt.Skipf(\"this libvips has no avg\")\n\t\t}\n"
	for _, c := range cases {
		if len(c.Body) < len(want) || c.Body[:len(want)] != want {
			t.Errorf("%s does not check the operation first:\n%s", c.Name, c.Body)
		}
	}
}

// TestErrorCaseBody pins the shape of the error subtest and the two reasons an operation does
// not get one.
func TestErrorCaseBody(t *testing.T) {
	op := coverageOp("shrink", "Shrink")
	op.HasThisImageInput = true
	op.HasOneImageOutput = true
	op.Arguments = []introspection.Argument{
		imageArg("in"),
		{Name: "hshrink", GoName: "hshrink", GoType: "float64", IsInput: true},
	}
	op.RequiredInputs = op.Arguments

	body, ok := errorCaseBody(op)
	if !ok {
		t.Fatal("an operation with a required number should get an error subtest")
	}
	want := "\t\timg := coverageImage(t)\n" +
		"\t\terr := img.Shrink(0)\n" +
		"\t\tif err == nil {\n" +
		"\t\t\tif _, evalErr := img.Avg(); evalErr == nil {\n" +
		"\t\t\t\tt.Skipf(\"shrink: accepted arguments libvips should reject\")\n" +
		"\t\t\t}\n" +
		"\t\t}\n"
	if body != want {
		t.Errorf("unexpected body\n got: %q\nwant: %q", body, want)
	}

	enum := coverageOp("rot", "Rot")
	enum.HasThisImageInput = true
	enum.HasOneImageOutput = true
	enum.Arguments = []introspection.Argument{
		imageArg("in"),
		{Name: "angle", GoName: "angle", GoType: "Angle", IsInput: true, IsEnum: true},
	}
	enum.RequiredInputs = enum.Arguments
	enumBody, ok := errorCaseBody(enum)
	if !ok || !strings.Contains(enumBody, "img.Rot(Angle(999))") {
		t.Errorf("an enum should be passed out of range, got %q", enumBody)
	}

	plain := coverageOp("invert", "Invert")
	plain.HasThisImageInput = true
	plain.HasOneImageOutput = true
	plain.Arguments = []introspection.Argument{imageArg("in")}
	plain.RequiredInputs = plain.Arguments
	if _, ok := errorCaseBody(plain); ok {
		t.Error("an operation with only an image has no argument to invalidate")
	}
}
