package introspection

import (
	"github.com/cshum/vipsgen"
	"strings"
)

// UpdateImageInputOutputFlags examines operation arguments and sets proper flags
func (v *Introspection) UpdateImageInputOutputFlags(op *vipsgen.Operation) {
	op.HasImageInput = false
	op.HasOneImageOutput = false
	op.HasArrayImageInput = false
	var imageOutputCount int

	// Check each argument to see if this operation takes/returns an image
	for _, arg := range op.Arguments {
		// Check for any input parameter with VipsImage* type
		if (arg.Type == "VipsImage" || arg.CType == "VipsImage*") && !arg.IsOutput {
			op.HasImageInput = true
		}

		// Check for "out" parameter with VipsImage* type
		if arg.Type == "VipsImage" && arg.CType == "VipsImage**" && arg.IsOutput {
			imageOutputCount++
		}

		// Check for array image inputs
		if strings.HasPrefix(arg.GoType, "[]*C.VipsImage") ||
			(strings.Contains(arg.CType, "VipsImage**") && !arg.IsOutput) {
			op.HasArrayImageInput = true
		}

		if arg.CType == "void**" && arg.Name == "buf" {
			op.HasBufferOutput = true
		}
		if arg.CType == "void*" && arg.Name == "buf" {
			op.HasBufferInput = true
		}
	}
	if imageOutputCount == 1 && !op.HasArrayImageInput {
		op.HasOneImageOutput = true
	}
}

// FixOperationTypes examines operations and adjusts their types based on patterns
func (v *Introspection) FixOperationTypes(op *vipsgen.Operation) {
	// Pattern detection: Vector return operations
	// If function has output param named "vector" paired with output param "n", it's returning an array
	hasVectorParam := false
	hasNParam := false

	for _, arg := range op.Outputs {
		if arg.Name == "vector" {
			hasVectorParam = true
		}
		if arg.Name == "n" {
			hasNParam = true
		}
	}

	// If we have both vector and n params, this is a vector return function
	if hasVectorParam && hasNParam {
		for i, arg := range op.Outputs {
			if arg.Name == "vector" {
				// Update the type to be a slice
				op.Outputs[i].GoType = "[]float64"

				// Also update in Arguments if present
				for j, mainArg := range op.Arguments {
					if mainArg.Name == "vector" {
						op.Arguments[j].GoType = "[]float64"
					}
				}
			}
		}
	}

	// Fix double* args
	for i, arg := range op.Arguments {
		if arg.CType == "double*" && !arg.IsOutput {
			op.Arguments[i].GoType = "[]float64"
		}
	}

	// Fix the mode parameter
	if strings.Contains(op.Name, "composite") {
		// Fix the mode parameter - should be an array of BlendMode
		for i, arg := range op.Arguments {
			if arg.Name == "mode" && arg.CType == "int*" && arg.GoType == "int" {
				// Update to array of BlendMode
				op.Arguments[i].GoType = "[]BlendMode"
				op.Arguments[i].IsEnum = true
				op.Arguments[i].EnumType = "BlendMode"

				// Also update in RequiredInputs if present
				for j, inputArg := range op.RequiredInputs {
					if inputArg.Name == "mode" {
						op.RequiredInputs[j].GoType = "[]BlendMode"
						op.RequiredInputs[j].IsEnum = true
						op.RequiredInputs[j].EnumType = "BlendMode"
					}
				}
			}
		}
	}

	// Fix the case operation - cases parameter should be an array of images
	if op.Name == "case" {
		for i, arg := range op.Arguments {
			if arg.Name == "cases" && arg.CType == "VipsImage**" {
				op.Arguments[i].GoType = "[]*C.VipsImage"
				for j, inputArg := range op.RequiredInputs {
					if inputArg.Name == "cases" {
						op.RequiredInputs[j].GoType = "[]*C.VipsImage"
					}
				}
			}
		}
	}
}
