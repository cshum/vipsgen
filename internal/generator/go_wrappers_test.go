package generator

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cshum/vipsgen/internal/introspection"
)

func TestGetOutputScalarCType(t *testing.T) {
	tests := []struct {
		name string
		arg  introspection.Argument
		want string
	}{
		{name: "gboolean c type", arg: introspection.Argument{CType: "gboolean*", GoType: "bool"}, want: "C.gboolean"},
		{name: "guint c type", arg: introspection.Argument{CType: "guint*", GoType: "int"}, want: "C.uint"},
		{name: "gint c type", arg: introspection.Argument{CType: "gint*", GoType: "int"}, want: "C.gint"},
		{name: "fallback go type", arg: introspection.Argument{CType: "VipsInterestingThing*", GoType: "float64"}, want: "C.double"},
		{name: "unsupported type", arg: introspection.Argument{CType: "char*", GoType: "string"}, want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := getOutputScalarCType(test.arg)
			if got != test.want {
				t.Fatalf("unexpected scalar c type: got %q want %q", got, test.want)
			}
		})
	}
}

func TestGenerateOutputErrorReturn(t *testing.T) {
	outputs := []introspection.Argument{
		{Name: "vector", GoType: "[]float64"},
		{Name: "n", GoType: "int", IsOutputN: true},
		{Name: "score", GoType: "float64"},
		{Name: "ok", GoType: "bool"},
	}

	got := generateOutputErrorReturn(outputs, "err")
	want := "return nil, 0, false, err"
	if got != want {
		t.Fatalf("unexpected output error return: got %q want %q", got, want)
	}
}

func TestGenerateTypedArrayConversionDeclaration(t *testing.T) {
	requiredArg := introspection.Argument{GoName: "values", GoType: "[]float64", IsRequired: true}
	required := generateTypedArrayConversionDeclaration(requiredArg, "return err", "convertToDoubleArray", "freeDoubleArray")
	if !strings.Contains(required, "if values == nil {") {
		t.Fatalf("required array declaration missing nil guard: %q", required)
	}
	if !strings.Contains(required, "values = []float64{}") {
		t.Fatalf("required array declaration missing safe default: %q", required)
	}
	if !strings.Contains(required, "cvalues, _, err := convertToDoubleArray(values)") {
		t.Fatalf("required array declaration missing conversion call: %q", required)
	}

	optionalArg := introspection.Argument{GoName: "images", GoType: "[]*Image", IsRequired: false}
	optional := generateTypedArrayConversionDeclaration(optionalArg, "return nil, err", "convertToImageArray", "freeImageArray")
	if strings.Contains(optional, "if images == nil {") {
		t.Fatalf("optional array declaration unexpectedly has nil guard: %q", optional)
	}
	if !strings.Contains(optional, "cimages, cimagesLength, err := convertToImageArray(images)") {
		t.Fatalf("optional array declaration missing conversion call: %q", optional)
	}
	if !strings.Contains(optional, "defer freeImageArray(cimages)") {
		t.Fatalf("optional array declaration missing cleanup: %q", optional)
	}
}

func TestGenerateArrayInputCallArgs(t *testing.T) {
	tests := []struct {
		name        string
		arg         introspection.Argument
		withOptions bool
		want        []string
	}{
		{name: "image array without options", arg: introspection.Argument{GoName: "images", GoType: "[]*C.VipsImage"}, withOptions: false, want: []string{"(**C.VipsImage)(cimages)", "cimagesLength"}},
		{name: "image array with options", arg: introspection.Argument{GoName: "images", GoType: "[]*C.VipsImage"}, withOptions: true, want: []string{"cimages", "cimagesLength"}},
		{name: "required numeric array", arg: introspection.Argument{GoName: "values", GoType: "[]float64", IsRequired: true}, withOptions: false, want: []string{"cvalues"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := generateArrayInputCallArgs(test.arg, test.withOptions)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("unexpected array input call args: got %#v want %#v", got, test.want)
			}
		})
	}
}

func TestGenerateFunctionCallArgs(t *testing.T) {
	op := introspection.Operation{
		Arguments: []introspection.Argument{
			{Name: "in", GoName: "in", GoType: "*C.VipsImage", IsInput: true},
			{Name: "name", GoName: "name", GoType: "string", IsInput: true},
			{Name: "enabled", GoName: "enabled", GoType: "bool", IsInput: true},
			{Name: "images", GoName: "images", GoType: "[]*C.VipsImage", IsInput: true},
			{Name: "mode", GoName: "mode", GoType: "Mode", Type: "VipsMode", IsInput: true, IsEnum: true},
			{Name: "out", GoName: "out", GoType: "*C.VipsImage", IsOutput: true},
			{Name: "score", GoName: "score", GoType: "float64", CType: "double*", IsOutput: true},
		},
		OptionalOutputs: []introspection.Argument{
			{Name: "pages", GoName: "pages", GoType: "int", CType: "int*"},
		},
	}

	got := generateFunctionCallArgs(op, true)
	want := "in, cname, C.int(boolToInt(enabled)), cimages, cimagesLength, C.VipsMode(mode), &out, cscore, cpages"
	if got != want {
		t.Fatalf("unexpected function call args: got %q want %q", got, want)
	}
}

func TestGenerateReturnValuesVectorOutput(t *testing.T) {
	op := introspection.Operation{
		RequiredOutputs: []introspection.Argument{
			{Name: "vector", GoName: "vector", GoType: "[]float64", IsOutput: true},
			{Name: "n", GoName: "count", GoType: "int", IsOutput: true},
			{Name: "flag", GoName: "flag", GoType: "bool", IsOutput: true},
		},
	}

	got := generateReturnValues(op)
	want := "result := make([]float64, count)\n\tcopy(result, (*[1024]float64)(unsafe.Pointer(out))[:count:count])\n\tgFreePointer(unsafe.Pointer(out))\n\treturn result, count, flag, nil"
	if got != want {
		t.Fatalf("unexpected return values: got %q want %q", got, want)
	}
}