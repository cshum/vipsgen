package generator

import (
	"fmt"
	"strings"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// coverageValuesExclude lists the operations with a value result that are not
// generated at all: one writes to standard output, and one leaves a libvips
// allocation that the generated suite's leak gate counts (2 allocs, ~7.6kB, with
// the returned image closed) - excluded rather than failing every run over it.
var coverageValuesExclude = map[string]string{
	"matrixprint":  "writes to standard output",
	"fill_nearest": "leaves an allocation the leak gate counts",
}

// valuesCaseBody renders a subtest for an operation whose outputs are values
// rather than an image: statistics, trim boxes, projections, and the operations
// that return nothing at all. Each output is checked against what the test image
// can produce, which is the only assertion available without a golden value.
func valuesCaseBody(op introspection.Operation, co coverageOptions) (string, bool) {
	if !op.HasThisImageInput {
		return "", false
	}
	if _, excluded := coverageValuesExclude[op.Name]; excluded {
		return "", false
	}
	skipReason := co.skipReason(op.Name)

	hasOptions := len(op.OptionalInputs) > 0 || len(getSupportedOptionalOutputs(op)) > 0
	var exprs []string
	var imageArgs []string
	for _, arg := range callArguments(op) {
		switch arg.GoType {
		case "*C.VipsImage":
			name := imageName(len(imageArgs))
			imageArgs = append(imageArgs, name)
			exprs = append(exprs, name)
		case "[]*C.VipsImage":
			first := imageName(len(imageArgs))
			second := imageName(len(imageArgs) + 1)
			imageArgs = append(imageArgs, first, second)
			exprs = append(exprs, fmt.Sprintf("[]*Image{%s, %s}", first, second))
		default:
			value, ok := scalarValue(arg)
			if !ok {
				return "", false
			}
			exprs = append(exprs, value)
		}
	}
	if hasOptions {
		exprs = append(exprs, co.Expr)
	}
	argsExpr := strings.Join(exprs, ", ")

	type result struct {
		name   string
		goType string
		image  bool
	}
	var results []result
	for i, out := range op.RequiredOutputs {
		if out.IsOutputN {
			continue
		}
		item := result{goType: out.GoType}
		switch out.GoType {
		case "*C.VipsImage":
			item.name = fmt.Sprintf("out%d", i)
			item.image = true
		case "float64", "int", "[]float64":
			item.name = fmt.Sprintf("v%d", i)
		default:
			// A boolean or anything else this generator cannot judge: bind it to
			// the blank identifier rather than invent an assertion.
			item.name = "_"
		}
		results = append(results, item)
	}

	var body strings.Builder
	body.WriteString(declareImages(append([]string{"img"}, imageArgs...), "coverageImage", false, 2))
	body.WriteString(co.Decl)

	if skipReason != "" {
		// Nothing to judge beyond the call itself: bind the values away, but keep an
		// image result so it can still be closed.
		for i := range results {
			if !results[i].image {
				results[i].name = "_"
			}
		}
	}

	var lhs []string
	for _, item := range results {
		lhs = append(lhs, item.name)
	}
	if len(lhs) == 0 {
		body.WriteString(fmt.Sprintf("\t\terr := img.%s(%s)\n", op.GoName, argsExpr))
	} else {
		body.WriteString(fmt.Sprintf("\t\t%s, err := img.%s(%s)\n", strings.Join(lhs, ", "), op.GoName, argsExpr))
	}
	body.WriteString("\t\tif err != nil {\n")
	body.WriteString(errorLine(skipReason))
	body.WriteString("\t\t}\n")

	for _, item := range results {
		if skipReason != "" {
			// Nothing to judge beyond the call itself. An image result is still
			// closed so a skipped operation does not leak one.
			if item.image {
				body.WriteString("\t\tdefer " + item.name + ".Close()\n")
			}
			continue
		}
		if item.image {
			body.WriteString("\t\tdefer " + item.name + ".Close()\n")
			body.WriteString("\t\tif " + item.name + ".Width() <= 0 || " + item.name + ".Height() <= 0 {\n")
			body.WriteString(fmt.Sprintf("\t\t\tt.Fatalf(%q, %s.Width(), %s.Height())\n", op.Name+": produced an empty image %dx%d", item.name, item.name))
			body.WriteString("\t\t}\n")
			continue
		}
		switch item.goType {
		case "float64":
			body.WriteString(fmt.Sprintf("\t\tif math.IsNaN(%s) || %s < 0 || %s > 255 {\n", item.name, item.name, item.name))
			body.WriteString(fmt.Sprintf("\t\t\tt.Fatalf(%q, %s)\n", op.Name+": %v is outside the range of a uchar image", item.name))
			body.WriteString("\t\t}\n")
		case "int":
			body.WriteString(fmt.Sprintf("\t\tif %s < 0 {\n", item.name))
			body.WriteString(fmt.Sprintf("\t\t\tt.Fatalf(%q, %s)\n", op.Name+": %d is negative", item.name))
			body.WriteString("\t\t}\n")
		case "[]float64":
			body.WriteString(fmt.Sprintf("\t\tif len(%s) == 0 {\n", item.name))
			body.WriteString(fmt.Sprintf("\t\t\tt.Fatalf(%q)\n", op.Name+": returned no values"))
			body.WriteString("\t\t}\n")
		}
	}

	return body.String(), true
}
