package generator

import (
	"fmt"
	"strings"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// CoverageCase is one generated subtest: an operation called with the smallest
// input that reaches its wrapper.
type CoverageCase struct {
	Name string // subtest name, the libvips operation name
	Body string // rendered test body, already indented for the t.Run closure
}

// coverageSkip records an operation whose wrapper is called but whose input
// cannot satisfy the operation's precondition, with the reason. The call still
// runs - it asserts no panic and no leak - but an error is reported as a skip.
//
// Each entry is a claim about the input, not about the code. Adding a row for an
// operation that fails for a new reason would hide that failure, so the reason
// is written down and reviewable.
var coverageSkip = map[string]string{
	"CMC2LCh":           "needs a CMC-coded input",
	"LCh2CMC":           "needs an LCh-coded input",
	"Lab2LabQ":          "needs a Lab-coded input",
	"Lab2LabS":          "needs a Lab-coded input",
	"LabQ2Lab":          "needs a LabQ-coded input",
	"LabQ2LabS":         "needs a LabQ-coded input",
	"LabQ2sRGB":         "needs a LabQ-coded input",
	"LabS2Lab":          "needs a LabS-coded input",
	"LabS2LabQ":         "needs a LabS-coded input",
	"Oklab2Oklch":       "needs an Oklab-coded input",
	"Oklch2Oklab":       "needs an Oklch-coded input",
	"XYZ2Yxy":           "needs an XYZ-coded input",
	"Yxy2XYZ":           "needs a Yxy-coded input",
	"CMYK2XYZ":          "needs a CMYK-coded input",
	"XYZ2CMYK":          "needs an XYZ-coded input",
	"float2rad":         "needs a float-coded input",
	"rad2float":         "needs a Radiance-coded input",
	"icc_import":        "needs an embedded or supplied input profile",
	"icc_export":        "needs an embedded profile",
	"icc_transform":     "needs an output profile",
	"autorot":           "needs an EXIF orientation tag",
	"transpose3d":       "needs a page-height header field",
	"uhdrsave_buffer":   "needs gainmap header fields",
	"hist_plot":         "needs a histogram image",
	"globalbalance":     "needs an alpha band",
	"uhdr2scRGB":        "needs a uchar three-band input",
	"buildlut":          "needs a matrix-interpretation input",
	"invertlut":         "needs a matrix-interpretation input",
	"matrixinvert":      "needs a matrix-interpretation input",
	"quadratic":         "needs a matrix-interpretation input",
	"convasep":          "needs a one-band image and a matrix mask",
	"convsep":           "needs a one-band image and a matrix mask",
	"recomb":            "needs a one-band image and a matrix mask",
	"hist_cum":          "needs a histogram: width or height 1",
	"hist_match":        "needs histograms: width or height 1",
	"maplut":            "needs a histogram: width or height 1",
	"mapim":             "needs a two-band or complex index image",
	"magicksave_buffer": "needs an output format",
}

// BuildOperationCoverage selects the operations a generated test can call with
// images alone and renders one subtest per operation. Operations that need a
// value in a range, a filename or a target are out of scope here.
func BuildOperationCoverage(operations []introspection.Operation) []CoverageCase {
	var cases []CoverageCase
	for _, op := range operations {
		if !coverableOperation(op) {
			continue
		}
		cases = append(cases, CoverageCase{
			Name: op.Name,
			Body: coverageBody(op),
		})
	}
	return cases
}

// coverableOperation is the selection rule: every required input is an image,
// and the operation hands back either one image or a buffer.
func coverableOperation(op introspection.Operation) bool {
	for _, arg := range op.RequiredInputs {
		if arg.IsInput && !arg.IsImage {
			return false
		}
	}
	return op.HasOneImageOutput || op.HasBufferOutput
}

// coverageBody renders the call and its assertions for one operation.
func coverageBody(op introspection.Operation) string {
	skipReason := coverageSkip[op.Name]
	extraImages := extraImageArgs(op)
	hasOptions := len(op.OptionalInputs) > 0 || len(getSupportedOptionalOutputs(op)) > 0

	var callArgs []string
	callArgs = append(callArgs, extraImages...)
	if hasOptions {
		callArgs = append(callArgs, "nil")
	}
	args := strings.Join(callArgs, ", ")

	var body strings.Builder
	if !op.HasThisImageInput {
		// A creator: NewIdentity(nil), NewTonelut(nil).
		body.WriteString(fmt.Sprintf("\t\tout, err := New%s(%s)\n", op.GoName, args))
		body.WriteString("\t\tif err != nil {\n")
		body.WriteString(errorLine(skipReason))
		body.WriteString("\t\t}\n")
		body.WriteString("\t\tdefer out.Close()\n")
		body.WriteString("\t\tif out.Width() <= 0 || out.Height() <= 0 {\n")
		body.WriteString(fmt.Sprintf("\t\t\tt.Fatalf(%q, out.Width(), out.Height())\n", op.Name+": produced an empty image %dx%d"))
		body.WriteString("\t\t}\n")
		return body.String()
	}

	body.WriteString(declareImages("coverageImage", extraImages, false, 2))
	body.WriteString(callLine(op, args, false, 2))

	if skipReason != "" {
		body.WriteString("\t\tif err != nil {\n")
		body.WriteString(errorLine(skipReason))
		body.WriteString("\t\t}\n")
		body.WriteString(assertions(op))
		return body.String()
	}

	// Two shapes of input: the colour operations want three bands, the masks,
	// matrices and histograms want one. Rather than classify the operation, call
	// it with the wider image and, if libvips refuses, again with a one-band one.
	// Both failing is a failure; one succeeding is the coverage this claims.
	body.WriteString("\t\tif err != nil {\n")
	body.WriteString("\t\t\tfirstErr := err\n")
	// Reuse the same locals: the assertions after the retry read them, so they
	// must not be shadowed by the second call.
	body.WriteString(declareImages("coverageMonoImage", extraImages, true, 3))
	body.WriteString(callLine(op, args, true, 3))
	body.WriteString("\t\t\tif err != nil {\n")
	body.WriteString(fmt.Sprintf("\t\t\t\tt.Fatalf(%q, firstErr, err)\n", op.Name+": wrapper failed for a three-band and a one-band image: %v / %v"))
	body.WriteString("\t\t\t}\n")
	body.WriteString("\t\t}\n")
	body.WriteString(assertions(op))
	return body.String()
}

// declareImages renders the image locals a call needs: the receiver and one
// image per extra image argument.
func declareImages(helper string, extraImages []string, assign bool, depth int) string {
	op := " := "
	if assign {
		op = " = "
	}
	tabs := strings.Repeat("\t", depth)
	code := tabs + "img" + op + helper + "(t)\n"
	for _, name := range extraImages {
		code += tabs + name + op + helper + "(t)\n"
	}
	return code
}

// callLine renders the call itself: a buffer-producing operation fills buf, any
// other fills err.
func callLine(op introspection.Operation, args string, assign bool, depth int) string {
	assignOp := ":="
	if assign {
		assignOp = "="
	}
	indent := strings.Repeat("\t", depth)
	if op.HasBufferOutput {
		return fmt.Sprintf("%sbuf, err %s img.%s(%s)\n", indent, assignOp, op.GoName, args)
	}
	return fmt.Sprintf("%serr %s img.%s(%s)\n", indent, assignOp, op.GoName, args)
}

// assertions checks that the call produced something usable.
func assertions(op introspection.Operation) string {
	if op.HasBufferOutput {
		return "\t\tif len(buf) == 0 {\n" +
			fmt.Sprintf("\t\t\tt.Fatalf(%q, len(buf))\n", op.Name+": wrote an empty buffer: %d bytes") +
			"\t\t}\n"
	}
	return "\t\tif img.Width() <= 0 || img.Height() <= 0 {\n" +
		fmt.Sprintf("\t\t\tt.Fatalf(%q, img.Width(), img.Height())\n", op.Name+": produced an empty image %dx%d") +
		"\t\t}\n"
}

// extraImageArgs names the image arguments that follow the receiver: other,
// other2, other3. They take their own image so an operation that combines two
// inputs is not handed the same pointer twice.
func extraImageArgs(op introspection.Operation) []string {
	var images []introspection.Argument
	for _, arg := range op.RequiredInputs {
		if arg.IsImage && !arg.IsInputN {
			images = append(images, arg)
		}
	}
	if op.HasThisImageInput && len(images) > 0 {
		images = images[1:]
	}
	names := make([]string, 0, len(images))
	for i := range images {
		if i == 0 {
			names = append(names, "other")
			continue
		}
		names = append(names, fmt.Sprintf("other%d", i+1))
	}
	return names
}

// errorLine renders what to do when the call reports an error: a skip carrying
// the recorded precondition, or a failure naming the operation.
func errorLine(skipReason string) string {
	if skipReason != "" {
		return fmt.Sprintf("\t\t\tt.Skipf(%q, err)\n", "input precondition: "+skipReason+": %v")
	}
	return "\t\t\tt.Fatalf(\"wrapper reported an error: %v\", err)\n"
}
