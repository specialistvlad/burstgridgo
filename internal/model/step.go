// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Vladyslav Kazantsev
//
// This file defines the Step structure, which is the atomic unit of work within
// a Grid. It represents a single, configured invocation of a Runner.
//
// Why the Step struct?
//
// While a Runner defines the "what" (the task's inputs, outputs, and logic),
// a Step defines the "how," "when," and "with what." It is the node in the
// execution graph. This struct is intentionally comprehensive, designed to capture
// every possible configuration attribute a user can provide in an HCL `step` block.
//
// Why store raw hcl.Expression fields?
//
// You'll notice that most fields are of type `hcl.Expression` rather than a
// primitive Go type. This is a deliberate and critical design choice. It allows
// us to defer the evaluation of configuration values until the graph is being
// built or executed. This is the mechanism that enables dynamic workflows, where
// a step's configuration (like its `count` or an argument) can be derived from
// the output of another step. The model captures the user's intent as an
// expression, and a later stage is responsible for resolving that expression
// into a concrete value.
package model

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/specialistvlad/burstgridgo/internal/bggoexpr"
	"github.com/specialistvlad/burstgridgo/internal/bggohcl"
)

// Step is the format-agnostic representation of a `step` block.
type Step struct {
	RunnerType    string
	Name          string
	FSInformation *FSInfo

	// Core Attributes
	Enabled     *hcl.Expression
	Description *hcl.Expression
	Tags        hcl.Expression
	Scope       hcl.Expression
	Uses        hcl.Expression
	Arguments   map[string]hcl.Expression

	// Looping
	Count   hcl.Expression
	ForEach hcl.Expression

	// Execution Control
	Priority    hcl.Expression
	DelayBefore hcl.Expression
	DelayAfter  hcl.Expression
	Timeouts    *Timeouts

	// Concurrency & Rate Limiting
	Concurrency *Concurrency
	RateLimit   *RateLimit

	// Error Handling
	Retry             *Retry
	OnError           *OnError
	ContinueOnFailure hcl.Expression

	// State & Idempotency
	Cache          *Cache
	Dedupe         *Dedupe
	IdempotencyKey hcl.Expression

	// Environment & Observability
	Sensitive *hcl.Expression
	Env       hcl.Expression
	Tracing   *Tracing
	Metrics   *Metrics

	// Placement
	Placement *Placement

	// Expression container
	Expressions *bggoexpr.Container
}

// NewStep creates a new, empty Step struct.
func NewStep() *Step {
	return &Step{
		Expressions: bggoexpr.NewContainer(),
	}
}

// hclStep represents a single 'step' block for initial decoding from HCL.
type hclStep struct {
	Type string   `hcl:"type,label"`
	Name string   `hcl:"name,label"`
	Body hcl.Body `hcl:",remain"`
}

// NewStepFromHCL creates a new Step from a parsed HCL step block.
func NewStepFromHCL(parsedStep *hclStep, filePath string) (*Step, hcl.Diagnostics) {
	step := NewStep()
	step.RunnerType = parsedStep.Type
	step.Name = parsedStep.Name

	step.FSInformation = NewFSInfo(filePath)

	var allDiags hcl.Diagnostics

	bodyContent, contentDiags := parsedStep.Body.Content(stepBodySchema)
	allDiags = append(allDiags, contentDiags...)
	if contentDiags.HasErrors() {
		return nil, allDiags
	}

	// --- Parse all simple attributes ---
	for _, parser := range attributeParsers {
		if attr, exists := bodyContent.Attributes[parser.Name]; exists {
			parser.Setter(step, attr.Expr)
			step.Expressions.Add(attr.Expr)
		}
	}

	// --- Handle special attributes (looping, dependencies) ---
	var specialDiags hcl.Diagnostics
	step.Count, specialDiags = parseCount(bodyContent.Attributes)
	allDiags = append(allDiags, specialDiags...)
	step.Expressions.Add(step.Count)

	step.ForEach, specialDiags = parseForEach(bodyContent.Attributes)
	allDiags = append(allDiags, specialDiags...)
	step.Expressions.Add(step.ForEach)

	depsExpr, depDiags := parseDependsOn(bodyContent.Attributes)
	allDiags = append(allDiags, depDiags...)
	step.Expressions.Add(depsExpr)

	// --- Handle nested blocks ---
	if argBlock, diags := bggohcl.FindUniqueBlock(bodyContent.Blocks, "arguments"); diags.HasErrors() {
		allDiags = append(allDiags, diags...)
	} else if argBlock != nil {
		var argDiags hcl.Diagnostics
		step.Arguments, argDiags = parseArguments(argBlock)
		allDiags = append(allDiags, argDiags...)
		for _, argExpr := range step.Arguments {
			step.Expressions.Add(argExpr)
		}
	}

	for _, parser := range blockParsers {
		result, blockExprs, blockDiags := parser.Parse(bodyContent.Blocks)
		allDiags = append(allDiags, blockDiags...)
		step.Expressions.Add(blockExprs...)
		if result != nil {
			parser.Setter(step, result)
		}
	}

	// --- Final validation ---
	validationDiags := validateStepLoopingAttributes(step.Count, step.ForEach, parsedStep.Body)
	allDiags = append(allDiags, validationDiags...)

	if allDiags.HasErrors() {
		return nil, allDiags
	}

	return step, allDiags
}
