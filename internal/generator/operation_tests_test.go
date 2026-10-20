package generator

import (
	"strings"
	"testing"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// coverageOp builds the shape of an operation the coverage selector reads.
func coverageOp(name, goName string) introspection.Operation {
	return introspection.Operation{Name: name, GoName: goName}
}

func imageArg(name string) introspection.Argument {
	return introspection.Argument{Name: name, GoName: name, GoType: "*C.VipsImage", IsInput: true, IsImage: true}
}

func TestCoverableOperationSelection(t *testing.T) {
	imageOnly := coverageOp("invert", "Invert")
	imageOnly.HasThisImageInput = true
	imageOnly.HasOneImageOutput = true
	imageOnly.RequiredInputs = []introspection.Argument{imageArg("in")}

	withFilename := coverageOp("jpegload", "Jpegload")
	withFilename.HasImageOutput = true
	withFilename.RequiredInputs = []introspection.Argument{
		{Name: "filename", GoName: "filename", GoType: "string", IsInput: true},
	}

	withWidth := coverageOp("black", "Black")
	withWidth.HasImageOutput = true
	withWidth.RequiredInputs = []introspection.Argument{
		{Name: "width", GoName: "width", GoType: "int", IsInput: true},
	}

	voidOp := coverageOp("sink", "Sink")
	voidOp.HasThisImageInput = true
	voidOp.RequiredInputs = []introspection.Argument{imageArg("in")}

	bufferOp := coverageOp("jpegsave_buffer", "JpegsaveBuffer")
	bufferOp.HasThisImageInput = true
	bufferOp.HasBufferOutput = true
	bufferOp.RequiredInputs = []introspection.Argument{imageArg("in")}

	tests := []struct {
		name string
		op   introspection.Operation
		want bool
	}{
		{"images only with an image result", imageOnly, true},
		{"a required filename", withFilename, false},
		{"a required dimension", withWidth, false},
		{"no result to inspect", voidOp, false},
		{"a buffer result", bufferOp, true},
	}

	for _, test := range tests {
		if got := coverableOperation(test.op); got != test.want {
			t.Errorf("%s: coverableOperation = %v, want %v", test.name, got, test.want)
		}
	}
}

// TestCoverageBodySnapshots pins the shape of a generated subtest: the call, the
// one-band retry for the operations that refuse a three-band image, and the
// assertion. The bodies are Go source, so the test compares them literally.
func TestCoverageBodySnapshots(t *testing.T) {
	circle := coverageOp("circle", "Circle")
	circle.HasThisImageInput = true
	circle.HasOneImageOutput = true
	circle.RequiredInputs = []introspection.Argument{imageArg("in")}
	circle.OptionalInputs = []introspection.Argument{
		{Name: "radius", GoName: "Radius", GoType: "int"},
	}

	wantBranch := "" +
		"\t\timg := coverageImage(t)\n" +
		"\t\terr := img.Circle(nil)\n" +
		"\t\tif err != nil {\n" +
		"\t\t\tfirstErr := err\n" +
		"\t\t\timg = coverageMonoImage(t)\n" +
		"\t\t\terr = img.Circle(nil)\n" +
		"\t\t\tif err != nil {\n" +
		"\t\t\t\tt.Fatalf(\"circle: wrapper failed for a three-band and a one-band image: %v / %v\", firstErr, err)\n" +
		"\t\t\t}\n" +
		"\t\t}\n" +
		"\t\tif img.Width() <= 0 || img.Height() <= 0 {\n" +
		"\t\t\tt.Fatalf(\"circle: produced an empty image %dx%d\", img.Width(), img.Height())\n" +
		"\t\t}\n"

	if got := coverageBody(circle); got != wantBranch {
		t.Errorf("unexpected branch body\n got: %q\nwant: %q", got, wantBranch)
	}

	save := coverageOp("pngsave_buffer", "PngsaveBuffer")
	save.HasThisImageInput = true
	save.HasBufferOutput = true
	save.RequiredInputs = []introspection.Argument{imageArg("in")}
	save.OptionalInputs = []introspection.Argument{
		{Name: "compression", GoName: "Compression", GoType: "int"},
	}

	wantBuffer := "" +
		"\t\timg := coverageImage(t)\n" +
		"\t\tbuf, err := img.PngsaveBuffer(nil)\n" +
		"\t\tif err != nil {\n" +
		"\t\t\tfirstErr := err\n" +
		"\t\t\timg = coverageMonoImage(t)\n" +
		"\t\t\tbuf, err = img.PngsaveBuffer(nil)\n" +
		"\t\t\tif err != nil {\n" +
		"\t\t\t\tt.Fatalf(\"pngsave_buffer: wrapper failed for a three-band and a one-band image: %v / %v\", firstErr, err)\n" +
		"\t\t\t}\n" +
		"\t\t}\n" +
		"\t\tif len(buf) == 0 {\n" +
		"\t\t\tt.Fatalf(\"pngsave_buffer: wrote an empty buffer: %d bytes\", len(buf))\n" +
		"\t\t}\n"

	if got := coverageBody(save); got != wantBuffer {
		t.Errorf("unexpected buffer body\n got: %q\nwant: %q", got, wantBuffer)
	}

	creator := coverageOp("identity", "Identity")
	creator.HasOneImageOutput = true
	creator.OptionalInputs = []introspection.Argument{
		{Name: "bands", GoName: "Bands", GoType: "int"},
	}

	wantCreator := "" +
		"\t\tout, err := NewIdentity(nil)\n" +
		"\t\tif err != nil {\n" +
		"\t\t\tt.Fatalf(\"wrapper reported an error: %v\", err)\n" +
		"\t\t}\n" +
		"\t\tdefer out.Close()\n" +
		"\t\tif out.Width() <= 0 || out.Height() <= 0 {\n" +
		"\t\t\tt.Fatalf(\"identity: produced an empty image %dx%d\", out.Width(), out.Height())\n" +
		"\t\t}\n"

	if got := coverageBody(creator); got != wantCreator {
		t.Errorf("unexpected creator body\n got: %q\nwant: %q", got, wantCreator)
	}
}

// TestCoverageSkipRecordedPrecondition pins that a listed operation skips with
// its reason and does not retry, so a precondition never reads as a wrapper bug.
func TestCoverageSkipRecordedPrecondition(t *testing.T) {
	op := coverageOp("icc_export", "IccExport")
	op.HasThisImageInput = true
	op.HasOneImageOutput = true
	op.RequiredInputs = []introspection.Argument{imageArg("in")}
	op.OptionalInputs = []introspection.Argument{
		{Name: "depth", GoName: "Depth", GoType: "int"},
	}

	body := coverageBody(op)
	if !strings.Contains(body, `t.Skipf("input precondition: needs an embedded profile: %v", err)`) {
		t.Errorf("body does not skip with its reason:\n%s", body)
	}
	if strings.Contains(body, "coverageMonoImage") {
		t.Errorf("a listed operation should not retry:\n%s", body)
	}
}

// TestCoverageSkipEntriesAreReasons keeps the list reviewable: a row without a
// stated precondition is a row that hides a failure.
func TestCoverageSkipEntriesAreReasons(t *testing.T) {
	for name, reason := range coverageSkip {
		if strings.TrimSpace(reason) == "" {
			t.Errorf("%s has no reason", name)
		}
		if len(strings.Fields(reason)) < 3 {
			t.Errorf("%s reason is too short to be a precondition: %q", name, reason)
		}
		if strings.HasSuffix(reason, ".") {
			t.Errorf("%s reason ends with a period: %q", name, reason)
		}
	}
}
