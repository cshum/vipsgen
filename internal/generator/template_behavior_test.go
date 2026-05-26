package generator

import (
	"testing"

	"github.com/cshum/vipsgen/internal/introspection"
)

func TestGenerateImageMethodBodySingleImageOutputSnapshot(t *testing.T) {
	op := introspection.Operation{
		GoName:            "Blur",
		HasOneImageOutput: true,
		Arguments: []introspection.Argument{
			{Name: "in", GoName: "in", GoType: "*C.VipsImage", IsInput: true, IsImage: true},
			{Name: "mask", GoName: "mask", GoType: "*C.VipsImage", IsInput: true, IsImage: true},
			{Name: "scale", GoName: "scale", GoType: "float64", IsInput: true},
		},
		OptionalInputs: []introspection.Argument{
			{Name: "extend", GoName: "Extend", GoType: "string"},
		},
		OptionalOutputs: []introspection.Argument{
			{Name: "score", GoName: "Score", GoType: "float64"},
		},
	}

	got := generateImageMethodBody(op)
	want := "if options != nil {\n\t\tout, err := vipsgenBlurWithOptions(r.image, mask.image, scale, options.Extend, &options.Score)\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\tr.setImage(out)\n\t\treturn nil\n\t}\n\tout, err := vipsgenBlur(r.image, mask.image, scale)\n\tif err != nil {\n\t\treturn err\n\t}\n\tr.setImage(out)\n\treturn nil"

	if got != want {
		t.Fatalf("unexpected image method body\n got: %q\nwant: %q", got, want)
	}
}

func TestGenerateImageMethodBodyImageOutputsSnapshot(t *testing.T) {
	op := introspection.Operation{
		GoName:         "Blend",
		HasImageOutput: true,
		Arguments: []introspection.Argument{
			{Name: "in", GoName: "in", GoType: "*C.VipsImage", IsInput: true, IsImage: true},
			{Name: "alpha", GoName: "alpha", GoType: "float64", IsInput: true},
			{Name: "out", GoName: "out", GoType: "*C.VipsImage", IsOutput: true},
			{Name: "flags", GoName: "flags", GoType: "int", IsOutput: true},
		},
		RequiredOutputs: []introspection.Argument{
			{Name: "out", GoName: "out", GoType: "*C.VipsImage", IsOutput: true},
			{Name: "flags", GoName: "flags", GoType: "int", IsOutput: true},
		},
		OptionalInputs: []introspection.Argument{
			{Name: "mode", GoName: "Mode", GoType: "string"},
		},
	}

	got := generateImageMethodBody(op)
	want := "if options != nil {\n\t\tout, flags, err := vipsgenBlendWithOptions(r.image, alpha, options.Mode)\n\t\tif err != nil {\n\t\t\treturn nil, 0, err\n\t\t}\n\t\toutImage := newImageRef(out, r.format, nil)\n\t\treturn outImage, flags, nil\n\t}\n\tout, flags, err := vipsgenBlend(r.image, alpha)\n\tif err != nil {\n\t\treturn nil, 0, err\n\t}\n\toutImage := newImageRef(out, r.format, nil)\n\treturn outImage, flags, nil"

	if got != want {
		t.Fatalf("unexpected image outputs method body\n got: %q\nwant: %q", got, want)
	}
}

func TestGenerateImageMethodBodySingleFloatReturnSnapshot(t *testing.T) {
	op := introspection.Operation{
		GoName: "Avg",
		Arguments: []introspection.Argument{
			{Name: "in", GoName: "in", GoType: "*C.VipsImage", IsInput: true, IsImage: true},
		},
		RequiredOutputs: []introspection.Argument{
			{Name: "out", GoName: "out", GoType: "float64", IsOutput: true},
		},
		OptionalInputs: []introspection.Argument{
			{Name: "precision", GoName: "Precision", GoType: "string"},
		},
		OptionalOutputs: []introspection.Argument{
			{Name: "deviation", GoName: "Deviation", GoType: "float64"},
		},
	}

	got := generateImageMethodBody(op)
	want := "if options != nil {\n\t\tout, err := vipsgenAvgWithOptions(r.image, options.Precision, &options.Deviation)\n\t\tif err != nil {\n\t\t\treturn 0, err\n\t\t}\n\t\treturn out, nil\n\t}\n\tout, err := vipsgenAvg(r.image)\n\tif err != nil {\n\t\treturn 0, err\n\t}\n\treturn out, nil"

	if got != want {
		t.Fatalf("unexpected single float method body\n got: %q\nwant: %q", got, want)
	}
}

func TestGenerateImageMethodBodyVoidReturnSafePointerSnapshot(t *testing.T) {
	op := introspection.Operation{
		GoName: "Relational",
		Arguments: []introspection.Argument{
			{Name: "in", GoName: "in", GoType: "*C.VipsImage", IsInput: true, IsImage: true},
		},
		OptionalInputs: []introspection.Argument{
			{Name: "mask", GoName: "Mask", GoType: "*C.VipsImage"},
			{Name: "mode", GoName: "Mode", GoType: "string"},
		},
		OptionalOutputs: []introspection.Argument{
			{Name: "score", GoName: "Score", GoType: "float64"},
		},
	}

	got := generateImageMethodBody(op)
	want := "if options != nil {\n\t\terr := vipsgenRelationalWithOptions(r.image, getImagePointer(options.Mask), options.Mode, &options.Score)\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\treturn nil\n\t}\n\terr := vipsgenRelational(r.image)\n\tif err != nil {\n\t\treturn err\n\t}\n\treturn nil"

	if got != want {
		t.Fatalf("unexpected void return method body\n got: %q\nwant: %q", got, want)
	}
}

func TestGenerateCreatorMethodBodyBufferInputSnapshot(t *testing.T) {
	op := introspection.Operation{
		Name:            "jpegload_buffer",
		GoName:          "JpegloadBuffer",
		HasBufferInput:  true,
		ImageTypeString: "ImageTypeJpeg",
		RequiredInputs: []introspection.Argument{
			{Name: "buf", GoName: "buf", GoType: "[]byte", CType: "void*", IsInput: true, IsBuffer: true},
			{Name: "len", GoName: "len", GoType: "int", CType: "size_t", IsInput: true},
		},
		OptionalInputs: []introspection.Argument{
			{Name: "shrink", GoName: "Shrink", GoType: "int"},
		},
		OptionalOutputs: []introspection.Argument{
			{Name: "pages", GoName: "Pages", GoType: "int"},
		},
	}

	got := generateCreatorMethodBody(op)
	want := "Startup(nil)\n\tif len(buf) == 0 {\n\t\treturn nil, fmt.Errorf(\"jpegload_buffer: buffer is empty\")\n\t}\n\tif options != nil {\n\t\tvipsImage, err := vipsgenJpegloadBufferWithOptions(buf, options.Shrink, &options.Pages)\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t\treturn newImageRef(vipsImage, ImageTypeJpeg, buf), nil\n\t}\n\tvipsImage, err := vipsgenJpegloadBuffer(buf)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\treturn newImageRef(vipsImage, ImageTypeJpeg, buf), nil"

	if got != want {
		t.Fatalf("unexpected creator method body\n got: %q\nwant: %q", got, want)
	}
}

func TestGenerateGoFunctionBodySingleScalarOutputSnapshot(t *testing.T) {
	op := introspection.Operation{
		Name:   "avg",
		GoName: "Avg",
		Arguments: []introspection.Argument{
			{Name: "in", GoName: "in", GoType: "*C.VipsImage", CType: "VipsImage*", IsInput: true, IsImage: true},
			{Name: "out", GoName: "out", GoType: "float64", CType: "double*", IsOutput: true},
		},
		RequiredOutputs: []introspection.Argument{
			{Name: "out", GoName: "out", GoType: "float64", CType: "double*", IsOutput: true},
		},
	}

	got := generateGoFunctionBody(op, false)
	want := "// vipsgenAvg \nfunc vipsgenAvg(in *C.VipsImage) (float64, error) {\n\tvar out float64\n\tcout := new(C.double)\n\tif err := C.vipsgen_avg(in, cout); err != 0 {\n\t\treturn 0, handleVipsError()\n\t}\n\tout = float64(*cout)\n\treturn out, nil\n}"

	if got != want {
		t.Fatalf("unexpected go wrapper body\n got: %q\nwant: %q", got, want)
	}
}

func TestGenerateCFunctionDeclarationBufferLoadWithOptionsSnapshot(t *testing.T) {
	op := introspection.Operation{
		Name: "jpegload_buffer",
		Arguments: []introspection.Argument{
			{Name: "buf", CType: "void*", GoType: "[]byte", IsInput: true, IsBuffer: true},
			{Name: "len", CType: "size_t", GoType: "int", IsInput: true},
			{Name: "out", CType: "VipsImage**", GoType: "*C.VipsImage", IsOutput: true},
		},
		OptionalInputs: []introspection.Argument{
			{Name: "shrink", CType: "int", GoType: "int"},
		},
		OptionalOutputs: []introspection.Argument{
			{Name: "pages", CType: "int*", GoType: "int"},
		},
	}

	got := generateCFunctionDeclaration(op)
	want := "int vipsgen_jpegload_buffer(void* buf, size_t len, VipsImage** out);\nint vipsgen_jpegload_buffer_with_options(void* buf, size_t len, VipsImage** out, int shrink, int* pages);"

	if got != want {
		t.Fatalf("unexpected C function declaration\n got: %q\nwant: %q", got, want)
	}
}

func TestGenerateCFunctionImplementationBufferLoadWithOptionsSnapshot(t *testing.T) {
	op := introspection.Operation{
		Name: "jpegload_buffer",
		Arguments: []introspection.Argument{
			{Name: "buf", CType: "void*", GoType: "[]byte", IsInput: true, IsBuffer: true},
			{Name: "len", CType: "size_t", GoType: "int", IsInput: true},
			{Name: "out", CType: "VipsImage**", GoType: "*C.VipsImage", IsOutput: true},
		},
		RequiredInputs: []introspection.Argument{
			{Name: "buf", CType: "void*", GoType: "[]byte", IsInput: true, IsBuffer: true},
			{Name: "len", CType: "size_t", GoType: "int", IsInput: true},
		},
		OptionalInputs: []introspection.Argument{
			{Name: "shrink", CType: "int", GoType: "int"},
		},
		OptionalOutputs: []introspection.Argument{
			{Name: "pages", CType: "int*", GoType: "int"},
		},
	}

	got := generateCFunctionImplementation(op)
	want := "int vipsgen_jpegload_buffer(void* buf, size_t len, VipsImage** out) {\n    return vips_jpegload_buffer(buf, len, out, NULL);\n}\n\nint vipsgen_jpegload_buffer_with_options(void* buf, size_t len, VipsImage** out, int shrink, int* pages) {\n    VipsOperation *operation = vips_operation_new(\"jpegload_buffer\");\n    if (!operation) return 1;\n    VipsBlob *blob = vips_blob_new(NULL, buf, len);\n    if (!blob) { g_object_unref(operation); return 1; }\n    if (\n        vips_object_set(VIPS_OBJECT(operation), \"buffer\", blob, NULL) ||\n        vipsgen_set_int(operation, \"shrink\", shrink)\n    ) {\n        vips_area_unref((VipsArea *)blob);\n        g_object_unref(operation);\n        return 1;\n    }\n    vips_area_unref((VipsArea *)blob);\n    int result = vipsgen_operation_execute(operation, \"out\", out, \"pages\", pages, NULL);\n    return result;\n}"

	if got != want {
		t.Fatalf("unexpected C function implementation\n got: %q\nwant: %q", got, want)
	}
}

func TestGenerateCFunctionImplementationWebpSaveAllowsZeroEffortSnapshot(t *testing.T) {
	op := introspection.Operation{
		Name: "webpsave",
		Arguments: []introspection.Argument{
			{Name: "in", CType: "VipsImage*", GoType: "*C.VipsImage", IsInput: true, IsImage: true},
			{Name: "filename", CType: "char*", GoType: "string", IsInput: true},
		},
		RequiredInputs: []introspection.Argument{
			{Name: "in", CType: "VipsImage*", GoType: "*C.VipsImage", IsInput: true, IsImage: true},
			{Name: "filename", CType: "char*", GoType: "string", IsInput: true},
		},
		OptionalInputs: []introspection.Argument{
			{Name: "effort", CType: "int", GoType: "int"},
		},
	}

	got := generateCFunctionImplementation(op)
	want := "int vipsgen_webpsave(VipsImage* in, char* filename) {\n    return vips_webpsave(in, filename, NULL);\n}\n\nint vipsgen_webpsave_with_options(VipsImage* in, char* filename, int effort) {\n    VipsOperation *operation = vips_operation_new(\"webpsave\");\n    if (!operation) return 1;\n    if (\n        vips_object_set(VIPS_OBJECT(operation), \"in\", in, NULL) ||\n        vips_object_set(VIPS_OBJECT(operation), \"filename\", filename, NULL) ||\n        vipsgen_set_int_allow_zero(operation, \"effort\", effort)\n    ) {\n        g_object_unref(operation);\n        return 1;\n    }\n    int result = vipsgen_operation_execute(operation, NULL);\n    return result;\n}"

	if got != want {
		t.Fatalf("unexpected webpsave C function implementation\n got: %q\nwant: %q", got, want)
	}
}

func TestGenerateCFunctionImplementationPngSaveKeepsZeroAsUnsetSnapshot(t *testing.T) {
	op := introspection.Operation{
		Name: "pngsave",
		Arguments: []introspection.Argument{
			{Name: "in", CType: "VipsImage*", GoType: "*C.VipsImage", IsInput: true, IsImage: true},
			{Name: "filename", CType: "char*", GoType: "string", IsInput: true},
		},
		RequiredInputs: []introspection.Argument{
			{Name: "in", CType: "VipsImage*", GoType: "*C.VipsImage", IsInput: true, IsImage: true},
			{Name: "filename", CType: "char*", GoType: "string", IsInput: true},
		},
		OptionalInputs: []introspection.Argument{
			{Name: "effort", CType: "int", GoType: "int"},
		},
	}

	got := generateCFunctionImplementation(op)
	want := "int vipsgen_pngsave(VipsImage* in, char* filename) {\n    return vips_pngsave(in, filename, NULL);\n}\n\nint vipsgen_pngsave_with_options(VipsImage* in, char* filename, int effort) {\n    VipsOperation *operation = vips_operation_new(\"pngsave\");\n    if (!operation) return 1;\n    if (\n        vips_object_set(VIPS_OBJECT(operation), \"in\", in, NULL) ||\n        vips_object_set(VIPS_OBJECT(operation), \"filename\", filename, NULL) ||\n        vipsgen_set_int(operation, \"effort\", effort)\n    ) {\n        g_object_unref(operation);\n        return 1;\n    }\n    int result = vipsgen_operation_execute(operation, NULL);\n    return result;\n}"

	if got != want {
		t.Fatalf("unexpected pngsave C function implementation\n got: %q\nwant: %q", got, want)
	}
}
