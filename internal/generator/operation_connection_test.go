package generator

import (
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
