package generator

import (
	"strings"
	"testing"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// loadTestArg is a loader argument as the generator reads it: only Name and
// GoType (plus EnumType for an enum) matter to load options.
func loadTestArg(name, goType, enumType string) introspection.Argument {
	return introspection.Argument{
		Name:     name,
		GoType:   goType,
		EnumType: enumType,
		IsEnum:   enumType != "",
	}
}

// loadTestOps covers every option in the policy, plus a saver whose options must
// not leak into LoadOptions.
func loadTestOps() []introspection.Operation {
	return []introspection.Operation{
		{Name: "jpegload", OptionalInputs: []introspection.Argument{
			loadTestArg("shrink", "int", ""),
			loadTestArg("autorotate", "bool", ""),
			loadTestArg("unlimited", "bool", ""),
			loadTestArg("memory", "bool", ""),
			loadTestArg("access", "Access", "Access"),
			loadTestArg("fail_on", "FailOn", "FailOn"),
		}},
		{Name: "heifload", OptionalInputs: []introspection.Argument{
			loadTestArg("thumbnail", "bool", ""),
		}},
		{Name: "pdfload", OptionalInputs: []introspection.Argument{
			loadTestArg("n", "int", ""),
			loadTestArg("page", "int", ""),
			loadTestArg("dpi", "float64", ""),
		}},
		{Name: "jpegsave", OptionalInputs: []introspection.Argument{
			loadTestArg("strip", "bool", ""),
		}},
	}
}

func loadTestEnums() []introspection.EnumTypeInfo {
	return []introspection.EnumTypeInfo{
		{
			CName:  "VipsFailOn",
			GoName: "FailOn",
			Values: []introspection.EnumValueInfo{
				{CName: "VIPS_FAIL_ON_NONE", GoName: "FailOnNone", Value: 0, Description: "none"},
				{CName: "VIPS_FAIL_ON_TRUNCATED", GoName: "FailOnTruncated", Value: 1, Description: "truncated"},
				{CName: "VIPS_FAIL_ON_ERROR", GoName: "FailOnError", Value: 2, Description: "error"},
				{CName: "VIPS_FAIL_ON_WARNING", GoName: "FailOnWarning", Value: 3, Description: "warning"},
				// 8.16 and 8.17 register the terminal marker, 8.18 does not.
				{CName: "VIPS_FAIL_ON_LAST", GoName: "FailOnLast", Value: 4, Description: "last"},
			},
		},
		{
			CName:  "VipsAccess",
			GoName: "Access",
			Values: []introspection.EnumValueInfo{
				{CName: "VIPS_ACCESS_RANDOM", GoName: "AccessRandom", Value: 0, Description: "random"},
				{CName: "VIPS_ACCESS_SEQUENTIAL", GoName: "AccessSequential", Value: 1, Description: "sequential"},
				{CName: "VIPS_ACCESS_SEQUENTIAL_UNBUFFERED", GoName: "AccessSequentialUnbuffered", Value: 2, Description: "sequential-unbuffered"},
				{CName: "VIPS_ACCESS_LAST", GoName: "AccessLast", Value: 3, Description: "last"},
			},
		},
	}
}

func buildTestLoadOptions(t *testing.T) *LoadOptionsTemplate {
	t.Helper()
	loadOptions, err := BuildLoadOptions(loadTestOps(), loadTestEnums())
	if err != nil {
		t.Fatalf("BuildLoadOptions: %v", err)
	}
	return loadOptions
}

// TestBuildLoadOptionsResolvesTypes pins where the Go type comes from: the
// introspected argument, unless the policy states one because v1 chose another.
func TestBuildLoadOptionsResolvesTypes(t *testing.T) {
	loadOptions := buildTestLoadOptions(t)

	// Dpi keeps v1's int even though every loader reports gdouble.
	byName := map[string]LoadOptionField{}
	for _, field := range loadOptions.Fields {
		byName[field.GoName] = field
	}
	if got := byName["Dpi"]; got.GoType != "int" || got.Kind != LoadOptionInt {
		t.Errorf("Dpi = %s/%d, want int/LoadOptionInt", got.GoType, got.Kind)
	}
	if got := byName["FailOn"]; got.Kind != LoadOptionEnum || got.ZeroName != "FailOnNone" || len(got.Cases) != 3 {
		t.Errorf("FailOn = %+v, want enum with zero FailOnNone and 3 cases", got)
	}
	if got := byName["Access"]; got.ZeroName != "AccessRandom" || len(got.Cases) != 2 {
		t.Errorf("Access = %+v, want zero AccessRandom and 2 cases", got)
	}
	// fail_on_error has no introspected argument at all: the deprecated "fail"
	// boolean is filtered out of introspection, so the row states its own name,
	// type and default.
	if got := byName["FailOnError"]; got.Option != "fail" || got.Kind != LoadOptionBool || got.Default != "true" {
		t.Errorf("FailOnError = %+v, want bool option fail defaulting to true", got)
	}
	if _, ok := byName["Strip"]; ok {
		t.Error("a saver option reached LoadOptions")
	}
	// The _LAST marker is registered on 8.16 and 8.17 but not 8.18: spelling it
	// would make the option string depend on the libvips the package was built
	// from, and no loader accepts it.
	for _, field := range loadOptions.Fields {
		for _, enumCase := range field.Cases {
			if enumCase.GoName == "FailOnLast" || enumCase.GoName == "AccessLast" {
				t.Errorf("%s: the %s sentinel reached the option string", field.GoName, enumCase.GoName)
			}
		}
	}
	for _, field := range loadOptions.Fields {
		if field.Emit == 0 {
			t.Errorf("%s is never emitted into the option string", field.GoName)
		}
	}
}

// TestLoadOptionSpecsAreConsistent keeps the policy table honest: every row
// carries documentation, and Emit numbers are unique and contiguous so the
// option string has one order rather than an accident of the slice.
func TestLoadOptionSpecsAreConsistent(t *testing.T) {
	seen := map[int]string{}
	for _, spec := range loadOptionSpecs {
		if len(spec.Doc) == 0 {
			t.Errorf("%s has no doc comment", spec.GoName)
		}
		if spec.Emit <= 0 {
			t.Errorf("%s has no emit position", spec.GoName)
			continue
		}
		if other, ok := seen[spec.Emit]; ok {
			t.Errorf("%s and %s share emit position %d", other, spec.GoName, spec.Emit)
		}
		seen[spec.Emit] = spec.GoName
	}
	for position := 1; position <= len(loadOptionSpecs); position++ {
		if _, ok := seen[position]; !ok {
			t.Errorf("emit position %d is unused", position)
		}
	}
}

// TestResolveLoadOptionsErrors pins that the policy cannot silently drop an
// option: an option libvips no longer reports, loaders that disagree, and an
// enum value without a nick all stop the build.
func TestResolveLoadOptionsErrors(t *testing.T) {
	t.Run("unknown option", func(t *testing.T) {
		_, err := resolveLoadOptions([]loadOptionSpec{{
			GoName:  "Bogus",
			ArgName: "bogus",
			Emit:    1,
		}}, loadTestOps(), loadTestEnums())
		if err == nil || !strings.Contains(err.Error(), "not on any loader operation") {
			t.Fatalf("err = %v, want unknown option error", err)
		}
	})

	t.Run("saver only option", func(t *testing.T) {
		_, err := resolveLoadOptions([]loadOptionSpec{{
			GoName:  "Strip",
			ArgName: "strip",
			Emit:    1,
		}}, loadTestOps(), loadTestEnums())
		if err == nil || !strings.Contains(err.Error(), "not on any loader operation") {
			t.Fatalf("err = %v, want saver option rejected", err)
		}
	})

	t.Run("loaders disagree", func(t *testing.T) {
		ops := []introspection.Operation{
			{Name: "pngload", OptionalInputs: []introspection.Argument{loadTestArg("speed", "int", "")}},
			{Name: "tiffload", OptionalInputs: []introspection.Argument{loadTestArg("speed", "float64", "")}},
		}
		_, err := resolveLoadOptions([]loadOptionSpec{{
			GoName:  "Speed",
			ArgName: "speed",
			Emit:    1,
		}}, ops, loadTestEnums())
		if err == nil || !strings.Contains(err.Error(), "loaders disagree") {
			t.Fatalf("err = %v, want type disagreement error", err)
		}
	})

	t.Run("enum without nick", func(t *testing.T) {
		enums := []introspection.EnumTypeInfo{{
			GoName: "Access",
			Values: []introspection.EnumValueInfo{
				{GoName: "AccessRandom", Value: 0, Description: "random"},
				{GoName: "AccessSequential", Value: 1},
			},
		}}
		_, err := resolveLoadOptions([]loadOptionSpec{{
			GoName:  "Access",
			ArgName: "access",
			Emit:    1,
		}}, loadTestOps(), enums)
		if err == nil || !strings.Contains(err.Error(), "has no libvips nick") {
			t.Fatalf("err = %v, want missing nick error", err)
		}
	})

	t.Run("unknown enum type", func(t *testing.T) {
		ops := []introspection.Operation{
			{Name: "pngload", OptionalInputs: []introspection.Argument{loadTestArg("mode", "NotAnEnum", "NotAnEnum")}},
		}
		_, err := resolveLoadOptions([]loadOptionSpec{{
			GoName:  "Mode",
			ArgName: "mode",
			Emit:    1,
		}}, ops, loadTestEnums())
		if err == nil || !strings.Contains(err.Error(), "neither int, bool nor a libvips enum") {
			t.Fatalf("err = %v, want unknown enum error", err)
		}
	})
}

// TestGenerateStopsWhenLoadOptionsUnresolved pins the wiring: a policy that does
// not resolve must stop generation before any template renders, rather than emit
// a package with options missing.
func TestGenerateStopsWhenLoadOptionsUnresolved(t *testing.T) {
	templateData := NewTemplateData("8.18.0", []introspection.Operation{{Name: "resize"}}, nil, nil, false)
	loader := &stubTemplateLoader{listFiles: []string{"image.go.tmpl"}}

	err := Generate(loader, templateData, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "load options") {
		t.Fatalf("Generate err = %v, want load options error", err)
	}
	if len(loader.generated) != 0 {
		t.Errorf("rendered %d templates with an unresolved policy", len(loader.generated))
	}
}

// TestGenerateLoadOptionsSnapshot pins the emitted LoadOptions copy, which is v1
// API: the fields, their order, and the option string it serialises to.
func TestGenerateLoadOptionsSnapshot(t *testing.T) {
	loadOptions := buildTestLoadOptions(t)

	wantFields := "" +
		"\t// N Number of pages to load, -1 for all\n" +
		"\tN int\n" +
		"\t// Page First page to load\n" +
		"\tPage int\n" +
		"\t// Dpi Resolution in DPI\n" +
		"\tDpi int\n" +
		"\t// Autorotate Rotate image using exif orientation\n" +
		"\tAutorotate bool\n" +
		"\t// FailOn is the error level to fail on (none/truncated/error/warning).\n" +
		"\t// When both FailOn and FailOnError are set, libvips ignores the\n" +
		"\t// deprecated \"fail\" boolean and uses fail_on.\n" +
		"\tFailOn FailOn\n" +
		"\t// FailOnError Fail on first error. Deprecated: maps to the libvips\n" +
		"\t// \"fail\" boolean, which libvips remaps to fail_on=warning. Prefer FailOn.\n" +
		"\tFailOnError bool\n" +
		"\t// Shrink Shrink factor for jpeg load\n" +
		"\tShrink int\n" +
		"\t// Thumbnail Load the thumbnail instead of main image (for HEIF)\n" +
		"\tThumbnail bool\n" +
		"\t// Unlimited Allow without size restrictions\n" +
		"\tUnlimited bool\n" +
		"\t// Memory Force open via memory\n" +
		"\tMemory bool\n" +
		"\t// Access Required access pattern for this file\n" +
		"\tAccess Access\n"

	if got := generateLoadOptionsFields(loadOptions); got != wantFields {
		t.Errorf("unexpected LoadOptions fields\n got: %q\nwant: %q", got, wantFields)
	}

	wantDefault := "\treturn &LoadOptions{\n\t\tFailOnError: true,\n\t}\n"
	if got := generateLoadOptionsDefault(loadOptions); got != wantDefault {
		t.Errorf("unexpected DefaultLoadOptions body\n got: %q\nwant: %q", got, wantDefault)
	}

	// The option string order is v1's, which is not the field order.
	wantOptionString := "" +
		"\tif i.N != 0 {\n\t\tvalues = append(values, \"n=\"+strconv.Itoa(i.N))\n\t}\n" +
		"\tif i.Page != 0 {\n\t\tvalues = append(values, \"page=\"+strconv.Itoa(i.Page))\n\t}\n" +
		"\tif i.Dpi != 0 {\n\t\tvalues = append(values, \"dpi=\"+strconv.Itoa(i.Dpi))\n\t}\n" +
		"\tif i.FailOn != FailOnNone {\n" +
		"\t\tswitch i.FailOn {\n" +
		"\t\tcase FailOnTruncated:\n\t\t\tvalues = append(values, \"fail_on=truncated\")\n" +
		"\t\tcase FailOnError:\n\t\t\tvalues = append(values, \"fail_on=error\")\n" +
		"\t\tcase FailOnWarning:\n\t\t\tvalues = append(values, \"fail_on=warning\")\n" +
		"\t\t}\n\t}\n" +
		"\tif i.FailOnError {\n\t\tvalues = append(values, \"fail=\"+boolToStr(i.FailOnError))\n\t}\n" +
		"\tif i.Shrink != 0 {\n\t\tvalues = append(values, \"shrink=\"+strconv.Itoa(i.Shrink))\n\t}\n" +
		"\tif i.Autorotate {\n\t\tvalues = append(values, \"autorotate=\"+boolToStr(i.Autorotate))\n\t}\n" +
		"\tif i.Unlimited {\n\t\tvalues = append(values, \"unlimited=\"+boolToStr(i.Unlimited))\n\t}\n" +
		"\tif i.Thumbnail {\n\t\tvalues = append(values, \"thumbnail=\"+boolToStr(i.Thumbnail))\n\t}\n" +
		"\tif i.Memory {\n\t\tvalues = append(values, \"memory=\"+boolToStr(i.Memory))\n\t}\n" +
		"\tif i.Access != AccessRandom {\n" +
		"\t\tswitch i.Access {\n" +
		"\t\tcase AccessSequential:\n\t\t\tvalues = append(values, \"access=sequential\")\n" +
		"\t\tcase AccessSequentialUnbuffered:\n\t\t\tvalues = append(values, \"access=sequential-unbuffered\")\n" +
		"\t\t}\n\t}\n"

	if got := generateLoadOptionsOptionString(loadOptions); got != wantOptionString {
		t.Errorf("unexpected OptionString body\n got: %q\nwant: %q", got, wantOptionString)
	}
}
