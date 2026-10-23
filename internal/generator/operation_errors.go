package generator

import (
	"fmt"
	"strings"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// errorShape is how the operation is called when its arguments are invalid: the three shapes
// a generated wrapper takes, each reporting the failure a different way.
type errorShape int

const (
	// shapeMethod modifies the image it is called on: img.Op(...).
	shapeMethod errorShape = iota
	// shapeCreator builds an image from arguments: NewOp(...).
	shapeCreator
	// shapeSaver writes an image somewhere: img.Op("").
	shapeSaver
)

func errorShapeOf(op introspection.Operation) (errorShape, bool) {
	switch {
	case op.HasThisImageInput && op.HasOneImageOutput:
		return shapeMethod, true
	case !op.HasThisImageInput && op.HasOneImageOutput:
		return shapeCreator, true
	case op.HasThisImageInput && !op.HasOneImageOutput &&
		len(op.RequiredOutputs) == 0 && len(op.OptionalOutputs) == 0:
		return shapeSaver, true
	}
	return 0, false
}

// invalidArgExpr is a value libvips rejects: a required number at zero, an enum outside its
// range, or an empty filename. libvips reports all three as a failed call.
func invalidArgExpr(arg introspection.Argument) (string, bool) {
	switch {
	case arg.IsEnum || arg.IsFlags:
		return fmt.Sprintf("%s(%d)", arg.GoType, invalidEnumValue), true
	case numericGoType(arg.GoType):
		return "0", true
	case arg.GoType == "string":
		return `""`, true
	}
	return "", false
}

// errorCaseBody renders a subtest that calls an operation with arguments libvips rejects, so
// the wrapper's error return runs. An operation that accepts them, or only fails once lazily
// evaluated, has nothing to show and skips: the point is the wrapper's own error path.
//
// co carries the options argument: nil for the plain call, or a literal, which reaches the
// second shim every operation with options has.
func errorCaseBody(op introspection.Operation, co coverageOptions) (string, bool) {
	shape, ok := errorShapeOf(op)
	if !ok {
		return "", false
	}
	if shape == shapeCreator && op.HasBufferInput {
		// The bytes are a hidden argument: callArguments drops it, so the call cannot be built.
		return "", false
	}

	var exprs []string
	injected := false
	for _, arg := range callArguments(op) {
		expr, ok := invalidArgExpr(arg)
		if !ok {
			// An image, source, target or array: there is no invalid one to pass.
			return "", false
		}
		exprs = append(exprs, expr)
		injected = true
	}
	if !injected {
		return "", false
	}

	callArgs := strings.Join(exprs, ", ")
	if co.Expr != "nil" || hasOptions(op) {
		callArgs += ", " + co.Expr
	}

	var body strings.Builder
	body.WriteString(co.Decl)
	switch shape {
	case shapeMethod:
		body.WriteString("\t\timg := coverageImage(t)\n")
		body.WriteString(fmt.Sprintf("\t\terr := img.%s(%s)\n", op.GoName, callArgs))
		body.WriteString("\t\tif err == nil {\n")
		body.WriteString("\t\t\tif _, evalErr := img.Avg(); evalErr == nil {\n")
		body.WriteString(fmt.Sprintf("\t\t\t\tt.Skipf(%q)\n", op.Name+": accepted arguments libvips should reject"))
		body.WriteString("\t\t\t}\n\t\t}\n")
	case shapeCreator:
		body.WriteString(fmt.Sprintf("\t\tout, err := New%s(%s)\n", op.GoName, callArgs))
		body.WriteString("\t\tif err == nil {\n")
		body.WriteString("\t\t\tdefer out.Close()\n")
		body.WriteString("\t\t\tif _, evalErr := out.Avg(); evalErr == nil {\n")
		body.WriteString(fmt.Sprintf("\t\t\t\tt.Skipf(%q)\n", op.Name+": accepted arguments libvips should reject"))
		body.WriteString("\t\t\t}\n\t\t}\n")
	case shapeSaver:
		body.WriteString("\t\timg := coverageImage(t)\n")
		body.WriteString(fmt.Sprintf("\t\tif err := img.%s(%s); err == nil {\n", op.GoName, callArgs))
		body.WriteString(fmt.Sprintf("\t\t\tt.Skipf(%q)\n", op.Name+": accepted arguments libvips should reject"))
		body.WriteString("\t\t}\n")
	}
	return body.String(), true
}

// hasOptions reports whether the operation has an options argument, which is what makes the
// with-options shim exist.
func hasOptions(op introspection.Operation) bool {
	return len(op.OptionalInputs) > 0 || len(getSupportedOptionalOutputs(op)) > 0
}

// numericGoType reports the Go types libvips reads as a number, where zero means "not set".
func numericGoType(goType string) bool {
	switch goType {
	case "int", "int64", "float64", "float32":
		return true
	}
	return false
}

// invalidEnumValue is outside every enum libvips defines; the largest, VipsOperationFlags,
// stops in the tens.
const invalidEnumValue = 999
