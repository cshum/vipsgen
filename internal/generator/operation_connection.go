package generator

import (
	"fmt"
	"strings"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// A format has two connection variants that need no file at all: a loader that
// reads from a Source, and a saver that writes to a Target. The fixture for the
// first is the same image written to bytes by the format's own saver; the second
// is checked by looking at what arrived on the other end, which is the mirror
// image of the file round trip.

// connectionKind classifies a loader that reads a Source, or a saver that writes a
// Target. A source loader is only coverable when some saver for the same format
// returns bytes this test can hand it.
func (f coverageFormats) connectionKind(op introspection.Operation) (kind string, producer coverageByteSaver, ok bool) {
	args := callArguments(op)
	if len(args) != 1 {
		return "", coverageByteSaver{}, false
	}
	token := formatToken(op.Name)
	if args[0].IsSource && !op.HasThisImageInput && op.HasOneImageOutput {
		producer, exists := f.bytes[token]
		if !exists {
			return "", coverageByteSaver{}, false
		}
		return "sourceLoader", producer, true
	}
	if args[0].IsTarget && op.HasThisImageInput && !op.HasImageOutput && !op.HasBufferOutput {
		return "targetSaver", coverageByteSaver{}, true
	}
	return "", coverageByteSaver{}, false
}

// bufferKind classifies a loader that takes the image's bytes directly: the same
// fixture as a source loader, handed over as a slice instead of a stream.
func (f coverageFormats) bufferKind(op introspection.Operation) (producer coverageByteSaver, ok bool) {
	if op.HasThisImageInput || !op.HasOneImageOutput || !strings.HasSuffix(op.Name, "_buffer") {
		return coverageByteSaver{}, false
	}
	for _, arg := range op.RequiredInputs {
		// The bytes are the argument the test supplies, and `len` is the synthetic
		// pair introspection adds for the C wrapper - the Go call takes the slice
		// alone. Anything else required is a value the test cannot invent.
		if arg.IsBuffer || arg.IsImage || arg.IsSource || arg.IsTarget || arg.IsOutput {
			continue
		}
		if arg.Name == "len" && arg.GoType == "int" {
			continue
		}
		return coverageByteSaver{}, false
	}
	producer, exists := f.bytes[formatToken(op.Name)]
	if !exists {
		return coverageByteSaver{}, false
	}
	return producer, true
}

// bufferLoaderCoverageBody writes the fixture and reads it back from a slice.
func bufferLoaderCoverageBody(op introspection.Operation, producer coverageByteSaver, co coverageOptions) string {
	skipReason := co.skipReason(op.Name)
	hasOptions := len(op.OptionalInputs) > 0 || len(getSupportedOptionalOutputs(op)) > 0
	callArgs := "data"
	if hasOptions {
		callArgs += ", " + co.Expr
	}

	var body strings.Builder
	body.WriteString("\t\timg := coverageImage(t)\n")
	if strings.HasSuffix(producer.GoName, "Target") {
		body.WriteString("\t\ttarget, buf := coverageTarget(t)\n")
		body.WriteString(fmt.Sprintf("\t\tif err := img.%s(target, %s); err != nil {\n", producer.GoName, producer.Options))
		body.WriteString("\t\t\tt.Skipf(\"fixture unavailable: the saver could not write it: %v\", err)\n")
		body.WriteString("\t\t}\n")
		body.WriteString("\t\tdata := buf.Bytes()\n")
	} else {
		body.WriteString(fmt.Sprintf("\t\tdata, err := img.%s(%s)\n", producer.GoName, producer.Options))
		body.WriteString("\t\tif err != nil {\n")
		body.WriteString("\t\t\tt.Skipf(\"fixture unavailable: the saver could not write it: %v\", err)\n")
		body.WriteString("\t\t}\n")
	}
	body.WriteString(co.Decl)
	body.WriteString(fmt.Sprintf("\t\tloaded, err := New%s(%s)\n", op.GoName, callArgs))
	body.WriteString("\t\tif err != nil {\n")
	body.WriteString(errorLine(skipReason))
	body.WriteString("\t\t}\n\t\tdefer loaded.Close()\n")
	if co.Expr == "nil" {
		body.WriteString("\t\tif loaded.Width() != img.Width() || loaded.Height() != img.Height() {\n")
		body.WriteString(fmt.Sprintf("\t\t\tt.Fatalf(%q, loaded.Width(), loaded.Height(), img.Width(), img.Height())\n", op.Name+": loaded %dx%d from a %dx%d buffer"))
		body.WriteString("\t\t}\n")
	} else {
		body.WriteString("\t\tif loaded.Width() <= 0 || loaded.Height() <= 0 {\n")
		body.WriteString(fmt.Sprintf("\t\t\tt.Fatalf(%q, loaded.Width(), loaded.Height())\n", op.Name+": loaded an empty image %dx%d"))
		body.WriteString("\t\t}\n")
	}
	return body.String()
}

// sourceLoaderCoverageBody writes the image to bytes with the paired saver, reads
// those bytes back through a Source, and compares the geometry.
func sourceLoaderCoverageBody(op introspection.Operation, producer coverageByteSaver, co coverageOptions) string {
	skipReason := co.skipReason(op.Name)
	hasOptions := len(op.OptionalInputs) > 0 || len(getSupportedOptionalOutputs(op)) > 0
	callArgs := "source"
	if hasOptions {
		callArgs += ", " + co.Expr
	}

	var body strings.Builder
	body.WriteString("\t\timg := coverageImage(t)\n")
	if strings.HasSuffix(producer.GoName, "Target") {
		// Only a target saver exists for this format - vips, csv, matrix, ppm - so
		// the fixture is caught on its way to a connection.
		body.WriteString("\t\ttarget, buf := coverageTarget(t)\n")
		body.WriteString(fmt.Sprintf("\t\tif err := img.%s(target, %s); err != nil {\n", producer.GoName, producer.Options))
		body.WriteString("\t\t\tt.Skipf(\"fixture unavailable: the saver could not write it: %v\", err)\n")
		body.WriteString("\t\t}\n")
		body.WriteString("\t\tdata := buf.Bytes()\n")
	} else {
		body.WriteString(fmt.Sprintf("\t\tdata, err := img.%s(%s)\n", producer.GoName, producer.Options))
		body.WriteString("\t\tif err != nil {\n")
		body.WriteString("\t\t\tt.Skipf(\"fixture unavailable: the saver could not write it: %v\", err)\n")
		body.WriteString("\t\t}\n")
	}
	body.WriteString("\t\tsource := coverageSource(t, data)\n")
	body.WriteString(co.Decl)
	body.WriteString(fmt.Sprintf("\t\tloaded, err := New%s(%s)\n", op.GoName, callArgs))
	body.WriteString("\t\tif err != nil {\n")
	body.WriteString(errorLine(skipReason))
	body.WriteString("\t\t}\n\t\tdefer loaded.Close()\n")
	if co.Expr == "nil" {
		body.WriteString("\t\tif loaded.Width() != img.Width() || loaded.Height() != img.Height() {\n")
		body.WriteString(fmt.Sprintf("\t\t\tt.Fatalf(%q, loaded.Width(), loaded.Height(), img.Width(), img.Height())\n", op.Name+": loaded %dx%d from a %dx%d stream"))
		body.WriteString("\t\t}\n")
	} else {
		body.WriteString("\t\tif loaded.Width() <= 0 || loaded.Height() <= 0 {\n")
		body.WriteString(fmt.Sprintf("\t\t\tt.Fatalf(%q, loaded.Width(), loaded.Height())\n", op.Name+": loaded an empty image %dx%d"))
		body.WriteString("\t\t}\n")
	}
	return body.String()
}

// targetSaverCoverageBody writes to a Target backed by memory and asserts the
// bytes arrived.
func targetSaverCoverageBody(op introspection.Operation, co coverageOptions) string {
	skipReason := co.skipReason(op.Name)
	hasOptions := len(op.OptionalInputs) > 0 || len(getSupportedOptionalOutputs(op)) > 0
	callArgs := "target"
	if hasOptions {
		callArgs += ", " + co.Expr
	}

	var body strings.Builder
	body.WriteString("\t\timg := coverageImage(t)\n")
	body.WriteString("\t\ttarget, buf := coverageTarget(t)\n")
	body.WriteString(co.Decl)
	body.WriteString(fmt.Sprintf("\t\terr := img.%s(%s)\n", op.GoName, callArgs))
	body.WriteString("\t\tif err != nil {\n")
	if skipReason != "" {
		body.WriteString(errorLine(skipReason))
		body.WriteString("\t\t}\n")
		body.WriteString(targetAssertion(op))
		return body.String()
	}
	body.WriteString("\t\t\tfirstErr := err\n")
	body.WriteString("\t\t\timg = coverageMonoImage(t)\n")
	body.WriteString("\t\t\ttarget, buf = coverageTarget(t)\n")
	body.WriteString(fmt.Sprintf("\t\t\terr = img.%s(%s)\n", op.GoName, callArgs))
	body.WriteString("\t\t\tif err != nil {\n")
	body.WriteString(fmt.Sprintf("\t\t\t\tt.Fatalf(%q, firstErr, err)\n", op.Name+": wrapper failed for a three-band and a one-band image: %v / %v"))
	body.WriteString("\t\t\t}\n\t\t}\n")
	body.WriteString(targetAssertion(op))
	return body.String()
}

// targetAssertion checks that a saver wrote something to the connection.
func targetAssertion(op introspection.Operation) string {
	return "\t\tif buf.Len() == 0 {\n" +
		fmt.Sprintf("\t\t\tt.Fatalf(%q)\n", op.Name+": wrote nothing to the target") +
		"\t\t}\n"
}

// coverageByteSaver is a saver a subtest calls to obtain bytes, with the argument that
// call takes: an ImageMagick saver carries no format of its own, so the test names one.
type coverageByteSaver struct {
	GoName  string
	Options string // "nil" when the call needs nothing else
}

// coverageByteSavers indexes the savers that return bytes: a buffer saver, or the
// target saver for a format whose only byte-producing operation writes to a
// connection. The returned GoName is what the test calls to make a fixture.
func coverageByteSavers(operations []introspection.Operation) map[string]coverageByteSaver {
	savers := map[string]coverageByteSaver{}
	for _, op := range operations {
		if !op.HasThisImageInput {
			continue
		}
		token := formatToken(op.Name)
		switch {
		case strings.HasSuffix(op.Name, "_buffer") && op.HasBufferOutput:
			if _, taken := savers[token]; !taken {
				savers[token] = coverageByteSaver{GoName: op.GoName, Options: byteSaverOptions(op)}
			}
		case strings.HasSuffix(op.Name, "_target") && !op.HasImageOutput:
			if _, taken := savers[token]; !taken {
				savers[token] = coverageByteSaver{GoName: op.GoName, Options: "nil"}
			}
		}
	}
	return savers
}

// byteSaverOptions names the format a saver cannot infer from a filename: a saver writing
// to bytes has no extension to read, and the operation says so by taking a format option.
func byteSaverOptions(op introspection.Operation) string {
	for _, arg := range op.OptionalInputs {
		if arg.Name == "format" && arg.GoType == "string" {
			return fmt.Sprintf("&%sOptions{Format: \"png\"}", op.GoName)
		}
	}
	return "nil"
}

// thumbnailCoverageBody covers the three thumbnail entry points. Unlike every other
// generated call they take a width beside the image, and they are what most callers
// start from: a file, a Source or a buffer, shrunk to fit.
func thumbnailCoverageBody(op introspection.Operation, co coverageOptions) (string, bool) {
	var variant string
	switch {
	case op.Name == "thumbnail":
		variant = "file"
	case op.Name == "thumbnail_source":
		variant = "source"
	case op.Name == "thumbnail_buffer":
		variant = "buffer"
	default:
		return "", false
	}

	hasOptions := len(op.OptionalInputs) > 0 || len(getSupportedOptionalOutputs(op)) > 0
	opts := "nil"
	if hasOptions {
		opts = co.Expr
	}
	skipReason := co.skipReason(op.Name)

	var body strings.Builder
	body.WriteString("\t\timg := coverageImage(t)\n")
	switch variant {
	case "file":
		body.WriteString("\t\tpath := coveragePath(t, \"png\")\n")
		body.WriteString("\t\tif err := img.Pngsave(path, nil); err != nil {\n")
		body.WriteString("\t\t\tt.Skipf(\"fixture unavailable: the saver could not write it: %v\", err)\n")
		body.WriteString("\t\t}\n")
		body.WriteString(co.Decl)
		body.WriteString(fmt.Sprintf("\t\tthumb, err := New%s(path, 32, %s)\n", op.GoName, opts))
	case "source", "buffer":
		body.WriteString("\t\tdata, err := img.PngsaveBuffer(nil)\n")
		body.WriteString("\t\tif err != nil {\n")
		body.WriteString("\t\t\tt.Skipf(\"fixture unavailable: the saver could not write it: %v\", err)\n")
		body.WriteString("\t\t}\n")
		if variant == "source" {
			body.WriteString("\t\tsource := coverageSource(t, data)\n")
		}
		body.WriteString(co.Decl)
		arg := "data"
		if variant == "source" {
			arg = "source"
		}
		body.WriteString(fmt.Sprintf("\t\tthumb, err := New%s(%s, 32, %s)\n", op.GoName, arg, opts))
	}
	body.WriteString("\t\tif err != nil {\n")
	body.WriteString(errorLine(skipReason))
	body.WriteString("\t\t}\n\t\tdefer thumb.Close()\n")
	body.WriteString("\t\tif thumb.Width() <= 0 || thumb.Width() > 32 {\n")
	body.WriteString(fmt.Sprintf("\t\t\tt.Fatalf(%q, thumb.Width(), thumb.Height())\n", op.Name+": asked for 32 and got %dx%d"))
	body.WriteString("\t\t}\n")
	return body.String(), true
}
