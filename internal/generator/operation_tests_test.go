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

	if got := coverageBody(circle, nilCoverageOptions()); got != wantBranch {
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

	if got := coverageBody(save, nilCoverageOptions()); got != wantBuffer {
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

	if got := coverageBody(creator, nilCoverageOptions()); got != wantCreator {
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

	body := coverageBody(op, nilCoverageOptions())
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

// TestFormatToken pins how an operation name becomes a format token: the
// direction and the connection suffix go, the format stays.
func TestFormatToken(t *testing.T) {
	tests := map[string]string{
		"jpegsave":        "jpeg",
		"jpegsave_buffer": "jpeg",
		"gifload_source":  "gif",
		"tiffload":        "tiff",
		"dzsave":          "dz",
		"pngsave_target":  "png",
	}
	for name, want := range tests {
		if got := formatToken(name); got != want {
			t.Errorf("formatToken(%q) = %q, want %q", name, got, want)
		}
	}
}

// TestForeignKindSelection pins which filename operations get a round trip: a
// saver that writes a file, and a loader whose format has a saver to write the
// fixture. Everything else stays out.
func TestForeignKindSelection(t *testing.T) {
	filenameArg := introspection.Argument{Name: "filename", GoName: "filename", GoType: "string", IsInput: true}

	// Arguments is what the generator's call-signature logic reads, so a fixture
	// has to populate it as introspection does: required inputs first.
	save := coverageOp("pngsave", "Pngsave")
	save.HasThisImageInput = true
	save.RequiredInputs = []introspection.Argument{imageArg("in"), filenameArg}
	save.Arguments = save.RequiredInputs

	load := coverageOp("pngload", "Pngload")
	load.HasOneImageOutput = true
	load.RequiredInputs = []introspection.Argument{filenameArg}
	load.Arguments = load.RequiredInputs

	raw := coverageOp("rawload", "Rawload")
	raw.HasOneImageOutput = true
	raw.RequiredInputs = []introspection.Argument{
		filenameArg,
		{Name: "width", GoName: "width", GoType: "int", IsInput: true},
		{Name: "height", GoName: "height", GoType: "int", IsInput: true},
		{Name: "bands", GoName: "bands", GoType: "int", IsInput: true},
	}
	raw.Arguments = raw.RequiredInputs

	dz := coverageOp("dzsave", "Dzsave")
	dz.HasThisImageInput = true
	dz.RequiredInputs = []introspection.Argument{imageArg("in"), filenameArg}
	dz.Arguments = dz.RequiredInputs

	pdf := coverageOp("pdfload", "Pdfload")
	pdf.HasOneImageOutput = true
	pdf.RequiredInputs = []introspection.Argument{filenameArg}
	pdf.Arguments = pdf.RequiredInputs

	formats := newCoverageFormats([]introspection.Operation{save, load, raw, dz, pdf})

	if kind, ext, saver, ok := formats.foreignKind(save); !ok || kind != "saver" || ext != "png" {
		t.Errorf("pngsave: got %q/%q/%q/%v, want saver/png", kind, ext, saver, ok)
	}
	if kind, ext, saver, ok := formats.foreignKind(load); !ok || kind != "loader" || ext != "png" || saver != "Pngsave" {
		t.Errorf("pngload: got %q/%q/%q/%v, want loader/png/Pngsave", kind, ext, saver, ok)
	}
	if _, _, _, ok := formats.foreignKind(raw); ok {
		t.Error("rawload needs the dimensions of the bytes, so it cannot round-trip")
	}
	if _, _, _, ok := formats.foreignKind(dz); ok {
		t.Error("dzsave writes a directory, not a file")
	}
	if _, _, _, ok := formats.foreignKind(pdf); ok {
		t.Error("pdfload has no saver to write its fixture")
	}
}

// TestRoundTripBodySnapshots pins the two generated shapes: a saver writing a
// file, and a loader that uses that saver for its fixture and compares geometry.
func TestRoundTripBodySnapshots(t *testing.T) {
	filenameArg := introspection.Argument{Name: "filename", GoName: "filename", GoType: "string", IsInput: true}

	save := coverageOp("pngsave", "Pngsave")
	save.HasThisImageInput = true
	save.RequiredInputs = []introspection.Argument{imageArg("in"), filenameArg}
	save.OptionalInputs = []introspection.Argument{{Name: "compression", GoName: "Compression", GoType: "int"}}

	wantSaver := "" +
		"\t\timg := coverageImage(t)\n" +
		"\t\tpath := coveragePath(t, \"png\")\n" +
		"\t\terr := img.Pngsave(path, nil)\n" +
		"\t\tif err != nil {\n" +
		"\t\t\tfirstErr := err\n" +
		"\t\t\timg = coverageMonoImage(t)\n" +
		"\t\t\terr = img.Pngsave(path, nil)\n" +
		"\t\t\tif err != nil {\n" +
		"\t\t\t\tt.Fatalf(\"pngsave: wrapper failed for a three-band and a one-band image: %v / %v\", firstErr, err)\n" +
		"\t\t\t}\n" +
		"\t\t}\n" +
		"\t\tif info, statErr := os.Stat(path); statErr != nil || info.Size() == 0 {\n" +
		"\t\t\tt.Fatalf(\"pngsave: wrote no file at %s\", path)\n" +
		"\t\t}\n"

	if got := saverCoverageBody(save, "png", nilCoverageOptions()); got != wantSaver {
		t.Errorf("unexpected saver body\n got: %q\nwant: %q", got, wantSaver)
	}

	load := coverageOp("pngload", "Pngload")
	load.HasOneImageOutput = true
	load.RequiredInputs = []introspection.Argument{filenameArg}
	load.OptionalInputs = []introspection.Argument{{Name: "unlimited", GoName: "Unlimited", GoType: "bool"}}

	wantLoader := "" +
		"\t\timg := coverageImage(t)\n" +
		"\t\tpath := coveragePath(t, \"png\")\n" +
		"\t\tif err := img.Pngsave(path, nil); err != nil {\n" +
		"\t\t\tt.Skipf(\"fixture unavailable: the saver could not write it: %v\", err)\n" +
		"\t\t}\n" +
		"\t\tloaded, err := NewPngload(path, nil)\n" +
		"\t\tif err != nil {\n" +
		"\t\t\tt.Fatalf(\"wrapper reported an error: %v\", err)\n" +
		"\t\t}\n" +
		"\t\tdefer loaded.Close()\n" +
		"\t\tif loaded.Width() != img.Width() || loaded.Height() != img.Height() {\n" +
		"\t\t\tt.Fatalf(\"pngload: loaded %dx%d from a %dx%d file\", loaded.Width(), loaded.Height(), img.Width(), img.Height())\n" +
		"\t\t}\n"

	if got := loaderCoverageBody(load, "png", "Pngsave", nilCoverageOptions()); got != wantLoader {
		t.Errorf("unexpected loader body\n got: %q\nwant: %q", got, wantLoader)
	}
}

// TestCoverageOptionsLiteral pins the options literal: the presence rule decides
// Ptr, a float keeps its point when it is wrapped, and a field this generator
// cannot give a value to is left out rather than set to something meaningless.
func TestCoverageOptionsLiteral(t *testing.T) {
	op := coverageOp("webpsave", "Webpsave")
	op.OptionalInputs = []introspection.Argument{
		{Name: "q", GoName: "Q", GoType: "int", DefaultValue: 75},
		{Name: "lossless", GoName: "Lossless", GoType: "bool", DefaultValue: false},
		{Name: "preset", GoName: "Preset", GoType: "WebpPreset", IsEnum: true, EnumType: "WebpPreset", EnumValues: []int{0, 1, 2}, DefaultValue: 0},
		{Name: "profile", GoName: "Profile", GoType: "string"},
		{Name: "sigma", GoName: "Sigma", GoType: "float64", DefaultValue: 1.0},
	}

	co, ok := coverageOptionsLiteral(op)
	if !ok {
		t.Fatal("expected a literal: four of the five fields can be given a value")
	}
	want := "\t\topts := &WebpsaveOptions{Q: Ptr(1), Lossless: true, Preset: 0, Sigma: Ptr(1.0)}\n"
	if co.Decl != want {
		t.Errorf("unexpected literal\n got: %q\nwant: %q", co.Decl, want)
	}
	if co.Expr != "opts" {
		t.Errorf("Expr = %q, want opts", co.Expr)
	}
}

func TestCoverageOptionsLiteralSkipsUnusableFields(t *testing.T) {
	op := coverageOp("thumbnail", "Thumbnail")
	op.OptionalInputs = []introspection.Argument{
		{Name: "input_profile", GoName: "InputProfile", GoType: "string"},
		{Name: "interpolate", GoName: "Interpolate", GoType: "*Interpolate"},
	}

	if _, ok := coverageOptionsLiteral(op); ok {
		t.Error("a string and an interpolator alone leave nothing to set")
	}
}

// TestCoverageOptionsSkipPrecedence pins that a variant's own reason wins, and
// that an operation-level reason still applies to the plain call.
func TestCoverageOptionsSkipPrecedence(t *testing.T) {
	if got := (coverageOptions{Skip: "variant reason"}).skipReason("rad2float"); got != "variant reason" {
		t.Errorf("variant reason did not win: %q", got)
	}
	if got := (coverageOptions{}).skipReason("rad2float"); got != coverageSkip["rad2float"] {
		t.Errorf("operation reason not used: %q", got)
	}
}

// TestValuesCaseBodySnapshots pins the three shapes a value-resulting operation
// gets: a scalar in the range the image can produce, an integer tuple that must
// not be negative, and a call with no result at all.
func TestValuesCaseBodySnapshots(t *testing.T) {
	avg := coverageOp("avg", "Avg")
	avg.HasThisImageInput = true
	avg.RequiredInputs = []introspection.Argument{imageArg("in")}
	avg.Arguments = append([]introspection.Argument{imageArg("in")},
		introspection.Argument{Name: "out", GoName: "out", GoType: "float64", IsOutput: true})
	avg.RequiredOutputs = []introspection.Argument{
		{Name: "out", GoName: "out", GoType: "float64", IsOutput: true},
	}

	wantAvg := "" +
		"\t\timg := coverageImage(t)\n" +
		"\t\tv0, err := img.Avg()\n" +
		"\t\tif err != nil {\n" +
		"\t\t\tt.Fatalf(\"wrapper reported an error: %v\", err)\n" +
		"\t\t}\n" +
		"\t\tif math.IsNaN(v0) || v0 < 0 || v0 > 255 {\n" +
		"\t\t\tt.Fatalf(\"avg: %v is outside the range of a uchar image\", v0)\n" +
		"\t\t}\n"

	if got, ok := valuesCaseBody(avg, nilCoverageOptions()); !ok || got != wantAvg {
		t.Errorf("unexpected scalar body\n got: %q\nwant: %q", got, wantAvg)
	}

	trim := coverageOp("find_trim", "FindTrim")
	trim.HasThisImageInput = true
	trim.RequiredInputs = []introspection.Argument{imageArg("in")}
	trim.Arguments = []introspection.Argument{imageArg("in")}
	trim.RequiredOutputs = []introspection.Argument{
		{Name: "left", GoName: "left", GoType: "int", IsOutput: true},
		{Name: "top", GoName: "top", GoType: "int", IsOutput: true},
	}

	wantTrim := "" +
		"\t\timg := coverageImage(t)\n" +
		"\t\tv0, v1, err := img.FindTrim()\n" +
		"\t\tif err != nil {\n" +
		"\t\t\tt.Fatalf(\"wrapper reported an error: %v\", err)\n" +
		"\t\t}\n" +
		"\t\tif v0 < 0 {\n\t\t\tt.Fatalf(\"find_trim: %d is negative\", v0)\n\t\t}\n" +
		"\t\tif v1 < 0 {\n\t\t\tt.Fatalf(\"find_trim: %d is negative\", v1)\n\t\t}\n"

	if got, ok := valuesCaseBody(trim, nilCoverageOptions()); !ok || got != wantTrim {
		t.Errorf("unexpected tuple body\n got: %q\nwant: %q", got, wantTrim)
	}

	void := coverageOp("draw_line", "DrawLine")
	void.HasThisImageInput = true
	void.RequiredInputs = []introspection.Argument{imageArg("in"), {Name: "x", GoName: "x", GoType: "int", IsInput: true}}
	void.Arguments = void.RequiredInputs

	wantVoid := "" +
		"\t\timg := coverageImage(t)\n" +
		"\t\terr := img.DrawLine(0)\n" +
		"\t\tif err != nil {\n" +
		"\t\t\tt.Fatalf(\"wrapper reported an error: %v\", err)\n" +
		"\t\t}\n"

	if got, ok := valuesCaseBody(void, nilCoverageOptions()); !ok || got != wantVoid {
		t.Errorf("unexpected void body\n got: %q\nwant: %q", got, wantVoid)
	}
}

// TestConnectionCaseBodySnapshots pins the two connection shapes: a loader that
// reads a Source - with the fixture written either to bytes or through a Target -
// and a saver that writes a Target.
func TestConnectionCaseBodySnapshots(t *testing.T) {
	load := introspection.Operation{
		Name:              "jpegload_source",
		GoName:            "JpegloadSource",
		HasOneImageOutput: true,
		Arguments: []introspection.Argument{
			{Name: "source", GoType: "*C.VipsSourceCustom", IsInput: true, IsSource: true},
			{Name: "out", GoName: "out", GoType: "*C.VipsImage", IsOutput: true},
		},
		RequiredInputs: []introspection.Argument{
			{Name: "source", GoType: "*C.VipsSourceCustom", IsInput: true, IsSource: true},
		},
	}

	loadTail := "" +
		"\t\tloaded, err := NewJpegloadSource(source)\n" +
		"\t\tif err != nil {\n" +
		"\t\t\tt.Fatalf(\"wrapper reported an error: %v\", err)\n" +
		"\t\t}\n" +
		"\t\tdefer loaded.Close()\n" +
		"\t\tif loaded.Width() != img.Width() || loaded.Height() != img.Height() {\n" +
		"\t\t\tt.Fatalf(\"jpegload_source: loaded %dx%d from a %dx%d stream\", loaded.Width(), loaded.Height(), img.Width(), img.Height())\n" +
		"\t\t}\n"

	wantBytes := "" +
		"\t\timg := coverageImage(t)\n" +
		"\t\tdata, err := img.JpegsaveBuffer(nil)\n" +
		"\t\tif err != nil {\n" +
		"\t\t\tt.Skipf(\"fixture unavailable: the saver could not write it: %v\", err)\n" +
		"\t\t}\n" +
		"\t\tsource := coverageSource(t, data)\n" +
		loadTail

	if got := sourceLoaderCoverageBody(load, "JpegsaveBuffer", nilCoverageOptions()); got != wantBytes {
		t.Errorf("unexpected source loader body\n got: %q\nwant: %q", got, wantBytes)
	}

	// A format whose only byte-producing saver writes to a connection: the fixture
	// is caught on the way to the target.
	wantTarget := "" +
		"\t\timg := coverageImage(t)\n" +
		"\t\ttarget, buf := coverageTarget(t)\n" +
		"\t\tif err := img.VipssaveTarget(target, nil); err != nil {\n" +
		"\t\t\tt.Skipf(\"fixture unavailable: the saver could not write it: %v\", err)\n" +
		"\t\t}\n" +
		"\t\tdata := buf.Bytes()\n" +
		"\t\tsource := coverageSource(t, data)\n" +
		loadTail

	if got := sourceLoaderCoverageBody(load, "VipssaveTarget", nilCoverageOptions()); got != wantTarget {
		t.Errorf("unexpected target-fixture body\n got: %q\nwant: %q", got, wantTarget)
	}

	save := introspection.Operation{
		Name:              "jpegsave_target",
		GoName:            "JpegsaveTarget",
		HasThisImageInput: true,
		Arguments: []introspection.Argument{
			imageArg("in"),
			{Name: "target", GoType: "*C.VipsTargetCustom", IsInput: true, IsTarget: true},
		},
		RequiredInputs: []introspection.Argument{
			imageArg("in"),
			{Name: "target", GoType: "*C.VipsTargetCustom", IsInput: true, IsTarget: true},
		},
	}

	wantSave := "" +
		"\t\timg := coverageImage(t)\n" +
		"\t\ttarget, buf := coverageTarget(t)\n" +
		"\t\terr := img.JpegsaveTarget(target)\n" +
		"\t\tif err != nil {\n" +
		"\t\t\tfirstErr := err\n" +
		"\t\t\timg = coverageMonoImage(t)\n" +
		"\t\t\ttarget, buf = coverageTarget(t)\n" +
		"\t\t\terr = img.JpegsaveTarget(target)\n" +
		"\t\t\tif err != nil {\n" +
		"\t\t\t\tt.Fatalf(\"jpegsave_target: wrapper failed for a three-band and a one-band image: %v / %v\", firstErr, err)\n" +
		"\t\t\t}\n" +
		"\t\t}\n" +
		"\t\tif buf.Len() == 0 {\n" +
		"\t\t\tt.Fatalf(\"jpegsave_target: wrote nothing to the target\")\n" +
		"\t\t}\n"

	if got := targetSaverCoverageBody(save, nilCoverageOptions()); got != wantSave {
		t.Errorf("unexpected target saver body\n got: %q\nwant: %q", got, wantSave)
	}
}

// TestBufferCaseBodySnapshots pins the buffer loader shapes: the fixture written to
// bytes by the format's own saver, or - where the format has no buffer saver -
// caught on its way to a target.
func TestBufferCaseBodySnapshots(t *testing.T) {
	load := introspection.Operation{
		Name:              "pngload_buffer",
		GoName:            "PngloadBuffer",
		HasOneImageOutput: true,
		Arguments: []introspection.Argument{
			{Name: "buf", GoName: "buf", GoType: "[]byte", IsInput: true, IsBuffer: true},
			{Name: "len", GoName: "len", GoType: "int", IsInput: true},
			{Name: "out", GoName: "out", GoType: "*C.VipsImage", IsOutput: true},
		},
		RequiredInputs: []introspection.Argument{
			{Name: "buf", GoName: "buf", GoType: "[]byte", IsInput: true, IsBuffer: true},
			{Name: "len", GoName: "len", GoType: "int", IsInput: true},
		},
	}

	tail := "" +
		"\t\tloaded, err := NewPngloadBuffer(data)\n" +
		"\t\tif err != nil {\n" +
		"\t\t\tt.Fatalf(\"wrapper reported an error: %v\", err)\n" +
		"\t\t}\n" +
		"\t\tdefer loaded.Close()\n" +
		"\t\tif loaded.Width() != img.Width() || loaded.Height() != img.Height() {\n" +
		"\t\t\tt.Fatalf(\"pngload_buffer: loaded %dx%d from a %dx%d buffer\", loaded.Width(), loaded.Height(), img.Width(), img.Height())\n" +
		"\t\t}\n"

	wantBytes := "" +
		"\t\timg := coverageImage(t)\n" +
		"\t\tdata, err := img.PngsaveBuffer(nil)\n" +
		"\t\tif err != nil {\n" +
		"\t\t\tt.Skipf(\"fixture unavailable: the saver could not write it: %v\", err)\n" +
		"\t\t}\n" +
		tail

	if got := bufferLoaderCoverageBody(load, "PngsaveBuffer", nilCoverageOptions()); got != wantBytes {
		t.Errorf("unexpected buffer loader body\n got: %q\nwant: %q", got, wantBytes)
	}

	wantTarget := "" +
		"\t\timg := coverageImage(t)\n" +
		"\t\ttarget, buf := coverageTarget(t)\n" +
		"\t\tif err := img.VipssaveTarget(target, nil); err != nil {\n" +
		"\t\t\tt.Skipf(\"fixture unavailable: the saver could not write it: %v\", err)\n" +
		"\t\t}\n" +
		"\t\tdata := buf.Bytes()\n" +
		tail

	if got := bufferLoaderCoverageBody(load, "VipssaveTarget", nilCoverageOptions()); got != wantTarget {
		t.Errorf("unexpected target-fixture buffer body\n got: %q\nwant: %q", got, wantTarget)
	}
}
