package generator

import (
	"fmt"
	"strings"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// errorCaseBody renders a subtest that calls an operation with arguments libvips rejects: a
// required number at zero, or a required enum outside its range. libvips reports both as
// "parameter not set", so the call fails, and the failure is the only way to execute the
// error return that every generated wrapper has.
//
// An operation that accepts them anyway, or fails only when lazily evaluated, has nothing to
// show here and skips: the point is the wrapper's own error path, not libvips's.
func errorCaseBody(op introspection.Operation) (string, bool) {
	if !op.HasThisImageInput || !op.HasOneImageOutput {
		return "", false
	}

	var exprs []string
	rejected := false
	for _, arg := range callArguments(op) {
		switch {
		case arg.GoType == "*C.VipsImage" || arg.GoType == "[]*C.VipsImage":
			// A second image is a different shape, and there is no invalid one to pass.
			return "", false
		case arg.IsEnum || arg.IsFlags:
			exprs = append(exprs, invalidEnumExpr(arg))
			rejected = true
		case numericGoType(arg.GoType):
			exprs = append(exprs, "0")
			rejected = true
		default:
			return "", false
		}
	}
	if !rejected {
		return "", false
	}

	callArgs := strings.Join(exprs, ", ")
	if len(op.OptionalInputs) > 0 || len(getSupportedOptionalOutputs(op)) > 0 {
		callArgs += ", nil"
	}

	var body strings.Builder
	body.WriteString("\t\timg := coverageImage(t)\n")
	body.WriteString(fmt.Sprintf("\t\terr := img.%s(%s)\n", op.GoName, callArgs))
	body.WriteString("\t\tif err == nil {\n")
	body.WriteString("\t\t\tif _, evalErr := img.Avg(); evalErr == nil {\n")
	body.WriteString(fmt.Sprintf("\t\t\t\tt.Skipf(%q)\n", op.Name+": accepted arguments libvips should reject"))
	body.WriteString("\t\t\t}\n")
	body.WriteString("\t\t}\n")
	return body.String(), true
}

// numericGoType reports the Go types libvips reads as a number, where zero means "not set".
func numericGoType(goType string) bool {
	switch goType {
	case "int", "int64", "float64", "float32":
		return true
	}
	return false
}

// invalidEnumExpr names a member the enum does not have, which libvips treats as unset.
func invalidEnumExpr(arg introspection.Argument) string {
	return fmt.Sprintf("%s(%d)", arg.GoType, invalidEnumValue)
}

// invalidEnumValue is outside every enum libvips defines; the largest, VipsOperationFlags,
// stops in the tens.
const invalidEnumValue = 999
