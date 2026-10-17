package introspection

import (
	"encoding/json"
	"math"
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
	if intent.DefaultValue != 1 {
		t.Fatalf("intent default = %v, want 1", intent.DefaultValue)
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

	const (
		noMinimum = 1
		noMaximum = 2
	)
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
				missing[op.Name+"."+arg.Name] |= noMinimum
			}
			if arg.HasRange && !arg.HasMaximum {
				missing[op.Name+"."+arg.Name] |= noMaximum
			}
		}
	}

	// Arguments whose pspec declares an unbounded range; both bounds for the
	// clamp/stdif ranges, the upper bound only for find_trim.
	want := map[string]int{
		"clamp.min":           noMinimum | noMaximum,
		"clamp.max":           noMinimum | noMaximum,
		"find_trim.threshold": noMaximum,
		"stdif.m0":            noMinimum | noMaximum,
		"stdif.s0":            noMinimum | noMaximum,
	}
	for name, bits := range want {
		op := name[:strings.Index(name, ".")]
		if !discovered[op] {
			t.Logf("operation %s not available in this libvips build", op)
			continue
		}
		if missing[name] != bits {
			t.Errorf("%s missing bounds = %d, want %d", name, missing[name], bits)
		}
	}
	for name, bits := range missing {
		if _, ok := want[name]; !ok {
			t.Logf("other unbounded range reported: %s (missing bounds %d)", name, bits)
		}
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
	if keep.DefaultValue != int64(63) {
		t.Errorf("keep: default = %#v, want int64(63) (all keep flags)", keep.DefaultValue)
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
	if filter.DefaultValue != int64(8) {
		t.Errorf("filter: default = %#v, want int64(8) (VIPS_FOREIGN_PNG_FILTER_NONE)", filter.DefaultValue)
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

	// Every flags argument must report a default: that is what lets the
	// presence rule tell an explicit zero mask from an omitted one.
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
			if arg.DefaultValue == nil {
				t.Errorf("%s.%s: flags argument without a default", op.Name, arg.Name)
			}
		}
	}
	if flagsArgs == 0 {
		t.Fatal("no flags arguments discovered; the flags metadata branch is dead")
	}
	t.Logf("flags arguments reported: %d", flagsArgs)
}
