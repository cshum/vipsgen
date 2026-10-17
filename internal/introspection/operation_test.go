package introspection

import (
	"encoding/json"
	"math"
	"sort"
	"strings"
	"testing"
)

func TestDiscoverOperationArgumentMetadata(t *testing.T) {
	introspection := NewIntrospection(false)
	args, err := introspection.DiscoverOperationArguments("icc_transform")
	if err != nil {
		t.Fatal(err)
	}

	intent := findArgument(t, args, "intent")
	if !intent.IsEnum || intent.EnumType != "Intent" {
		t.Fatalf("intent metadata = %+v, want Intent enum", intent)
	}
	if intent.IsFlags {
		t.Errorf("intent IsFlags = true, want false: a plain enum must not look like a flags class")
	}
	// The default itself is a property of the libvips build, so only the shape
	// the presence rule reads is asserted: a non-zero default with a zero member.
	intentDefault, ok := intent.DefaultValue.(int)
	if !ok {
		t.Fatalf("intent default = %#v, want an int", intent.DefaultValue)
	}
	if intentDefault == 0 {
		t.Errorf("intent default = 0, want a non-zero default")
	}
	if !containsInt(intent.EnumValues, 0) {
		t.Fatalf("intent enum values = %v, want zero-valued member", intent.EnumValues)
	}

	blackPointCompensation := findArgument(t, args, "black_point_compensation")
	if blackPointCompensation.DefaultValue != false {
		t.Fatalf("black point compensation default = %v, want false", blackPointCompensation.DefaultValue)
	}

	inputProfile := findArgument(t, args, "input_profile")
	if inputProfile.HasRange {
		t.Fatalf("string input_profile unexpectedly has numeric range: %+v", inputProfile)
	}

	thumbnailArgs, err := introspection.DiscoverOperationArguments("thumbnail")
	if err != nil {
		t.Fatal(err)
	}
	thumbnailHeight := findArgument(t, thumbnailArgs, "height")
	if !thumbnailHeight.HasRange {
		t.Fatal("thumbnail height has no numeric range metadata")
	}
	if thumbnailHeight.Maximum < thumbnailHeight.Minimum {
		t.Fatalf("thumbnail height range = [%v, %v]", thumbnailHeight.Minimum, thumbnailHeight.Maximum)
	}
}

func findArgument(t *testing.T, args []Argument, name string) Argument {
	t.Helper()
	for _, arg := range args {
		if arg.Name == name {
			return arg
		}
	}
	t.Fatalf("argument %q not found", name)
	return Argument{}
}

func containsInt(values []int, wanted int) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

// TestOperationBoundsAreFiniteOrUnbounded guards the -debug dump. GParamSpec
// bounds can be infinite, and a single infinite value aborted the JSON marshal
// for every operation, so -debug silently stopped writing the dump at all.
// Bounds are therefore reported as finite-or-absent: the value is only
// meaningful when HasMinimum/HasMaximum says the bound was declared.
func TestOperationBoundsAreFiniteOrUnbounded(t *testing.T) {
	introspection := NewIntrospection(false)
	operations := introspection.DiscoverOperations()
	if len(operations) == 0 {
		t.Fatal("no operations discovered")
	}

	missing := map[string]int{}
	discovered := map[string]bool{}
	for _, op := range operations {
		discovered[op.Name] = true
		// Arguments holds required inputs and outputs; optional inputs live in
		// their own slice, and the unbounded ranges are all optional.
		args := append(append([]Argument{}, op.Arguments...), op.OptionalInputs...)
		for _, arg := range args {
			if math.IsInf(arg.Minimum, 0) || math.IsInf(arg.Maximum, 0) ||
				math.IsNaN(arg.Minimum) || math.IsNaN(arg.Maximum) {
				t.Errorf("%s.%s has a non-finite bound: min=%v max=%v", op.Name, arg.Name, arg.Minimum, arg.Maximum)
			}
			if arg.HasRange && !arg.HasMinimum {
				missing[op.Name+"."+arg.Name] |= 1
			}
			if arg.HasRange && !arg.HasMaximum {
				missing[op.Name+"."+arg.Name] |= 2
			}
		}
	}

	// Which arguments have unbounded ranges is a property of the libvips build,
	// not of this code, so it is reported rather than asserted: pinning it here
	// would fail the older libvips versions in the CI matrix.
	if len(missing) > 0 {
		names := make([]string, 0, len(missing))
		for name := range missing {
			names = append(names, name)
		}
		sort.Strings(names)
		t.Logf("arguments with an unbounded range: %s", strings.Join(names, ", "))
	}

	// The actual regression: the operations must marshal for the dump.
	if _, err := json.Marshal(operations); err != nil {
		t.Fatalf("marshalling operations for the -debug dump failed: %v", err)
	}
}

// TestFlagsAnd64BitArgumentMetadata covers the two argument classes that used to
// carry no metadata at all: GFlags bitmasks (keep, filter) and 64-bit integers.
// Both are inputs to the option-presence rule, which cannot classify a field
// without knowing its default.
//
// The assertions are deliberately version-independent: the exact masks differ
// between libvips releases (VIPS_FOREIGN_KEEP_ALL is 31 before gainmap support
// and 63 from 8.18), and pinning those numbers here would fail the older
// versions in the CI matrix. What the rule needs is stable: a non-zero default
// that is itself a member, and whether zero is a member.
func TestFlagsAnd64BitArgumentMetadata(t *testing.T) {
	introspection := NewIntrospection(false)

	keepArgs, err := introspection.DiscoverOperationArguments("jpegsave")
	if err != nil {
		t.Fatal(err)
	}
	keep := findArgument(t, keepArgs, "keep")
	if !keep.IsFlags || !keep.IsEnum {
		t.Fatalf("keep: IsFlags=%v IsEnum=%v, want both", keep.IsFlags, keep.IsEnum)
	}
	if keep.HasRange {
		t.Errorf("keep: HasRange=%v, want false (a flags mask has no numeric range)", keep.HasRange)
	}
	keepDefault, ok := keep.DefaultValue.(int64)
	if !ok {
		t.Fatalf("keep: default = %#v, want an int64 mask", keep.DefaultValue)
	}
	if keepDefault == 0 {
		t.Errorf("keep: default = %d, want a non-zero all-flags mask", keepDefault)
	}
	if !containsInt(keep.EnumValues, 0) {
		t.Errorf("keep: members = %v, want VIPS_FOREIGN_KEEP_NONE(0) among them", keep.EnumValues)
	}

	filterArgs, err := introspection.DiscoverOperationArguments("pngsave")
	if err != nil {
		t.Fatal(err)
	}
	filter := findArgument(t, filterArgs, "filter")
	if !filter.IsFlags {
		t.Fatalf("filter: IsFlags=%v, want true", filter.IsFlags)
	}
	if _, ok := filter.DefaultValue.(int64); !ok {
		t.Fatalf("filter: default = %#v, want an int64 mask", filter.DefaultValue)
	}
	if containsInt(filter.EnumValues, 0) {
		t.Errorf("filter: members = %v, want no zero member", filter.EnumValues)
	}

	offsetArgs, err := introspection.DiscoverOperationArguments("rawload")
	if err != nil {
		t.Fatal(err)
	}
	offset := findArgument(t, offsetArgs, "offset")
	if offset.GoType != "uint64" {
		t.Fatalf("offset: GoType = %q, want uint64", offset.GoType)
	}
	if !offset.HasRange || !offset.HasMinimum || !offset.HasMaximum {
		t.Errorf("offset: range metadata = %+v, want a bounded range", offset)
	}
	if offset.DefaultValue != int64(0) {
		t.Errorf("offset: default = %#v, want int64(0)", offset.DefaultValue)
	}

	// Every flags argument must report a default that is one of its own
	// members: that is what lets the presence rule tell an explicit zero mask
	// from an omitted one, and a default outside the class would be a
	// metadata bug that classification cannot survive.
	operations := introspection.DiscoverOperations()
	flagsArgs := 0
	for _, op := range operations {
		args := append(append([]Argument{}, op.Arguments...), op.OptionalInputs...)
		for _, arg := range args {
			if !arg.IsFlags {
				continue
			}
			flagsArgs++
			if !arg.IsEnum {
				t.Errorf("%s.%s: IsFlags but not IsEnum", op.Name, arg.Name)
			}
			mask, ok := arg.DefaultValue.(int64)
			if !ok {
				t.Errorf("%s.%s: flags argument default = %#v, want an int64 mask", op.Name, arg.Name, arg.DefaultValue)
				continue
			}
			if !containsInt(arg.EnumValues, int(mask)) {
				t.Errorf("%s.%s: default %d is not one of the flag members %v", op.Name, arg.Name, mask, arg.EnumValues)
			}
		}
	}
	if flagsArgs == 0 {
		t.Fatal("no flags arguments discovered; the flags metadata branch is dead")
	}
	t.Logf("flags arguments reported: %d", flagsArgs)
}
