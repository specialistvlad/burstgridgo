// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Vladyslav Kazantsev
//
// This file defines the structure for a Runner's output values and the logic
// for parsing their definitions from HCL.
//
// Why define outputs?
//
// Formally defining a runner's outputs serves as a public contract for what a
// successfully completed step will produce. This schema is essential for the
// dependency graph's integrity and enables static analysis of expressions that
// reference this runner's outputs.
//
// When another step references an output (e.g., in an expression like
// `${step.A.output.id}`), the system can use this schema to:
//
//  1. Validate References: Ensure that the requested output field (`id`)
//     actually exists on the runner's definition.
//
//  2. Perform Type Checking: Check that the type of the output (e.g., `string`)
//     is compatible with how it is being used in the expression.
//
// This prevents an entire class of runtime errors that would otherwise only be
// discovered when the workflow is already executing.
package model

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/specialistvlad/burstgridgo/internal/bggohcl"
	"github.com/zclconf/go-cty/cty"
)

// RunnerOutputDefinition defines a single output value from a runner.
type RunnerOutputDefinition struct {
	Name        string
	Type        cty.Type
	Description string
}

// outputBodySchema is the HCL schema for the body of an `output` block.
var outputBodySchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "type", Required: true},
		{Name: "description"},
	},
}

// parseRunnerOutputs finds and decodes all 'output' blocks from a runner's HCL body.
func parseRunnerOutputs(blocks hcl.Blocks) (map[string]RunnerOutputDefinition, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	outputs := make(map[string]RunnerOutputDefinition)

	outputBlocks := blocks.OfType("output")
	for _, block := range outputBlocks {
		// The schema guarantees us one label for the output name.
		outputName := block.Labels[0]

		if _, exists := outputs[outputName]; exists {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate output definition",
				Detail:   fmt.Sprintf("An output named '%s' has already been defined.", outputName),
				Subject:  &block.DefRange,
			})
			continue
		}

		bodyContent, contentDiags := block.Body.Content(outputBodySchema)
		diags = append(diags, contentDiags...)
		if contentDiags.HasErrors() {
			continue
		}

		// The schema enforces that 'type' is required, so we can safely access it.
		typeAttr := bodyContent.Attributes["type"]
		ctyType, typeDiags := bggohcl.HCLTypeToCtyType(typeAttr.Expr)
		diags = append(diags, typeDiags...)
		if typeDiags.HasErrors() {
			continue
		}

		// Decode the optional 'description' attribute.
		var description string
		if descAttr, exists := bodyContent.Attributes["description"]; exists {
			evalDiags := gohcl.DecodeExpression(descAttr.Expr, nil, &description)
			diags = append(diags, evalDiags...)
		}

		outputs[outputName] = RunnerOutputDefinition{
			Name:        outputName,
			Type:        ctyType,
			Description: description,
		}
	}

	return outputs, diags
}
