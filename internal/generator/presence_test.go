package generator

import (
	"strings"
	"testing"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// TestArgPresenceFollowsMetadata checks the v2 rule against the metadata of every
// operation the installed libvips exposes, and pins the invariants that make the
// pointer fields safe:
//
//   - a pointer field's type zero is a value libvips accepts, so Ptr(0) is a
//     meaningful request rather than an error;
//   - a pointer field's zero differs from the argument's default, so the two cases
//     the field exists to tell apart really are different;
//   - nothing that carries a pointer, image, array, source or target becomes a
//     pointer, because presence for those has no separate meaning.
//
// Values from libvips are not pinned, so a libvips upgrade shows up as different
// counts rather than a failure.
func TestArgPresenceFollowsMetadata(t *testing.T) {
	i := introspection.NewIntrospection(false)
	operations := i.DiscoverOperations()
	if len(operations) == 0 {
		t.Fatal("no operations discovered")
	}

	var pointer, value int
	for _, op := range operations {
		for _, opt := range op.OptionalInputs {
			switch argPresence(opt) {
			case PresencePointer:
				pointer++
				if !zeroValueAccepted(opt) {
					t.Errorf("%s.%s is a pointer but its zero value is not one libvips accepts", op.Name, opt.Name)
				}
				if isZeroDefault(opt.DefaultValue) {
					t.Errorf("%s.%s is a pointer but its zero value equals its default", op.Name, opt.Name)
				}
				if opt.IsArray || opt.IsImage || opt.IsSource || opt.IsTarget || opt.IsOutput {
					t.Errorf("%s.%s is a pointer but does not hold a scalar", op.Name, opt.Name)
				}
			case PresenceValue:
				value++
			}
		}
	}
	if pointer == 0 {
		t.Fatal("no pointer optional inputs found; the rule matched nothing")
	}
	if value == 0 {
		t.Fatal("no value optional inputs found; the rule matched everything")
	}
	t.Logf("optional inputs: %d pointer, %d value", pointer, value)
}

// TestArgPresenceSpotsTheZeroValueCases checks the arguments where v1 needed a
// hand-written exception, on operations present in every supported libvips.
func TestArgPresenceSpotsTheZeroValueCases(t *testing.T) {
	i := introspection.NewIntrospection(false)
	operations := i.DiscoverOperations()

	find := func(opName, argName string) (introspection.Argument, bool) {
		for _, op := range operations {
			if op.Name != opName {
				continue
			}
			for _, opt := range op.OptionalInputs {
				if opt.Name == argName {
					return opt, true
				}
			}
		}
		return introspection.Argument{}, false
	}

	// libvips accepts 0 for these, and their defaults are non-zero, so they have to
	// be pointers: this is exactly what the v1 allowlist was compensating for.
	for _, want := range []struct{ op, arg string }{
		{"webpsave_buffer", "effort"},
		{"pngsave_buffer", "compression"},
		{"thumbnail", "intent"},
	} {
		opt, ok := find(want.op, want.arg)
		if !ok {
			t.Fatalf("%s.%s not found; libvips changed", want.op, want.arg)
		}
		if argPresence(opt) != PresencePointer {
			t.Errorf("%s.%s should be a pointer (zero is a value, default is %v)", want.op, want.arg, opt.DefaultValue)
		}
	}

	// A boolean whose default is false has no ambiguity: false is the default.
	if opt, ok := find("thumbnail", "no_rotate"); ok {
		if argPresence(opt) != PresenceValue {
			t.Errorf("thumbnail.no_rotate should stay a value (default %v)", opt.DefaultValue)
		}
	}
	// A string with an empty default has no ambiguity either.
	if opt, ok := find("pngsave", "profile"); ok {
		if argPresence(opt) != PresenceValue {
			t.Errorf("pngsave.profile should stay a value (default %q)", opt.DefaultValue)
		}
	}
}

// TestDescribeOptionDefault checks the annotation that tells a reader what happens
// when an option is left nil, and what values libvips accepts.
func TestDescribeOptionDefault(t *testing.T) {
	// the enum index is per-run state, filled by NewTemplateData
	NewTemplateData("test", nil, []introspection.EnumTypeInfo{{
		CName:  "VipsIntent",
		GoName: "Intent",
		Values: []introspection.EnumValueInfo{
			{CName: "VIPS_INTENT_PERCEPTUAL", GoName: "IntentPerceptual", Value: 0},
			{CName: "VIPS_INTENT_RELATIVE", GoName: "IntentRelative", Value: 1},
		},
	}}, nil, false)

	cases := []struct {
		name string
		arg  introspection.Argument
		want string
	}{
		{
			name: "pointer numeric with a range",
			arg: introspection.Argument{Name: "compression", GoType: "int", CType: "int",
				HasMinimum: true, Minimum: 0, HasMaximum: true, Maximum: 9, DefaultValue: 6},
			want: " (range 0-9, default 6)",
		},
		{
			name: "plain numeric outside the range still states its default",
			arg: introspection.Argument{Name: "bitdepth", GoType: "int", CType: "int",
				HasMinimum: true, Minimum: 8, HasMaximum: true, Maximum: 16, DefaultValue: 8},
			want: " (range 8-16, default 8)",
		},
		{
			name: "enum default by member name",
			arg: introspection.Argument{Name: "intent", GoType: "Intent", CType: "VipsIntent", IsEnum: true,
				EnumType: "Intent", EnumValues: []int{0, 1}, DefaultValue: int64(1)},
			want: " (default IntentRelative)",
		},
		{
			name: "unbounded numeric",
			arg: introspection.Argument{Name: "effort", GoType: "int", CType: "int", DefaultValue: 4},
			want: " (default 4)",
		},
		{
			name: "plain field whose zero is the default",
			arg: introspection.Argument{Name: "page_height", GoType: "int", CType: "int", DefaultValue: 0},
			want: "",
		},
		{
			name: "string default",
			arg: introspection.Argument{Name: "separator", GoType: "string", CType: "char*", DefaultValue: ";"},
			want: ` (default ";")`,
		},
		{
			name: "output capture",
			arg: introspection.Argument{Name: "x", GoType: "int", CType: "int*", IsOutput: true},
			want: "",
		},
	}

	for _, tc := range cases {
		if got := describeOptionDefault(tc.arg); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestDefaultFieldLiteral checks what the Default<Op>Options constructors fill in:
// pointer fields wrapped in Ptr, enums named rather than numbered, and nothing for a
// field whose zero value is already the default.
func TestDefaultFieldLiteral(t *testing.T) {
	NewTemplateData("test", nil, []introspection.EnumTypeInfo{
		{
			CName:  "VipsIntent",
			GoName: "Intent",
			Values: []introspection.EnumValueInfo{
				{CName: "VIPS_INTENT_PERCEPTUAL", GoName: "IntentPerceptual", Value: 0},
				{CName: "VIPS_INTENT_RELATIVE", GoName: "IntentRelative", Value: 1},
			},
		},
		{
			CName:  "VipsForeignKeep",
			GoName: "Keep",
			Values: []introspection.EnumValueInfo{
				{CName: "VIPS_FOREIGN_KEEP_NONE", GoName: "KeepNone", Value: 0},
				{CName: "VIPS_FOREIGN_KEEP_EXIF", GoName: "KeepExif", Value: 1},
				{CName: "VIPS_FOREIGN_KEEP_XMP", GoName: "KeepXmp", Value: 2},
			},
		},
	}, nil, false)

	cases := []struct {
		name  string
		arg   introspection.Argument
		want  string
		found bool
	}{
		{
			name:  "numeric pointer",
			arg:   introspection.Argument{Name: "compression", GoType: "int", CType: "int", DefaultValue: 6},
			want:  "Ptr(6)",
			found: true,
		},
		{
			name:  "enum pointer named",
			arg:   introspection.Argument{Name: "intent", GoType: "Intent", CType: "VipsIntent", IsEnum: true, EnumType: "Intent", EnumValues: []int{0, 1}, DefaultValue: int64(1)},
			want:  "Ptr(IntentRelative)",
			found: true,
		},
		{
			name:  "flags pointer named",
			arg:   introspection.Argument{Name: "keep", GoType: "Keep", CType: "VipsForeignKeep", IsEnum: true, IsFlags: true, EnumType: "Keep", EnumValues: []int{0, 1, 2}, DefaultValue: int64(2)},
			want:  "Ptr(KeepXmp)",
			found: true,
		},
		{
			name:  "plain field with a non-zero default",
			arg:   introspection.Argument{Name: "bitdepth", GoType: "int", CType: "int", HasMinimum: true, Minimum: 8, HasMaximum: true, Maximum: 16, DefaultValue: 8},
			want:  "8",
			found: true,
		},
		{
			name:  "plain field whose zero value is the default",
			arg:   introspection.Argument{Name: "page", GoType: "int", CType: "int", DefaultValue: 0},
			found: false,
		},
		{
			name:  "output capture",
			arg:   introspection.Argument{Name: "x", GoType: "int", CType: "int*", IsOutput: true},
			found: false,
		},
	}

	for _, tc := range cases {
		got, ok := defaultFieldLiteral(tc.arg)
		if ok != tc.found || (ok && got != tc.want) {
			t.Errorf("%s: got (%q, %v), want (%q, %v)", tc.name, got, ok, tc.want, tc.found)
		}
	}
}

// TestPointerOptionsFitTheHelpers pins the assumption the optionInt/optionFloat/
// optionString helpers rest on: every pointer option is carried losslessly by an
// int64, a float64 or a C string. A new libvips option outside that set fails here
// rather than being truncated silently.
func TestPointerOptionsFitTheHelpers(t *testing.T) {
	i := introspection.NewIntrospection(false)
	operations := i.DiscoverOperations()

	var narrowed, explicit int
	for _, op := range operations {
		for _, opt := range op.OptionalInputs {
			if argPresence(opt) != PresencePointer {
				continue
			}
			helper, _ := presenceHelper(opt)
			switch helper {
			case "optionInt":
				narrowed++
				if !opt.IsEnum && !opt.IsFlags && !isNumericOption(opt) {
					t.Errorf("%s.%s uses optionInt but its Go type is %s", op.Name, opt.Name, opt.GoType)
				}
				if strings.TrimSpace(opt.CType) != "gint" && !opt.IsEnum && !opt.IsFlags {
					t.Errorf("%s.%s uses optionInt for C type %q, which may not fit an int64", op.Name, opt.Name, opt.CType)
				}
			case "optionFloat":
				narrowed++
				if opt.GoType != "float64" && opt.GoType != "float32" {
					t.Errorf("%s.%s uses optionFloat but its Go type is %s", op.Name, opt.Name, opt.GoType)
				}
			case "optionString":
				narrowed++
				if opt.GoType != "string" {
					t.Errorf("%s.%s uses optionString but its Go type is %s", op.Name, opt.Name, opt.GoType)
				}
			default:
				explicit++
				t.Logf("%s.%s falls back to the explicit form (C type %q, Go type %s)", op.Name, opt.Name, opt.CType, opt.GoType)
			}
		}
	}
	if narrowed == 0 {
		t.Fatal("no pointer options used a helper")
	}
	t.Logf("pointer options: %d through helpers, %d explicit", narrowed, explicit)
}

// TestGeneratePresenceDeclarationShape checks that an option costs one line, and that
// a string option also releases its C copy.
func TestGeneratePresenceDeclarationShape(t *testing.T) {
	cases := []struct {
		name string
		opt  introspection.Argument
		want string
	}{
		{
			name: "enum",
			opt:  introspection.Argument{Name: "intent", GoName: "Intent", GoType: "Intent", CType: "VipsIntent", IsEnum: true},
			want: "cIntent, IntentSet := optionInt(Intent)",
		},
		{
			name: "integer",
			opt:  introspection.Argument{Name: "effort", GoName: "Effort", GoType: "int", CType: "gint"},
			want: "cEffort, EffortSet := optionInt(Effort)",
		},
		{
			name: "float",
			opt:  introspection.Argument{Name: "gap", GoName: "Gap", GoType: "float64", CType: "double"},
			want: "cGap, GapSet := optionFloat(Gap)",
		},
		{
			name: "string",
			opt:  introspection.Argument{Name: "profile", GoName: "Profile", GoType: "string", CType: "const char*"},
			want: "cProfile, ProfileSet := optionString(Profile)\n\tdefer freeCString(cProfile)",
		},
		{
			// no libvips option is a pointer boolean today, so this is the fallback the
			// helpers do not cover - a case that must still generate correct code
			name: "boolean falls back to the explicit form",
			opt:  introspection.Argument{Name: "linear", GoName: "Linear", GoType: "bool", CType: "gboolean"},
			want: "var cLinear C.gboolean\n\tLinearSet := C.int(0)\n\tif Linear != nil {\n\t\tcLinear = C.gboolean(boolToInt(*Linear))\n\t\tLinearSet = 1\n\t}",
		},
	}
	for _, tc := range cases {
		if got := generatePresenceDeclaration(tc.opt); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestPresenceCallValueCastsHelperResults checks that a helper's Go number is cast to
// the option's C type at the call, while the explicit form already holds it.
func TestPresenceCallValueCastsHelperResults(t *testing.T) {
	if got := presenceCallValue(introspection.Argument{Name: "gap", GoName: "Gap", GoType: "float64", CType: "double"}); got != "C.double(cGap)" {
		t.Errorf("float: got %q", got)
	}
	if got := presenceCallValue(introspection.Argument{Name: "profile", GoName: "Profile", GoType: "string", CType: "const char*"}); got != "cProfile" {
		t.Errorf("string: got %q", got)
	}
	if got := presenceCallValue(introspection.Argument{Name: "flag", GoName: "Flag", GoType: "bool", CType: "gboolean"}); got != "cFlag" {
		t.Errorf("explicit form: got %q", got)
	}
}

// TestNoPlainOptionDropsAMeaningfulZero is the invariant the whole v2 change exists
// to establish: a plain (non-pointer) option must be one where libvips' zero value is
// either not a value it accepts, or is exactly the option's default. If some plain
// option accepted zero and defaulted elsewhere, leaving the field out would silently
// apply a different value than the caller asked for - the v1 bug class.
//
// Options whose metadata records no default are reported separately: for those the
// generator cannot know, and they are the honest limit of a metadata-driven rule.
func TestNoPlainOptionDropsAMeaningfulZero(t *testing.T) {
	i := introspection.NewIntrospection(false)
	operations := i.DiscoverOperations()

	var checked, unknownDefault int
	var unknown []string
	for _, op := range operations {
		for _, opt := range op.OptionalInputs {
			if argPresence(opt) != PresenceValue {
				continue
			}
			if opt.DefaultValue == nil {
				if zeroValueAccepted(opt) {
					unknownDefault++
					unknown = append(unknown, op.Name+"."+opt.Name)
					// A string with no recorded default is the empty string, which is
					// what "not supplied" means anyway. Anything else would be a gap in
					// the metadata the rule leans on.
					if opt.GoType != "string" {
						t.Errorf("%s.%s has no recorded default but is a %s: skipping zero would guess", op.Name, opt.Name, opt.GoType)
					}
				}
				continue
			}
			checked++
			if zeroValueAccepted(opt) && !isZeroDefault(opt.DefaultValue) {
				t.Errorf("%s.%s is a plain option, libvips accepts zero, but its default is %v: an omitted zero would be dropped",
					op.Name, opt.Name, opt.DefaultValue)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no plain options with a recorded default were checked")
	}
	t.Logf("plain options checked against their default: %d", checked)
	if unknownDefault > 0 {
		t.Logf("plain options with no recorded default, so skipping zero is a guess: %d (%v)", unknownDefault, unknown)
	}
}
