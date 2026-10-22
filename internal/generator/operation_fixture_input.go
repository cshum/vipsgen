package generator

import (
	"fmt"
	"strings"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// fixtureRef points at a fixture and the loader that opens it.
type fixtureRef struct {
	Loader string // GoName of the file loader for that format
	File   string // file in testdata/
}

// coverableFixtures indexes the loaders a fixture feeds, by format token. A format that
// ships a fixture can then be handed to the operations that consume it: an image carrying
// gainmap metadata is not something a test can make up, but it can be read from the file.
func coverableFixtures(operations []introspection.Operation) map[string]fixtureRef {
	refs := map[string]fixtureRef{}
	for _, op := range operations {
		variant, file, ok := fixtureKind(op)
		if !ok || variant != "file" {
			continue
		}
		token := formatToken(op.Name)
		if _, taken := refs[token]; !taken {
			refs[token] = fixtureRef{Loader: op.GoName, File: file}
		}
	}
	return refs
}

// fixtureRefFor finds the fixture an operation consumes: a loader or saver names its format
// as a suffix, and a conversion names it as a prefix, as uhdr2scRGB does. The prefix has to
// end at a non-letter so that "mat" does not claim matrixprint.
func fixtureRefFor(refs map[string]fixtureRef, name string) (fixtureRef, bool) {
	if ref, ok := refs[formatToken(name)]; ok {
		return ref, true
	}
	lower := strings.ToLower(name)
	for token, ref := range refs {
		if !strings.HasPrefix(lower, token) || len(lower) == len(token) {
			continue
		}
		if c := lower[len(token)]; c >= 'a' && c <= 'z' {
			continue
		}
		return ref, true
	}
	return fixtureRef{}, false
}

// fixtureInputKind classifies an operation that consumes a format the fixtures cover, and
// reports what shape the call has: it happens in place, writes a file, returns bytes, or
// writes through a connection.
func fixtureInputKind(op introspection.Operation, refs map[string]fixtureRef) (ref fixtureRef, shape string, ok bool) {
	ref, listed := fixtureRefFor(refs, op.Name)
	if !listed || !op.HasThisImageInput {
		return fixtureRef{}, "", false
	}
	for _, arg := range op.RequiredInputs {
		// The destination is not part of the decision: a filename, a buffer or a
		// connection all say where the result goes, not what the call needs.
		if arg.IsImage || arg.IsTarget || arg.IsBuffer {
			continue
		}
		if arg.Name == "filename" && arg.GoType == "string" {
			continue
		}
		if arg.IsInput {
			return fixtureRef{}, "", false
		}
	}
	switch {
	case argNamed(op, "filename") != nil:
		return ref, "file", true
	case op.HasBufferOutput:
		return ref, "buffer", true
	case argIsTarget(op):
		return ref, "target", true
	}
	if op.HasImageOutput || !op.HasBufferOutput {
		return ref, "inplace", true
	}
	return fixtureRef{}, "", false
}

func argNamed(op introspection.Operation, name string) *introspection.Argument {
	for i := range op.RequiredInputs {
		if op.RequiredInputs[i].Name == name {
			return &op.RequiredInputs[i]
		}
	}
	return nil
}

func argIsTarget(op introspection.Operation) bool {
	for _, arg := range op.RequiredInputs {
		if arg.IsTarget {
			return true
		}
	}
	return false
}

// fixtureInputBody opens the fixture and then makes the call the operation describes. The
// image is what carries the metadata the operation needs - gainmap fields, for the UltraHDR
// pair - which is why it comes from a file rather than from coverageImage.
func fixtureInputBody(op introspection.Operation, ref fixtureRef, shape string, co coverageOptions) string {
	var args string
	if len(op.OptionalInputs) > 0 || len(getSupportedOptionalOutputs(op)) > 0 {
		args = co.Expr
	}

	var body strings.Builder
	body.WriteString(fmt.Sprintf("\t\timg, err := New%s(testdataPath(t, %q), nil)\n", ref.Loader, ref.File))
	body.WriteString("\t\tif err != nil {\n")
	body.WriteString("\t\t\tt.Fatalf(\"opening the fixture: %v\", err)\n")
	body.WriteString("\t\t}\n\t\tdefer img.Close()\n")
	body.WriteString(co.Decl)

	switch shape {
	case "file":
		body.WriteString("\t\tpath := coveragePath(t, \"jpg\")\n")
		body.WriteString(fmt.Sprintf("\t\tif err := img.%s(path, %s); err != nil {\n", op.GoName, args))
		body.WriteString(errorLine(co.Skip))
		body.WriteString("\t\t}\n")
		body.WriteString("\t\twritten, err := os.Stat(path)\n")
		body.WriteString("\t\tif err != nil {\n")
		body.WriteString("\t\t\tt.Fatalf(\"no file written: %v\", err)\n")
		body.WriteString("\t\t}\n")
		body.WriteString(fmt.Sprintf("\t\tif written.Size() == 0 {\n\t\t\tt.Fatalf(%q)\n\t\t}\n", op.Name+": wrote an empty file"))
	case "buffer":
		body.WriteString(fmt.Sprintf("\t\tdata, err := img.%s(%s)\n", op.GoName, args))
		body.WriteString("\t\tif err != nil {\n")
		body.WriteString(errorLine(co.Skip))
		body.WriteString("\t\t}\n")
		body.WriteString(fmt.Sprintf("\t\tif len(data) == 0 {\n\t\t\tt.Fatalf(%q)\n\t\t}\n", op.Name+": returned no bytes"))
	case "target":
		body.WriteString("\t\ttarget, buf := coverageTarget(t)\n")
		body.WriteString(fmt.Sprintf("\t\tif err := img.%s(target, %s); err != nil {\n", op.GoName, args))
		body.WriteString(errorLine(co.Skip))
		body.WriteString("\t\t}\n")
		body.WriteString(fmt.Sprintf("\t\tif buf.Len() == 0 {\n\t\t\tt.Fatalf(%q)\n\t\t}\n", op.Name+": wrote nothing to the connection"))
	default:
		body.WriteString(fmt.Sprintf("\t\tif err := img.%s(%s); err != nil {\n", op.GoName, args))
		body.WriteString(errorLine(co.Skip))
		body.WriteString("\t\t}\n")
		body.WriteString("\t\tif img.Width() <= 0 || img.Height() <= 0 {\n")
		body.WriteString(fmt.Sprintf("\t\t\tt.Fatalf(%q, img.Width(), img.Height())\n", op.Name+": produced an empty image %dx%d"))
		body.WriteString("\t\t}\n")
	}
	return body.String()
}
