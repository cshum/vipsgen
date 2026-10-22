package generator

import (
	"fmt"
	"strings"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// coverageFixtures maps a format token to a file in testdata/, for the formats libvips
// can read but not write: a generated round trip has no other way to obtain one. The
// generated test skips when the file is not there, which is the case for a tree made by
// the vipsgen command, so the fixtures stay with this repository.
var coverageFixtures = map[string]string{
	"analyze": "analyze.hdr", // the matching .img has to sit beside it
	"mat":     "sample.mat",
	"openexr": "sample.exr",
	"pdf":     "sample.pdf",
	"svg":     "sample.svg",
}

// fixtureKind classifies a loader whose fixture ships with the repository, and reports
// how it takes the file: by name, as a Source, or as a buffer.
func fixtureKind(op introspection.Operation) (variant, file string, ok bool) {
	file, listed := coverageFixtures[formatToken(op.Name)]
	if !listed {
		return "", "", false
	}
	if op.HasThisImageInput || !op.HasOneImageOutput {
		return "", "", false
	}
	// A buffer loader takes the bytes as a hidden argument: callArguments drops it, so
	// the operation's own flag is what says the fixture goes in as a slice.
	if strings.HasSuffix(op.Name, "_buffer") && op.HasBufferInput {
		return "buffer", file, true
	}
	args := callArguments(op)
	if len(args) != 1 {
		return "", "", false
	}
	switch {
	case args[0].GoType == "string" && args[0].Name == "filename":
		return "file", file, true
	case args[0].IsSource:
		return "source", file, true
	}
	return "", "", false
}

// fixtureLoaderBody opens the shipped fixture and checks that an image came back. The
// geometry is not pinned: poppler and librsvg differ between the versions the packages
// are generated against, and this is the binding being tested, not their rendering.
func fixtureLoaderBody(op introspection.Operation, variant, file string, co coverageOptions) string {
	skipReason := co.skipReason(op.Name)
	hasOptions := len(op.OptionalInputs) > 0 || len(getSupportedOptionalOutputs(op)) > 0
	opts := "nil"
	if hasOptions {
		opts = co.Expr
	}

	var body strings.Builder
	body.WriteString(fmt.Sprintf("\t\tif !HasOperation(%q) {\n", op.Name))
	body.WriteString(fmt.Sprintf("\t\t\tt.Skipf(\"this libvips has no %s\")\n", op.Name))
	body.WriteString("\t\t}\n")
	body.WriteString(fmt.Sprintf("\t\tpath := testdataPath(t, %q)\n", file))
	var callArgs string
	switch variant {
	case "file":
		callArgs = "path"
	case "source", "buffer":
		body.WriteString("\t\tdata, err := os.ReadFile(path)\n")
		body.WriteString("\t\tif err != nil {\n")
		body.WriteString("\t\t\tt.Fatalf(\"reading fixture: %v\", err)\n")
		body.WriteString("\t\t}\n")
		if variant == "source" {
			body.WriteString("\t\tsource := coverageSource(t, data)\n")
			callArgs = "source"
		} else {
			callArgs = "data"
		}
	}
	if hasOptions {
		callArgs += ", " + opts
	}
	body.WriteString(co.Decl)
	body.WriteString(fmt.Sprintf("\t\tloaded, err := New%s(%s)\n", op.GoName, callArgs))
	body.WriteString("\t\tif err != nil {\n")
	body.WriteString(errorLine(skipReason))
	body.WriteString("\t\t}\n\t\tdefer loaded.Close()\n")
	body.WriteString("\t\tif loaded.Width() <= 0 || loaded.Height() <= 0 {\n")
	body.WriteString(fmt.Sprintf("\t\t\tt.Fatalf(%q, loaded.Width(), loaded.Height())\n", op.Name+": opened an empty image %dx%d"))
	body.WriteString("\t\t}\n")
	return body.String()
}
