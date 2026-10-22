package generator

import (
	"testing"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// TestCodedRule pins that the two ends of an A2B name are read as interpretations, and
// that only an operation taking nothing but its own image qualifies.
func TestCodedRule(t *testing.T) {
	lab2xyz := coverageOp("Lab2XYZ", "Lab2XYZ")
	lab2xyz.HasThisImageInput = true
	lab2xyz.HasOneImageOutput = true
	lab2xyz.RequiredInputs = []introspection.Argument{imageArg("in")}

	in, out, ok := codedRule(lab2xyz)
	if !ok || in != "InterpretationLab" || out != "InterpretationXyz" {
		t.Errorf("Lab2XYZ should run Lab into XYZ, got %q %q %v", in, out, ok)
	}

	scrgb := coverageOp("sRGB2scRGB", "SRGB2scRGB")
	scrgb.HasThisImageInput = true
	scrgb.HasOneImageOutput = true
	scrgb.RequiredInputs = []introspection.Argument{imageArg("in")}
	if in, out, ok := codedRule(scrgb); !ok || in != "InterpretationSrgb" || out != "InterpretationScrgb" {
		t.Errorf("sRGB2scRGB should run sRGB into scRGB, got %q %q %v", in, out, ok)
	}

	extra := coverageOp("Lab2XYZ", "Lab2XYZ")
	extra.HasThisImageInput = true
	extra.HasOneImageOutput = true
	extra.RequiredInputs = []introspection.Argument{
		imageArg("in"),
		{Name: "temp", GoName: "temp", GoType: "[]float64", IsInput: true},
	}
	if _, _, ok := codedRule(extra); ok {
		t.Error("an operation with another required argument is a different shape")
	}

	plain := coverageOp("avg", "Avg")
	plain.HasThisImageInput = true
	plain.HasOneImageOutput = true
	plain.RequiredInputs = []introspection.Argument{imageArg("in")}
	if _, _, ok := codedRule(plain); ok {
		t.Error("avg is not a colour-space conversion")
	}

	unknown := coverageOp("nonsense2xyz", "Nonsense2Xyz")
	unknown.HasThisImageInput = true
	unknown.HasOneImageOutput = true
	unknown.RequiredInputs = []introspection.Argument{imageArg("in")}
	if _, _, ok := codedRule(unknown); ok {
		t.Error("a name whose left side is not a coding is not a conversion")
	}
}

// TestCodedBodySnapshot pins the three things the body does: convert, call, and assert
// the coding that came out.
func TestCodedBodySnapshot(t *testing.T) {
	op := coverageOp("Lab2XYZ", "Lab2XYZ")
	op.HasThisImageInput = true
	op.HasOneImageOutput = true
	op.RequiredInputs = []introspection.Argument{imageArg("in")}

	want := "\t\timg := coverageImage(t)\n" +
		"\t\tif err := img.Colourspace(InterpretationLab, nil); err != nil {\n" +
		"\t\t\tt.Fatalf(\"converting to InterpretationLab: %v\", err)\n" +
		"\t\t}\n" +
		"\t\tif err := img.Lab2XYZ(); err != nil {\n" +
		"\t\t\tt.Fatalf(\"wrapper reported an error: %v\", err)\n" +
		"\t\t}\n" +
		"\t\tif got := img.Interpretation(); got != InterpretationXyz {\n" +
		"\t\t\tt.Fatalf(\"Lab2XYZ: left the image coded as %v\", got)\n" +
		"\t\t}\n"

	if got := codedBody(op, "InterpretationLab", "InterpretationXyz", nilCoverageOptions()); got != want {
		t.Errorf("unexpected body\n got: %q\nwant: %q", got, want)
	}
}
