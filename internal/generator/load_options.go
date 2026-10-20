package generator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// LoadOptionKind is how a LoadOptions field is spelled into the option string.
type LoadOptionKind int

const (
	// LoadOptionInt is a Go int sent as a decimal string.
	LoadOptionInt LoadOptionKind = iota
	// LoadOptionBool is a Go bool sent as true/false.
	LoadOptionBool
	// LoadOptionEnum is a libvips enum, sent by the value's libvips nick.
	LoadOptionEnum
)

// LoadOptionEnumCase is one arm of the generated switch for an enum option.
type LoadOptionEnumCase struct {
	GoName string // Go constant, e.g. FailOnTruncated
	Nick   string // libvips nick, e.g. "truncated"
}

// LoadOptionField is one generated field of LoadOptions.
type LoadOptionField struct {
	GoName   string
	GoType   string
	Doc      []string
	Kind     LoadOptionKind
	Option   string // name in the option string
	Default  string // literal for DefaultLoadOptions, empty for the Go zero value
	Emit     int    // position in the option string, zero for none
	ZeroName string // enum constant carrying value zero, empty when the enum has none
	Cases    []LoadOptionEnumCase
}

// LoadOptionsTemplate is the resolved input of the LoadOptions template block.
type LoadOptionsTemplate struct {
	Fields []LoadOptionField
}

// loadOptionSpec is one row of the LoadOptions policy. The table is data, not
// code: a field is here because a caller of the generic loader can meaningfully
// set it, which is why the set is not the intersection of the loader options
// (that would be access, fail_on, memory and revalidate only) nor their union
// (password, page_box and stylesheet belong to one format each).
//
// ArgName is the option's name in libvips introspection, which is where the Go
// type and an enum's values and nicks come from. An empty ArgName marks an
// option introspection does not report, so it cannot be resolved and cannot be
// checked at build time. Doc is hand-written on purpose: a union option has no
// single libvips blurb, and the per-loader blurbs contradict each other
// ("Allow SVG of any size" against "Remove all denial of service limits").
type loadOptionSpec struct {
	GoName  string
	ArgName string
	Option  string   // name in the option string when it differs from ArgName
	GoType  string   // overrides the introspected type when v1 chose another
	Doc     []string // field comment lines, without the leading tabs and slashes
	Default string   // literal for DefaultLoadOptions, empty for the Go zero value
	Emit    int      // position in the option string, zero for none
}

// loadOptionSpecs is the LoadOptions policy, ordered by struct field. Emit keeps
// the option string in the order v1 wrote it, which is not the field order.
var loadOptionSpecs = []loadOptionSpec{
	{
		GoName:  "N",
		ArgName: "n",
		Doc:     []string{"N Number of pages to load, -1 for all"},
		Emit:    1,
	},
	{
		GoName:  "Page",
		ArgName: "page",
		Doc:     []string{"Page First page to load"},
		Emit:    2,
	},
	{
		GoName:  "Dpi",
		ArgName: "dpi",
		// libvips reports gdouble; v1 exposed int and sends it as an integer,
		// which libvips accepts. Changing it now would break the API.
		GoType: "int",
		Doc:    []string{"Dpi Resolution in DPI"},
		Emit:   3,
	},
	{
		GoName:  "Autorotate",
		ArgName: "autorotate",
		Doc:     []string{"Autorotate Rotate image using exif orientation"},
		Emit:    7,
	},
	{
		GoName:  "FailOn",
		ArgName: "fail_on",
		Doc: []string{
			"FailOn is the error level to fail on (none/truncated/error/warning).",
			"When both FailOn and FailOnError are set, libvips ignores the",
			`deprecated "fail" boolean and uses fail_on.`,
		},
		Emit: 4,
	},
	{
		// The "fail" boolean is deprecated and introspection drops it, so this
		// row is the only record that the option exists at all.
		GoName:  "FailOnError",
		Option:  "fail",
		GoType:  "bool",
		Default: "true",
		Doc: []string{
			"FailOnError Fail on first error. Deprecated: maps to the libvips",
			`"fail" boolean, which libvips remaps to fail_on=warning. Prefer FailOn.`,
		},
		Emit: 5,
	},
	{
		GoName:  "Shrink",
		ArgName: "shrink",
		Doc:     []string{"Shrink Shrink factor for jpeg load"},
		Emit:    6,
	},
	{
		GoName:  "Thumbnail",
		ArgName: "thumbnail",
		Doc:     []string{"Thumbnail Load the thumbnail instead of main image (for HEIF)"},
		Emit:    9,
	},
	{
		GoName:  "Unlimited",
		ArgName: "unlimited",
		Doc:     []string{"Unlimited Allow without size restrictions"},
		Emit:    8,
	},
	{
		GoName:  "Memory",
		ArgName: "memory",
		Doc:     []string{"Memory Force open via memory"},
		Emit:    10,
	},
	{
		GoName:  "Access",
		ArgName: "access",
		Doc:     []string{"Access Required access pattern for this file"},
		Emit:    11,
	},
}

// isLoaderOperation reports whether an operation loads an image through the
// foreign loader machinery, which is where LoadOptions applies.
func isLoaderOperation(name string) bool {
	return strings.HasSuffix(name, "load") ||
		strings.HasSuffix(name, "load_buffer") ||
		strings.HasSuffix(name, "load_source")
}

// loadArg is what the loader operations agree on for one option name.
type loadArg struct {
	GoType string
}

// BuildLoadOptions resolves the LoadOptions policy against the introspected
// loaders.
func BuildLoadOptions(operations []introspection.Operation, enumTypes []introspection.EnumTypeInfo) (*LoadOptionsTemplate, error) {
	return resolveLoadOptions(loadOptionSpecs, operations, enumTypes)
}

// resolveLoadOptions resolves spec against the introspected loaders. It returns
// an error when a spec names an option libvips no longer reports, when loaders
// disagree about an option's type, or when an enum value has no libvips nick to
// spell it with: a loader option that silently stops being generated would
// change what callers' options do, so the mismatch has to stop the build.
func resolveLoadOptions(specs []loadOptionSpec, operations []introspection.Operation, enumTypes []introspection.EnumTypeInfo) (*LoadOptionsTemplate, error) {
	args := map[string]*loadArg{}
	for _, op := range operations {
		if !isLoaderOperation(op.Name) {
			continue
		}
		for _, arg := range op.OptionalInputs {
			known, ok := args[arg.Name]
			if !ok {
				args[arg.Name] = &loadArg{GoType: arg.GoType}
				continue
			}
			if known.GoType != arg.GoType {
				return nil, fmt.Errorf("load option %q is %s and %s: loaders disagree on its type",
					arg.Name, known.GoType, arg.GoType)
			}
		}
	}

	enums := make(map[string]introspection.EnumTypeInfo, len(enumTypes))
	for _, enum := range enumTypes {
		enums[enum.GoName] = enum
	}

	fields := make([]LoadOptionField, 0, len(specs))
	for _, spec := range specs {
		goType := spec.GoType
		if spec.ArgName != "" {
			arg, ok := args[spec.ArgName]
			if !ok {
				return nil, fmt.Errorf("load option %q is not on any loader operation", spec.ArgName)
			}
			if goType == "" {
				goType = arg.GoType
			}
		}

		field := LoadOptionField{
			GoName:  spec.GoName,
			GoType:  goType,
			Doc:     spec.Doc,
			Option:  spec.Option,
			Default: spec.Default,
			Emit:    spec.Emit,
		}
		if field.Option == "" {
			field.Option = spec.ArgName
		}
		if field.Option == "" {
			return nil, fmt.Errorf("load option field %q has no option name", spec.GoName)
		}

		switch {
		case goType == "bool":
			field.Kind = LoadOptionBool
		case goType == "int":
			field.Kind = LoadOptionInt
		default:
			enum, ok := enums[goType]
			if !ok {
				return nil, fmt.Errorf("load option %q is %q, which is neither int, bool nor a libvips enum", field.Option, goType)
			}
			field.Kind = LoadOptionEnum
			for _, value := range enum.Values {
				if value.Value == 0 {
					field.ZeroName = value.GoName
					continue
				}
				if value.Description == "" {
					return nil, fmt.Errorf("enum %s value %s has no libvips nick", goType, value.GoName)
				}
				field.Cases = append(field.Cases, LoadOptionEnumCase{
					GoName: value.GoName,
					Nick:   value.Description,
				})
			}
		}

		fields = append(fields, field)
	}

	return &LoadOptionsTemplate{Fields: fields}, nil
}

// generateLoadOptionsFields emits the body of the LoadOptions struct.
func generateLoadOptionsFields(loadOptions *LoadOptionsTemplate) string {
	var result strings.Builder
	for _, field := range loadOptions.Fields {
		for _, line := range field.Doc {
			result.WriteString("\t// " + line + "\n")
		}
		result.WriteString("\t" + field.GoName + " " + field.GoType + "\n")
	}
	return result.String()
}

// generateLoadOptionsDefault emits the body of DefaultLoadOptions.
func generateLoadOptionsDefault(loadOptions *LoadOptionsTemplate) string {
	var result strings.Builder
	result.WriteString("\treturn &LoadOptions{\n")
	for _, field := range loadOptions.Fields {
		if field.Default == "" {
			continue
		}
		result.WriteString("\t\t" + field.GoName + ": " + field.Default + ",\n")
	}
	result.WriteString("\t}\n")
	return result.String()
}

// generateLoadOptionsOptionString emits the body of LoadOptions.OptionString.
// A field is sent only when it differs from the Go zero value, not when it
// differs from libvips' default: the string reaches whichever loader the format
// picks, and a loader that does not know an option rejects the whole load.
func generateLoadOptionsOptionString(loadOptions *LoadOptionsTemplate) string {
	emitting := make([]LoadOptionField, 0, len(loadOptions.Fields))
	for _, field := range loadOptions.Fields {
		if field.Emit > 0 {
			emitting = append(emitting, field)
		}
	}
	sort.SliceStable(emitting, func(i, j int) bool {
		return emitting[i].Emit < emitting[j].Emit
	})

	var result strings.Builder
	for _, field := range emitting {
		switch field.Kind {
		case LoadOptionBool:
			result.WriteString(fmt.Sprintf("\tif i.%s {\n", field.GoName))
			result.WriteString(fmt.Sprintf("\t\tvalues = append(values, %q+boolToStr(i.%s))\n", field.Option+"=", field.GoName))
			result.WriteString("\t}\n")
		case LoadOptionEnum:
			guard := "0"
			if field.ZeroName != "" {
				guard = field.ZeroName
			}
			result.WriteString(fmt.Sprintf("\tif i.%s != %s {\n", field.GoName, guard))
			result.WriteString(fmt.Sprintf("\t\tswitch i.%s {\n", field.GoName))
			for _, enumCase := range field.Cases {
				result.WriteString(fmt.Sprintf("\t\tcase %s:\n", enumCase.GoName))
				result.WriteString(fmt.Sprintf("\t\t\tvalues = append(values, %q)\n", field.Option+"="+enumCase.Nick))
			}
			result.WriteString("\t\t}\n")
			result.WriteString("\t}\n")
		default:
			result.WriteString(fmt.Sprintf("\tif i.%s != 0 {\n", field.GoName))
			result.WriteString(fmt.Sprintf("\t\tvalues = append(values, %q+strconv.Itoa(i.%s))\n", field.Option+"=", field.GoName))
			result.WriteString("\t}\n")
		}
	}
	return result.String()
}
