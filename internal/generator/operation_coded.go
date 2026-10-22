package generator

import (
	"fmt"
	"strings"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// codedInterpretations maps the colour-space code that names an A2B operation to the
// interpretation an image carries in and out of it. The code in the name is libvips'
// own spelling of the pair, so the mapping is read off the operation list rather than
// decided here.
var codedInterpretations = map[string]string{
	"BW":    "InterpretationBW",
	"CMC":   "InterpretationCmc",
	"CMYK":  "InterpretationCmyk",
	"HSV":   "InterpretationHsv",
	"Lab":   "InterpretationLab",
	"LabQ":  "InterpretationLabq",
	"LabS":  "InterpretationLabs",
	"LCh":   "InterpretationLch",
	"Oklab": "InterpretationOklab",
	"Oklch": "InterpretationOklch",
	"sRGB":  "InterpretationSrgb",
	"scRGB": "InterpretationScrgb",
	"XYZ":   "InterpretationXyz",
	"Yxy":   "InterpretationYxy",
}

// codedRule splits an A2B operation name and reports the interpretations at both ends.
// Such an operation cannot be called on an ordinary image - it wants a Lab-coded one,
// say - but libvips converts between codings, so a test can start from the sRGB image
// every other subtest uses, convert, call, and then assert the coding that came out.
func codedRule(op introspection.Operation) (in, out string, ok bool) {
	// The conversion takes the receiver, so the operation has to put its image there and
	// ask for nothing else: an extra required argument is a different shape.
	if !op.HasThisImageInput || !op.HasOneImageOutput {
		return "", "", false
	}
	parts := strings.Split(op.Name, "2")
	if len(parts) != 2 {
		return "", "", false
	}
	in, ok = codedInterpretations[parts[0]]
	if !ok {
		return "", "", false
	}
	out, ok = codedInterpretations[parts[1]]
	if !ok {
		return "", "", false
	}
	for _, arg := range op.RequiredInputs {
		if !arg.IsImage && arg.IsInput {
			return "", "", false
		}
	}
	return in, out, true
}

// codedBody converts the image into the coding the operation expects, calls it, and
// checks the coding it left behind: the wrapper's job is to reach libvips with the
// right arguments, and a conversion that landed on the wrong space is silent otherwise.
func codedBody(op introspection.Operation, in, out string, co coverageOptions) string {
	var args string
	if len(op.OptionalInputs) > 0 || len(getSupportedOptionalOutputs(op)) > 0 {
		args = co.Expr
	}

	var body strings.Builder
	body.WriteString("\t\timg := coverageImage(t)\n")
	body.WriteString(co.Decl)
	body.WriteString(fmt.Sprintf("\t\tif err := img.Colourspace(%s, nil); err != nil {\n", in))
	body.WriteString(fmt.Sprintf("\t\t\tt.Fatalf(%q, err)\n", "converting to "+in+": %v"))
	body.WriteString("\t\t}\n")
	body.WriteString(fmt.Sprintf("\t\tif err := img.%s(%s); err != nil {\n", op.GoName, args))
	body.WriteString(errorLine(co.Skip))
	body.WriteString("\t\t}\n")
	body.WriteString(fmt.Sprintf("\t\tif got := img.Interpretation(); got != %s {\n", out))
	body.WriteString(fmt.Sprintf("\t\t\tt.Fatalf(%q, got)\n", op.Name+": left the image coded as %v"))
	body.WriteString("\t\t}\n")
	return body.String()
}
