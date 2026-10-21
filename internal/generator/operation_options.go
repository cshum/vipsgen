package generator

import (
	"strings"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// coverageOptions is the option argument of a generated call: a nil the operation
// defaults from, or a literal with every field this generator can give a value to.
type coverageOptions struct {
	Decl string // prelude statement, empty for nil
	Expr string // the argument itself: "nil" or "opts"
	Skip string // reason to report instead of failing, for this variant only
}

// skipReason is the reason to report when the call errors: the variant's own if it
// has one, otherwise the operation's.
func (co coverageOptions) skipReason(opName string) string {
	if co.Skip != "" {
		return co.Skip
	}
	return coverageSkip[opName]
}

// nilCoverageOptions passes no options at all.
func nilCoverageOptions() coverageOptions {
	return coverageOptions{Expr: "nil"}
}

// coverageOptionsLiteral builds an options literal from the argument metadata:
// every optional field becomes a value from the same range and enum rules the
// positional arguments use, wrapped in Ptr when the field's zero is a value
// libvips accepts - which is exactly the presence rule the generator applies.
//
// Fields whose type this generator cannot make up are left nil: strings (a
// profile name, a colour descriptor), images, interpolators and image slices.
// Leaving them unset is honest - the subtest then exercises the fields it did
// set - and it keeps the literal readable.
func coverageOptionsLiteral(op introspection.Operation) (coverageOptions, bool) {
	if len(op.OptionalInputs) == 0 {
		return coverageOptions{}, false
	}

	var fields []string
	for _, opt := range op.OptionalInputs {
		if !settableOption(opt) {
			continue
		}
		value, ok := scalarValue(opt)
		if !ok {
			continue
		}
		if argPresence(opt) == PresencePointer {
			// Ptr(1) infers int, which does not fit a *float64 field: a float
			// value has to carry its point into the literal.
			if opt.GoType == "float64" && !strings.Contains(value, ".") {
				value += ".0"
			}
			value = "Ptr(" + value + ")"
		}
		fields = append(fields, strings.Title(opt.GoName)+": "+value)
	}
	if len(fields) == 0 {
		return coverageOptions{}, false
	}

	return coverageOptions{
		Decl: "\t\topts := &" + op.GoName + "Options{" + strings.Join(fields, ", ") + "}\n",
		Expr: "opts",
	}, true
}

// settableOption reports whether an optional argument's value can be made up.
func settableOption(opt introspection.Argument) bool {
	if opt.IsEnum {
		return true
	}
	switch opt.GoType {
	case "int", "float64", "bool", "[]float64", "[]int":
		return true
	}
	return false
}
