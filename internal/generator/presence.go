package generator

import (
	"math"
	"strconv"
	"strings"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// Presence describes how a generated optional argument tells libvips whether the
// caller supplied it.
//
// v1 guessed from the value: a plain field whose zero value meant "not supplied",
// except for a hand-written list of arguments where libvips accepts zero. v2
// derives the answer from GParamSpec metadata instead, and represents the
// arguments where it matters as pointers, so that nil means "not supplied" and a
// non-nil pointer means the caller gave a value, whatever that value is.
type Presence int

const (
	// PresenceValue: a plain field. The zero value is either not a value libvips
	// accepts, or is what libvips would have used anyway, so omitting it is the
	// same as not supplying it. The C wrapper still skips zero values.
	PresenceValue Presence = iota
	// PresencePointer: a *T field. The zero value is a value libvips accepts and
	// differs from the option's default, so the two cases must be distinguishable.
	PresencePointer
)

// argPresence classifies one argument.
//
// An optional input needs a pointer when both hold:
//
//   - the type's zero value is one libvips accepts: for numbers, zero is inside
//     the accepted range; for enums and flags, zero is one of the members.
//   - that zero differs from the argument's default, so passing it is a choice
//     rather than a no-op.
//
// Anything else stays a plain value: the parts of libvips' semantics that
// GParamSpec does not describe are not guessed here.
// zeroRejectedArguments are optional arguments where libvips rejects an explicit zero.
// Measured, not derived - webpload.n and gifload.n have the same range and default, and
// only the second rejects a zero. vips/zero_pins_test.go holds the classification.
var zeroRejectedArguments = map[[2]string]bool{
	{"gifload", "n"}:              true,
	{"gifload_buffer", "n"}:       true,
	{"gifload_source", "n"}:       true,
	{"tiffload", "n"}:             true,
	{"tiffload_buffer", "n"}:      true,
	{"tiffload_source", "n"}:      true,
	{"tiffload", "subifd"}:        true,
	{"tiffload_buffer", "subifd"}: true,
	{"tiffload_source", "subifd"}: true,
	{"webpload", "scale"}:         true,
	{"webpload_buffer", "scale"}:  true,
	{"webpload_source", "scale"}:  true,
	{"svgload", "scale"}:          true,
	{"svgload_buffer", "scale"}:   true,
	{"svgload_source", "scale"}:   true,
	{"heifload", "n"}:        true,
	{"heifload_buffer", "n"}: true,
	{"heifload_source", "n"}: true,
	{"pdfload", "n"}:         true,
	{"pdfload_buffer", "n"}:  true,
	{"pdfload_source", "n"}:  true,
	{"magickload", "n"}:        true,
	{"magickload_buffer", "n"}: true,
	{"magickload_source", "n"}: true,
	{"dzsave", "suffix"}:       true,
	{"dzsave_buffer", "suffix"}: true,
	{"dzsave_target", "suffix"}: true,
	{"similarity", "scale"}:    true,
}

// ApplyMeasuredZeroPolicy marks the arguments above so argPresence returns a plain
// field for them. Re-measure with internal/zeroprobe before changing this table: the
// verdicts are libvips behaviour, not vocabulary.
func ApplyMeasuredZeroPolicy(ops []introspection.Operation) {
	for oi := range ops {
		op := &ops[oi]
		// OptionalInputs holds copies, appended one at a time during introspection, so the
		// flag has to be set on that slice too - it is the slice the classifiers read.
		mark := func(args []introspection.Argument) {
			for ai := range args {
				if zeroRejectedArguments[[2]string{op.Name, args[ai].Name}] {
					args[ai].ZeroRejected = true
				}
			}
		}
		mark(op.Arguments)
		mark(op.OptionalInputs)
	}
}

func argPresence(opt introspection.Argument) Presence {
	if opt.ZeroRejected {
		return PresenceValue
	}
	if opt.IsRequired || opt.IsOutput || opt.IsArray || opt.IsImage || opt.IsSource || opt.IsTarget {
		return PresenceValue
	}
	if !zeroValueAccepted(opt) {
		return PresenceValue
	}
	if isZeroDefault(opt.DefaultValue) {
		return PresenceValue
	}
	return PresencePointer
}

// zeroValueAccepted reports whether libvips accepts the argument type's zero value.
func zeroValueAccepted(opt introspection.Argument) bool {
	switch {
	case opt.IsEnum || opt.IsFlags:
		for _, v := range opt.EnumValues {
			if v == 0 {
				return true
			}
		}
		return false
	case isNumericOption(opt):
		if opt.HasMinimum && opt.Minimum > 0 {
			return false
		}
		if opt.HasMaximum && opt.Maximum < 0 {
			return false
		}
		return true
	case opt.GoType == "bool":
		// A boolean zero (false) is always an acceptable value; it differs from
		// the default only when the default is true, which isZeroDefault checks.
		return true
	case opt.GoType == "string":
		return true
	}
	return false
}

func isNumericOption(opt introspection.Argument) bool {
	switch opt.GoType {
	case "int", "int64", "uint64", "float64", "float32":
		return true
	}
	return false
}

func isZeroDefault(v interface{}) bool {
	switch t := v.(type) {
	case nil:
		return true
	case bool:
		return !t
	case int:
		return t == 0
	case int64:
		return t == 0
	case uint64:
		return t == 0
	case float64:
		return t == 0
	case string:
		return t == ""
	}
	return false
}

// optionFieldType returns the Go type of an option's field in the options struct,
// and the type its value takes as a parameter of the generated wrapper.
func optionFieldType(opt introspection.Argument) string {
	fieldType := opt.GoType
	if opt.GoType == "*C.VipsImage" {
		fieldType = "*Image"
	} else if opt.GoType == "[]*C.VipsImage" {
		fieldType = "[]*Image"
	} else if opt.CType == "void*" {
		fieldType = "[]byte"
	}
	if opt.IsEnum && opt.EnumType != "" {
		fieldType = opt.EnumType
	}
	if argPresence(opt) == PresencePointer {
		return "*" + fieldType
	}
	return fieldType
}

// cgoOptionType names the cgo type that holds a pointer option's value.
func cgoOptionType(opt introspection.Argument) string {
	switch opt.GoType {
	case "string":
		return "*C.char"
	case "bool":
		return "C.gboolean"
	case "float64":
		return "C.double"
	case "float32":
		return "C.float"
	}
	if opt.IsEnum || opt.IsFlags {
		if t := strings.TrimSpace(opt.CType); t != "" && !strings.Contains(t, " ") {
			return "C." + t
		}
	}
	if t := strings.TrimSpace(opt.CType); t != "" && !strings.Contains(t, " ") {
		if strings.HasPrefix(t, "*") {
			return "C." + t
		}
		return "C." + t
	}
	switch opt.GoType {
	case "int", "int64":
		return "C.int"
	}
	return "C.int"
}

// presenceHelper names the generated helper that turns an optional field into a C
// value and a set flag. Anything it cannot carry falls back to the explicit form.
func presenceHelper(opt introspection.Argument) (helper string, cast bool) {
	if opt.GoType == "string" {
		// the helper hands back a *C.char, so the C type is never spelled out
		return "optionString", false
	}
	// a cast needs a C type that can be written as a cgo expression
	if strings.Contains(strings.TrimSpace(opt.CType), " ") {
		return "", false
	}
	switch {
	case opt.GoType == "float64" || opt.GoType == "float32":
		return "optionFloat", true
	case opt.IsEnum || opt.IsFlags || isNumericOption(opt):
		return "optionInt", true
	}
	return "", false
}

// generatePresenceDeclaration declares the C value and the set flag for a pointer
// option, reading both from the options struct field. The helpers keep this to the
// one line an option deserves.
func generatePresenceDeclaration(opt introspection.Argument) string {
	goName := opt.GoName
	cName := "c" + goName
	setName := goName + "Set"

	if helper, _ := presenceHelper(opt); helper != "" {
		declaration := cName + ", " + setName + " := " + helper + "(" + goName + ")"
		if helper == "optionString" {
			// freeCString accepts nil, so the copy is always released.
			return declaration + "\n\tdefer freeCString(" + cName + ")"
		}
		return declaration
	}

	if opt.GoType == "string" {
		return strings.Join([]string{
			"var " + cName + " *C.char",
			setName + " := C.int(0)",
			"if " + goName + " != nil {",
			"\t" + cName + " = C.CString(*" + goName + ")",
			"\t" + setName + " = 1",
			"\tdefer freeCString(" + cName + ")",
			"}",
		}, "\n\t")
	}

	cType := cgoOptionType(opt)
	value := cType + "(*" + goName + ")"
	if opt.GoType == "bool" {
		value = "C.gboolean(boolToInt(*" + goName + "))"
	}

	return strings.Join([]string{
		"var " + cName + " " + cType,
		setName + " := C.int(0)",
		"if " + goName + " != nil {",
		"\t" + cName + " = " + value,
		"\t" + setName + " = 1",
		"}",
	}, "\n\t")
}

// presenceCallValue renders the argument that carries a pointer option's value into
// the C call. A helper hands back a Go number, so the call casts it; the explicit
// form already holds the C type.
func presenceCallValue(opt introspection.Argument) string {
	cName := "c" + opt.GoName
	if helper, cast := presenceHelper(opt); helper != "" && cast {
		return "C." + strings.TrimSpace(opt.CType) + "(" + cName + ")"
	}
	return cName
}

// describeOptionDefault renders an option's accepted range and default for the
// generated doc comment. The generator reads both from GParamSpec anyway, for the
// presence rule, so the answer to "what happens if I leave this nil?" belongs in the
// editor rather than only in libvips' own documentation.
func describeOptionDefault(opt introspection.Argument) string {
	if opt.IsOutput {
		return ""
	}

	var parts []string
	if rangeText := describeOptionRange(opt); rangeText != "" {
		parts = append(parts, rangeText)
	}
	if defaultText := describeOptionDefaultValue(opt); defaultText != "" {
		parts = append(parts, defaultText)
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

func describeOptionRange(opt introspection.Argument) string {
	if !isNumericOption(opt) || (!opt.HasMinimum && !opt.HasMaximum) {
		return ""
	}
	switch {
	case opt.HasMinimum && opt.HasMaximum:
		return "range " + formatOptionNumber(opt.Minimum) + "-" + formatOptionNumber(opt.Maximum)
	case opt.HasMinimum:
		return "min " + formatOptionNumber(opt.Minimum)
	default:
		return "max " + formatOptionNumber(opt.Maximum)
	}
}

func describeOptionDefaultValue(opt introspection.Argument) string {
	// A plain field whose zero value is already the default does not need saying.
	if argPresence(opt) == PresenceValue && isZeroDefault(opt.DefaultValue) {
		return ""
	}
	switch v := opt.DefaultValue.(type) {
	case nil:
		return ""
	case bool:
		if !v {
			return ""
		}
		return "default true"
	case int:
		if name, ok := enumMemberName(opt, int64(v)); ok {
			return "default " + name
		}
		return "default " + formatOptionNumber(float64(v))
	case int64:
		if name, ok := enumMemberName(opt, v); ok {
			return "default " + name
		}
		return "default " + formatOptionNumber(float64(v))
	case uint64:
		if name, ok := enumMemberName(opt, int64(v)); ok {
			return "default " + name
		}
		return "default " + formatOptionNumber(float64(v))
	case float64:
		return "default " + formatOptionNumber(v)
	case string:
		if v == "" {
			return ""
		}
		return "default " + strconv.Quote(v)
	}
	return ""
}

// enumMemberName names an enum or flags default when one of its members has exactly
// that value. A bitmask that is not a single member keeps its number.
func enumMemberName(opt introspection.Argument, value int64) (string, bool) {
	if !opt.IsEnum && !opt.IsFlags {
		return "", false
	}
	name, ok := enumValueNames[opt.EnumType][int(value)]
	return name, ok
}

func formatOptionNumber(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'g', -1, 64)
}

// defaultFieldLiteral renders an option's default as a Go literal for the generated
// Default<Op>Options constructor, wrapped in Ptr for the fields that carry presence.
// It returns false when the default is the field's own zero value, which the
// struct literal already says.
func defaultFieldLiteral(opt introspection.Argument) (string, bool) {
	var value string
	switch v := opt.DefaultValue.(type) {
	case nil:
		return "", false
	case bool:
		if !v {
			return "", false
		}
		value = "true"
	case int:
		value = formatIntDefault(opt, int64(v))
	case int64:
		value = formatIntDefault(opt, v)
	case uint64:
		value = formatFloatDefault(opt, float64(v))
	case float64:
		if v == 0 {
			return "", false
		}
		value = formatFloatDefault(opt, v)
	case string:
		if v == "" {
			return "", false
		}
		value = strconv.Quote(v)
	default:
		return "", false
	}

	if argPresence(opt) == PresencePointer {
		return "Ptr(" + value + ")", true
	}
	if isZeroDefault(opt.DefaultValue) {
		return "", false
	}
	return value, true
}

func formatIntDefault(opt introspection.Argument, value int64) string {
	if name, ok := enumMemberName(opt, value); ok {
		return name
	}
	return formatFloatDefault(opt, float64(value))
}

// formatFloatDefault keeps a float field's literal typed: Ptr(100) would infer *int,
// so an integral default on a float option is written as 100.0.
func formatFloatDefault(opt introspection.Argument, value float64) string {
	text := formatOptionNumber(value)
	if (opt.GoType == "float64" || opt.GoType == "float32") && !strings.ContainsAny(text, ".eE") {
		text += ".0"
	}
	return text
}
