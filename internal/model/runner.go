// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Vladyslav Kazantsev
//
// This file defines the Runner, which is the reusable definition or "template"
// for a type of task that can be executed.
//
// Why distinguish between a Runner and a Step?
//
// This separation is core to the system's design and promotes reusability. A
// Runner is analogous to a function definition in a programming language: it
// defines a contract. Specifically, it declares what named inputs it requires
// (`Inputs`), what named outputs it will produce (`Outputs`), and what internal
// logic to execute (`Lifecycle`).
//
// A `Step`, in contrast, is an *invocation* or a *call* to that function. A user
// can define a single Runner (e.g., a "http-request" runner) and then instantiate
// it many times throughout their Grid with different `arguments` in each `step`
// block.
//
// This design allows the system to perform static analysis. When parsing a
// `step`, the graph builder can look up the corresponding `Runner` definition and
// validate that the `arguments` provided in the step match the schema defined
// by the runner's `Inputs`.
package model

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
)

// Runner is the format-agnostic representation of a runner's manifest.
type Runner struct {
	Type          string
	Description   string
	FSInformation *FSInfo
	Lifecycle     RunnerLifecycle
	Inputs        map[string]RunnerInputDefinition
	Outputs       map[string]RunnerOutputDefinition
}

// NewRunner is a factory function for creating Runner definitions.
func NewRunner(ctx context.Context, hclFile *hcl.File, filePath string) ([]*Runner, error) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Creating new runner definition", "file_path", filePath)

	runners, diags := ParseRunnerFile(ctx, hclFile, filePath)
	if diags.HasErrors() {
		return nil, diags
	}

	return runners, nil
}

// runnerRootSchema defines the top-level structure of the file, expecting one or more 'runner' blocks.
type runnerRootSchema struct {
	Runners []*hclRunner `hcl:"runner,block"`
}

// hclRunner represents a single 'runner' block in the HCL file for decoding purposes.
type hclRunner struct {
	Type string   `hcl:"type,label"`
	Body hcl.Body `hcl:",remain"`
}

// ParseRunnerFile decodes an HCL file that contains one or more 'runner' blocks.
func ParseRunnerFile(ctx context.Context, hclFile *hcl.File, filePath string) ([]*Runner, hcl.Diagnostics) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Parsing runner definitions from file", "file_path", filePath)

	var allDiags hcl.Diagnostics
	if hclFile == nil {
		allDiags = append(allDiags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "HCL file is nil",
		})
		return nil, allDiags
	}

	schema := &runnerRootSchema{}
	diags := gohcl.DecodeBody(hclFile.Body, nil, schema)
	allDiags = append(allDiags, diags...)
	if diags.HasErrors() {
		return nil, allDiags
	}

	runners := make([]*Runner, 0, len(schema.Runners))

	// Define the schema for the *body* of a 'runner' block.
	runnerBodySchema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "description"},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "lifecycle"},
			{Type: "input", LabelNames: []string{"name"}},
			{Type: "output", LabelNames: []string{"name"}},
		},
	}

	for _, parsedRunner := range schema.Runners {
		bodyContent, contentDiags := parsedRunner.Body.Content(runnerBodySchema)
		allDiags = append(allDiags, contentDiags...)
		if contentDiags.HasErrors() {
			continue // Skip this runner but continue parsing others
		}

		definition := &Runner{
			Type:    parsedRunner.Type,
			Inputs:  make(map[string]RunnerInputDefinition),
			Outputs: make(map[string]RunnerOutputDefinition),
		}

		definition.FSInformation = NewFSInfo(filePath)

		// Parse simple attributes
		if attr, exists := bodyContent.Attributes["description"]; exists {
			exprDiags := gohcl.DecodeExpression(attr.Expr, nil, &definition.Description)
			allDiags = append(allDiags, exprDiags...)
		}

		// Parse nested blocks
		var lifecycleDiags hcl.Diagnostics
		definition.Lifecycle, lifecycleDiags = parseRunnerLifecycle(bodyContent.Blocks)
		allDiags = append(allDiags, lifecycleDiags...)

		var inputDiags hcl.Diagnostics
		definition.Inputs, inputDiags = parseRunnerInputs(bodyContent.Blocks)
		allDiags = append(allDiags, inputDiags...)

		var outputDiags hcl.Diagnostics
		definition.Outputs, outputDiags = parseRunnerOutputs(bodyContent.Blocks)
		allDiags = append(allDiags, outputDiags...)

		runners = append(runners, definition)
	}

	if allDiags.HasErrors() {
		return nil, allDiags
	}

	logger.Debug("Successfully parsed runner definitions", "count", len(runners))
	return runners, nil
}
