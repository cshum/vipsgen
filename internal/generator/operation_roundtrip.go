package generator

import (
	"fmt"
	"strings"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// A foreign format is one libvips can both write and read: the saver provides the
// fixture its loader needs, so a generated subtest can round-trip the format
// instead of asserting on a file no one can make.

// coverageFormatExt is the file extension a loader expects for a format token.
// The token is the operation name with the direction and connection suffix
// removed, so jpegsave and jpegload share `jpeg`. A format without an entry here
// is not round-tripped: `dz` writes a directory, and the loaders whose fixture
// readers are external - analyze, dcraw, mat, openexr, openslide, pdf, svg - have
// no saver to pair with.
var coverageFormatExt = map[string]string{
	"csv":    "csv",
	"fits":   "fits",
	"gif":    "gif",
	"heif":   "heic",
	"jp2k":   "jp2",
	"jpeg":   "jpg",
	"jxl":    "jxl",
	"magick": "miff",
	"matrix": "mat",
	"png":    "png",
	"ppm":    "ppm",
	"rad":    "hdr",
	"raw":    "raw",
	"tiff":   "tif",
	"uhdr":   "jpg",
	"vips":   "v",
	"webp":   "webp",
}

// coverageFormats indexes the savers by format token, so a loader can find the
// operation that writes its fixture.
type coverageFormats struct {
	saver map[string]string // token -> saver GoName that writes a file
	bytes map[string]string // token -> saver GoName that returns bytes
}

func newCoverageFormats(operations []introspection.Operation) coverageFormats {
	formats := coverageFormats{saver: map[string]string{}, bytes: coverageByteSavers(operations)}
	for _, op := range operations {
		if !strings.HasSuffix(op.Name, "save") || !op.HasThisImageInput {
			continue
		}
		if op.HasImageOutput || op.HasBufferOutput {
			continue // a buffer saver has no filename to read back
		}
		token := formatToken(op.Name)
		if _, ok := coverageFormatExt[token]; !ok {
			continue
		}
		formats.saver[token] = op.GoName
	}
	return formats
}

// foreignKind classifies an operation that takes a filename: a saver writes the
// fixture, a loader reads one - and a loader is only coverable when a saver for
// the same format exists to write what it opens.
func (f coverageFormats) foreignKind(op introspection.Operation) (kind, ext, saver string, ok bool) {
	// Exactly one argument: the filename. A loader that also needs the image's
	// dimensions - rawload - cannot be handed a file this generator knows how to
	// write, and a saver that needs more than a path is not round-tripped either.
	args := callArguments(op)
	if len(args) != 1 || args[0].GoType != "string" || args[0].Name != "filename" {
		return "", "", "", false
	}
	for _, arg := range op.RequiredInputs {
		if arg.IsBuffer || arg.IsSource || arg.IsTarget {
			return "", "", "", false
		}
	}

	token := formatToken(op.Name)
	ext, known := coverageFormatExt[token]
	if !known {
		return "", "", "", false
	}

	switch {
	case strings.HasSuffix(op.Name, "save") && op.HasThisImageInput:
		return "saver", ext, "", true
	case strings.HasSuffix(op.Name, "load") && !op.HasThisImageInput:
		paired, exists := f.saver[token]
		if !exists {
			return "", "", "", false
		}
		return "loader", ext, paired, true
	}
	return "", "", "", false
}

// formatToken is the operation name without its direction or connection suffix:
// jpegsave_buffer -> jpeg, dzsave -> dz.
func formatToken(name string) string {
	name = strings.TrimSuffix(name, "_buffer")
	name = strings.TrimSuffix(name, "_source")
	name = strings.TrimSuffix(name, "_target")
	name = strings.TrimSuffix(name, "save")
	name = strings.TrimSuffix(name, "load")
	return name
}

// saverCoverageBody writes the fixture and asserts something landed on disk.
func saverCoverageBody(op introspection.Operation, ext string, co coverageOptions) string {
	skipReason := co.skipReason(op.Name)
	hasOptions := len(op.OptionalInputs) > 0 || len(getSupportedOptionalOutputs(op)) > 0
	callArgs := "path"
	if hasOptions {
		callArgs += ", " + co.Expr
	}

	var body strings.Builder
	body.WriteString(fmt.Sprintf("\t\timg := coverageImage(t)\n\t\tpath := coveragePath(t, %q)\n", ext))
	body.WriteString(co.Decl)
	body.WriteString(fmt.Sprintf("\t\terr := img.%s(%s)\n", op.GoName, callArgs))
	body.WriteString("\t\tif err != nil {\n")
	if skipReason != "" {
		body.WriteString(errorLine(skipReason))
		body.WriteString("\t\t}\n")
		body.WriteString(fileAssertion(op))
		return body.String()
	}
	body.WriteString("\t\t\tfirstErr := err\n")
	body.WriteString("\t\t\timg = coverageMonoImage(t)\n")
	body.WriteString(fmt.Sprintf("\t\t\terr = img.%s(%s)\n", op.GoName, callArgs))
	body.WriteString("\t\t\tif err != nil {\n")
	body.WriteString(fmt.Sprintf("\t\t\t\tt.Fatalf(%q, firstErr, err)\n", op.Name+": wrapper failed for a three-band and a one-band image: %v / %v"))
	body.WriteString("\t\t\t}\n\t\t}\n")
	body.WriteString(fileAssertion(op))
	return body.String()
}

// loaderCoverageBody writes the fixture with the paired saver, then loads it and
// compares the geometry: a loader that hands back the wrong image is the failure
// a signature check cannot see.
func loaderCoverageBody(op introspection.Operation, ext, saver string, co coverageOptions) string {
	skipReason := co.skipReason(op.Name)
	hasOptions := len(op.OptionalInputs) > 0 || len(getSupportedOptionalOutputs(op)) > 0
	callArgs := "path"
	if hasOptions {
		callArgs += ", " + co.Expr
	}

	var body strings.Builder
	body.WriteString(fmt.Sprintf("\t\timg := coverageImage(t)\n\t\tpath := coveragePath(t, %q)\n", ext))
	body.WriteString(fmt.Sprintf("\t\tif err := img.%s(path, nil); err != nil {\n", saver))
	body.WriteString(fmt.Sprintf("\t\t\tt.Skipf(%q, err)\n", "fixture unavailable: the saver could not write it: %v"))
	body.WriteString("\t\t}\n")
	body.WriteString(co.Decl)
	body.WriteString(fmt.Sprintf("\t\tloaded, err := New%s(%s)\n", op.GoName, callArgs))
	body.WriteString("\t\tif err != nil {\n")
	body.WriteString(errorLine(skipReason))
	body.WriteString("\t\t}\n\t\tdefer loaded.Close()\n")
	if co.Expr == "nil" {
		// Without options the round trip must preserve geometry.
		body.WriteString("\t\tif loaded.Width() != img.Width() || loaded.Height() != img.Height() {\n")
		body.WriteString(fmt.Sprintf("\t\t\tt.Fatalf(%q, loaded.Width(), loaded.Height(), img.Width(), img.Height())\n", op.Name+": loaded %dx%d from a %dx%d file"))
		body.WriteString("\t\t}\n")
	} else {
		// With options set, the geometry may legitimately change - csvload's lines
		// reads a subset, jpegload's shrink scales - so only the result is checked.
		body.WriteString("\t\tif loaded.Width() <= 0 || loaded.Height() <= 0 {\n")
		body.WriteString(fmt.Sprintf("\t\t\tt.Fatalf(%q, loaded.Width(), loaded.Height())\n", op.Name+": loaded an empty image %dx%d"))
		body.WriteString("\t\t}\n")
	}
	return body.String()
}

// fileAssertion checks that a saver wrote something.
func fileAssertion(op introspection.Operation) string {
	return "\t\tif info, statErr := os.Stat(path); statErr != nil || info.Size() == 0 {\n" +
		fmt.Sprintf("\t\t\tt.Fatalf(%q, path)\n", op.Name+": wrote no file at %s") +
		"\t\t}\n"
}
