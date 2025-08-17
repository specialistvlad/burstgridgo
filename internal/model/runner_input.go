// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Vladyslav Kazantsev
//
// This file defines the structure for a Runner's input arguments and the logic
// for parsing them from HCL.
//
// Why have a formal RunnerInputDefinition?
//
// By defining a clear, typed schema for a runner's inputs, we establish a formal
// "contract" or "API." This contract is the key to providing robust static
// validation. When a user writes a `step` block, the system can use these
// definitions to:
//
//  1. Validate Arguments: Check that all required arguments are provided and that
//     their types are correct. For example, it can ensure a value passed to an
//     input of `type = string` is actually a string.
//
//  2. Provide Default Values: Automatically apply default values for optional
//     arguments that the user has not specified.
//
//  3. Enable Rich Tooling: The schema can be used to generate documentation for
//     a runner or to provide features like auto-completion in a code editor.
//
// This approach shifts error detection from runtime (when the step executes) to
// parse-time, providing much faster and clearer feedback to the user.
package model

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/specialistvlad/burstgridgo/internal/bggohcl"
	"github.com/zclconf/go-cty/cty"
)

// RunnerInputDefinition defines a single input argument for a runner.
// This struct holds the fully parsed and type-checked definition of an input.
type RunnerInputDefinition struct {
	// Name is the name of the input, taken from the HCL block label.
	// For example, in `input "message" {}`, the Name is "message".
	Name string

	// Type is the value type that this input is expected to have.
	Type cty.Type

	// Description is an optional markdown string that describes the input's purpose.
	Description string

	// Default is an optional pointer to a cty.Value that should be used if
	// the caller does not provide one. If this field is nil, the input is required.
	Default *cty.Value
}

// inputBodySchema is the HCL schema for the body of an `input` block.
var inputBodySchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		// `type` is required, but we check for its existence manually
		// to provide a better error message.
		{Name: "type"},
		{Name: "description"},
		{Name: "default"},
	},
}

// parseRunnerInputs finds and decodes all 'input' blocks from a runner's HCL body.
func parseRunnerInputs(blocks hcl.Blocks) (map[string]RunnerInputDefinition, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	inputs := make(map[string]RunnerInputDefinition)

	inputBlocks := blocks.OfType("input")
	for _, block := range inputBlocks {
		// The schema guarantees us one label.
		inputName := block.Labels[0]

		if _, exists := inputs[inputName]; exists {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate input definition",
				Detail:   fmt.Sprintf("An input named '%s' has already been defined.", inputName),
				Subject:  &block.DefRange,
			})
			continue
		}

		bodyContent, contentDiags := block.Body.Content(inputBodySchema)
		diags = append(diags, contentDiags...)
		if contentDiags.HasErrors() {
			continue
		}

		// Manually check for the required 'type' attribute for a better error.
		typeAttr, exists := bodyContent.Attributes["type"]
		if !exists {
			missingItemRange := block.Body.MissingItemRange()
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing 'type' attribute",
				Detail:   "The 'type' attribute is required for all input blocks.",
				Subject:  &missingItemRange,
			})
			continue
		}

		ctyType, typeDiags := bggohcl.HCLTypeToCtyType(typeAttr.Expr)
		diags = append(diags, typeDiags...)
		if typeDiags.HasErrors() {
			continue
		}

		// Decode optional attributes
		var description string
		if descAttr, exists := bodyContent.Attributes["description"]; exists {
			evalDiags := gohcl.DecodeExpression(descAttr.Expr, nil, &description)
			diags = append(diags, evalDiags...)
		}

		var defaultValue *cty.Value
		if defaultAttr, exists := bodyContent.Attributes["default"]; exists {
			// A nil eval context is used because defaults must be literal values.
			val, valDiags := defaultAttr.Expr.Value(nil)
			diags = append(diags, valDiags...)
			if valDiags.HasErrors() {
				continue
			}

			// Ensure the default value's type conforms to the declared type.
			if !val.Type().Equals(ctyType) {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid default value type",
					Detail:   fmt.Sprintf("The default value for '%s' is not compatible with its type, '%s'.", inputName, ctyType.FriendlyName()),
					Subject:  defaultAttr.Expr.Range().Ptr(),
				})
				continue
			}
			defaultValue = &val
		}

		inputs[inputName] = RunnerInputDefinition{
			Name:        inputName,
			Type:        ctyType,
			Description: description,
			Default:     defaultValue,
		}
	}

	return inputs, diags
}
