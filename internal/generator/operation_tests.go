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
	"heifsave_buffer":   "needs a heif encoder for its default compression",
	"uhdrsave":          "needs gainmap header fields",
	"heifsave":          "needs a heif encoder for its default compression",
	"grid":              "needs an image that is a multiple of the tile size",
	"match":             "needs an invertible matrix image",
	"morph":             "needs a matrix-interpretation mask",
	"mosaic":            "needs two overlapping images",
	"mosaic1":           "needs two overlapping images",
	"sdf":               "needs its shape options set",
	"switch":            "needs one-band test images",
}

// BuildOperationCoverage selects the operations a generated test can call and
// renders one subtest per operation. Selection is the argument list: an
// operation qualifies when every required argument is an image or a value this
// generator can make up - a number, a flag, an enum member or an array of
// numbers. Anything that needs a filename, a source or a target is out of scope.
func BuildOperationCoverage(operations []introspection.Operation) []CoverageCase {
	formats := newCoverageFormats(operations)
	var cases []CoverageCase
	for _, op := range operations {
		if body, ok := coverageCaseBody(op, formats); ok {
			cases = append(cases, CoverageCase{Name: op.Name, Body: body})
		}
	}
	return cases
}

// coverageCaseBody picks the kind of subtest an operation gets: a round trip for a
// format libvips both writes and reads, otherwise a call with made-up arguments.
func coverageCaseBody(op introspection.Operation, formats coverageFormats) (string, bool) {
	if kind, ext, saver, ok := formats.foreignKind(op); ok {
		if kind == "saver" {
			return saverCoverageBody(op, ext), true
		}
		return loaderCoverageBody(op, ext, saver), true
	}
	if !coverableOperation(op) {
		return "", false
	}
	return coverageBody(op), true
}

// coverableOperation is the selection rule.
func coverableOperation(op introspection.Operation) bool {
	if !op.HasOneImageOutput && !op.HasBufferOutput {
		return false
	}
	// A required buffer is a loader's formatted bytes, which a generated test
	// cannot make up: the format is the operation's whole subject.
	for _, arg := range op.RequiredInputs {
		if arg.IsBuffer {
			return false
		}
	}
	for _, arg := range callArguments(op) {
		switch arg.GoType {
		case "*C.VipsImage", "[]*C.VipsImage", "int", "float64", "bool", "[]float64", "[]int":
		default:
			if !arg.IsEnum {
				return false
			}
		}
	}
	return true
}

// callArguments is the argument list of the generated call, in signature order
// and without the options struct: the receiver is skipped for a method, as are
// the input length libvips derives from a buffer and the array length vipsgen
// derives from a slice.
func callArguments(op introspection.Operation) []introspection.Argument {
	if op.HasThisImageInput {
		return detectMethodArguments(op)
	}
	var args []introspection.Argument
	for _, arg := range op.RequiredInputs {
		if arg.IsInputN || arg.IsBuffer || arg.Name == "len" {
			continue
		}
		args = append(args, arg)
	}
	return args
}

// coverageBody renders the call and its assertions for one operation.
func coverageBody(op introspection.Operation) string {
	skipReason := coverageSkip[op.Name]
	hasOptions := len(op.OptionalInputs) > 0 || len(getSupportedOptionalOutputs(op)) > 0

	var exprs []string
	var imageArgs []string
	for _, arg := range callArguments(op) {
		switch arg.GoType {
		case "*C.VipsImage":
			name := imageName(len(imageArgs))
			imageArgs = append(imageArgs, name)
			exprs = append(exprs, name)
		case "[]*C.VipsImage":
			first := imageName(len(imageArgs))
			second := imageName(len(imageArgs) + 1)
			imageArgs = append(imageArgs, first, second)
			exprs = append(exprs, fmt.Sprintf("[]*Image{%s, %s}", first, second))
		default:
			value, ok := scalarValue(arg)
			if !ok {
				return ""
			}
			exprs = append(exprs, value)
		}
	}
	if hasOptions {
		exprs = append(exprs, "nil")
	}
	argsExpr := strings.Join(exprs, ", ")

	var body strings.Builder
	if !op.HasThisImageInput {
		// A creator: NewIdentity(nil), NewBandjoin([]*Image{...}).
		body.WriteString(declareImages(imageArgs, "coverageImage", false, 2))
		body.WriteString(fmt.Sprintf("\t\tout, err := New%s(%s)\n", op.GoName, argsExpr))
		body.WriteString("\t\tif err != nil {\n")
		body.WriteString(errorLine(skipReason))
		body.WriteString("\t\t}\n")
		body.WriteString("\t\tdefer out.Close()\n")
		body.WriteString("\t\tif out.Width() <= 0 || out.Height() <= 0 {\n")
		body.WriteString(fmt.Sprintf("\t\t\tt.Fatalf(%q, out.Width(), out.Height())\n", op.Name+": produced an empty image %dx%d"))
		body.WriteString("\t\t}\n")
		return body.String()
	}

	body.WriteString(declareImages(append([]string{"img"}, imageArgs...), "coverageImage", false, 2))
	body.WriteString(callLine(op, argsExpr, false, 2))

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
	body.WriteString(declareImages(append([]string{"img"}, imageArgs...), "coverageMonoImage", true, 3))
	body.WriteString(callLine(op, argsExpr, true, 3))
	body.WriteString("\t\t\tif err != nil {\n")
	body.WriteString(fmt.Sprintf("\t\t\t\tt.Fatalf(%q, firstErr, err)\n", op.Name+": wrapper failed for a three-band and a one-band image: %v / %v"))
	body.WriteString("\t\t\t}\n")
	body.WriteString("\t\t}\n")
	body.WriteString(assertions(op))
	return body.String()
}

// scalarValue renders a value for a required argument from its metadata, and
// reports whether this generator can make one up.
//
// The rules are name-aware where libvips demands a particular kind of number (a
// size, a coordinate) and range-clamped everywhere else, because a value outside
// its range is a call libvips rejects for a reason the test would report as a
// wrapper failure.
func scalarValue(arg introspection.Argument) (string, bool) {
	if arg.IsEnum {
		if strings.HasPrefix(arg.GoType, "[]") {
			return fmt.Sprintf("[]%s{%s}", strings.TrimPrefix(arg.GoType, "[]"), enumValue(arg)), true
		}
		return enumValue(arg), true
	}
	switch arg.GoType {
	case "bool":
		return "true", true
	case "int":
		return intValue(arg), true
	case "float64":
		return floatValue(arg), true
	case "[]float64":
		return floatArrayValue(arg), true
	case "[]int":
		return "[]int{1}", true
	}
	return "", false
}

// intValue picks a small integer that respects the argument's range.
func intValue(arg introspection.Argument) string {
	value := 1
	switch arg.Name {
	case "width", "height", "tile_width", "tile_height":
		value = 8
	case "left", "top", "x", "y", "xoffset", "yoffset", "xr1", "xr2", "yr1", "yr2":
		value = 0
	case "xfac", "yfac", "across", "down", "n", "bands", "depth", "page", "index":
		value = 1
	}
	return fmt.Sprintf("%d", clampInt(value, arg))
}

// floatValue picks a small positive float that respects the argument's range.
func floatValue(arg introspection.Argument) string {
	value := 1.0
	switch arg.Name {
	case "fractal_dimension":
		value = 2.0
	case "b", "c":
		// The off-diagonal coefficients of affine() and the like: one on the
		// diagonal, zero beside it, or the matrix inverts to nothing.
		value = 0
	}
	return trimFloat(clampFloat(value, arg))
}

// floatArrayValue supplies the number array an argument takes. The ink of a draw
// operation is a colour, the coefficients of linear() are the identity, and the
// rest are constants the operation scales by.
func floatArrayValue(arg introspection.Argument) string {
	switch arg.Name {
	case "ink":
		return "[]float64{255}"
	case "a":
		return "[]float64{1}"
	case "b":
		return "[]float64{0}"
	}
	return "[]float64{1}"
}

// enumValue names the first member of the argument's enum: the value libvips
// documents as the default for every argument this generator selects.
// enumFallback names the member to use where value zero is a sentinel rather
// than something libvips accepts: Interpretation starts with ERROR.
var enumFallback = map[string]string{
	"Interpretation": "InterpretationSrgb",
}

func enumValue(arg introspection.Argument) string {
	if name, ok := enumFallback[arg.EnumType]; ok {
		return name
	}
	if len(arg.EnumValues) > 0 {
		if name, ok := enumValueNames[arg.EnumType][arg.EnumValues[0]]; ok {
			return name
		}
	}
	if name, ok := enumValueNames[arg.EnumType][0]; ok {
		return name
	}
	return "0"
}

func clampInt(value int, arg introspection.Argument) int {
	if arg.HasMinimum && float64(value) < arg.Minimum {
		value = int(arg.Minimum)
	}
	if arg.HasMaximum && float64(value) > arg.Maximum {
		value = int(arg.Maximum)
	}
	return value
}

func clampFloat(value float64, arg introspection.Argument) float64 {
	if arg.HasMinimum && value < arg.Minimum {
		value = arg.Minimum
	}
	if arg.HasMaximum && value > arg.Maximum {
		value = arg.Maximum
	}
	return value
}

// trimFloat renders a whole float without a trailing ".0", so the generated call
// reads like one a caller would write.
func trimFloat(value float64) string {
	if value == float64(int64(value)) {
		return fmt.Sprintf("%d", int64(value))
	}
	return fmt.Sprintf("%g", value)
}

// imageName names the n-th image argument: other, other2, other3.
func imageName(n int) string {
	if n == 0 {
		return "other"
	}
	return fmt.Sprintf("other%d", n+1)
}

// declareImages renders the image locals a call needs. The first name is always
// the receiver's img for a method.
func declareImages(names []string, helper string, assign bool, depth int) string {
	op := " := "
	if assign {
		op = " = "
	}
	tabs := strings.Repeat("\t", depth)
	var code strings.Builder
	for _, name := range names {
		code.WriteString(tabs + name + op + helper + "(t)\n")
	}
	return code.String()
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

// errorLine renders what to do when the call reports an error: a skip carrying
// the recorded precondition, or a failure naming the operation.
func errorLine(skipReason string) string {
	if skipReason != "" {
		return fmt.Sprintf("\t\t\tt.Skipf(%q, err)\n", "input precondition: "+skipReason+": %v")
	}
	return "\t\t\tt.Fatalf(\"wrapper reported an error: %v\", err)\n"
}
