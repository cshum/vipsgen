package introspection

import "testing"

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
