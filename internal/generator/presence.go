package generator

import (
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
func argPresence(opt introspection.Argument) Presence {
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

// generatePresenceDeclaration declares the C value and the set flag for a pointer
// option, reading both from the options struct field.
func generatePresenceDeclaration(opt introspection.Argument) string {
	goName := opt.GoName
	cName := "c" + goName
	setName := goName + "Set"

	if opt.GoType == "string" {
		return strings.Join([]string{
			"var " + cName + " *C.char",
			setName + " := C.int(0)",
			"if " + goName + " != nil {",
			"	" + cName + " = C.CString(*" + goName + ")",
			"	" + setName + " = 1",
			"	defer freeCString(" + cName + ")",
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
		"	" + cName + " = " + value,
		"	" + setName + " = 1",
		"}",
	}, "\n\t")
}
