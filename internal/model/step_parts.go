// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Vladyslav Kazantsev
//
// This file implements a table-driven parser for the body of a `step` block.
// It defines the schemas and parsing functions for all the nested blocks and
// attributes that a step can contain.
//
// Why use a table-driven design?
//
// A `step` block is complex, containing over a dozen optional attributes and
// nested blocks. A naive implementation would involve a very large function with
// many conditional checks, which would be difficult to read, modify, and maintain.
//
// The table-driven approach used here (`attributeParsers` and `blockParsers`)
// is a deliberate architectural choice for extensibility and clarity.
//
//   - Extensibility: To add a new attribute (e.g., `new_flag`) or a new block
//     (e.g., `new_config {}`) to the `step` definition, a developer simply adds an
//     entry to the corresponding table. The core parsing logic in `step.go` does
//     not need to change.
//
//   - Readability: The logic for parsing each specific part of a step is
//     self-contained in its parser definition. This makes it easy to understand
//     how each piece is handled without needing to understand the entire parsing
//     flow.
//
// This pattern makes the parser highly declarative and significantly reduces the
// complexity of maintaining the `step` schema over time.
package model

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/specialistvlad/burstgridgo/internal/bggoexpr"
)

// blockParser defines a generic interface for parsing a block and setting it on a Step.
type blockParser struct {
	Parse  func(blocks hcl.Blocks) (result interface{}, exprs []hcl.Expression, diags hcl.Diagnostics)
	Setter func(step *Step, result interface{})
}

// makeBlockParser is a generic helper to create a blockParser entry.
// It reduces boilerplate by generating the Parse and Setter functions based on the
// provided generic type T and a strongly-typed setter function.
// The type T is constrained to implement bggoexpr.Expressioner, which is required
// by the underlying bggoexpr.ParseBlock function.
func makeBlockParser[T bggoexpr.Expressioner](blockName string, setter func(s *Step, val T)) blockParser {
	return blockParser{
		// The Parse function calls the generic ParseBlock with the captured type and name.
		Parse: func(blocks hcl.Blocks) (interface{}, []hcl.Expression, hcl.Diagnostics) {
			return bggoexpr.ParseBlock[T](blocks, blockName)
		},
		// The Setter does a safe type assertion and then calls the strongly-typed setter.
		Setter: func(s *Step, v interface{}) {
			if r, ok := v.(T); ok {
				setter(s, r)
			}
		},
	}
}

// blockParsers is the table that drives the block parsing logic.
var blockParsers = map[string]blockParser{
	"timeouts":    makeBlockParser("timeouts", func(s *Step, v *Timeouts) { s.Timeouts = v }),
	"concurrency": makeBlockParser("concurrency", func(s *Step, v *Concurrency) { s.Concurrency = v }),
	"rate_limit":  makeBlockParser("rate_limit", func(s *Step, v *RateLimit) { s.RateLimit = v }),
	"retry":       makeBlockParser("retry", func(s *Step, v *Retry) { s.Retry = v }),
	"on_error":    makeBlockParser("on_error", func(s *Step, v *OnError) { s.OnError = v }),
	"cache":       makeBlockParser("cache", func(s *Step, v *Cache) { s.Cache = v }),
	"dedupe":      makeBlockParser("dedupe", func(s *Step, v *Dedupe) { s.Dedupe = v }),
	"tracing":     makeBlockParser("tracing", func(s *Step, v *Tracing) { s.Tracing = v }),
	"metrics":     makeBlockParser("metrics", func(s *Step, v *Metrics) { s.Metrics = v }),
	"placement":   makeBlockParser("placement", func(s *Step, v *Placement) { s.Placement = v }),
}

// attributeParser defines a generic interface for parsing a simple attribute and setting it on a Step.
type attributeParser struct {
	Name   string
	Setter func(step *Step, expr hcl.Expression)
}

// attributeParsers is the table that drives the simple attribute parsing logic.
var attributeParsers = []attributeParser{
	{"enabled", func(s *Step, e hcl.Expression) { s.Enabled = &e }},
	{"description", func(s *Step, e hcl.Expression) { s.Description = &e }},
	{"tags", func(s *Step, e hcl.Expression) { s.Tags = e }},
	{"scope", func(s *Step, e hcl.Expression) { s.Scope = e }},
	{"uses", func(s *Step, e hcl.Expression) { s.Uses = e }},
	{"priority", func(s *Step, e hcl.Expression) { s.Priority = e }},
	{"delay_before", func(s *Step, e hcl.Expression) { s.DelayBefore = e }},
	{"delay_after", func(s *Step, e hcl.Expression) { s.DelayAfter = e }},
	{"continue_on_failure", func(s *Step, e hcl.Expression) { s.ContinueOnFailure = e }},
	{"idempotency_key", func(s *Step, e hcl.Expression) { s.IdempotencyKey = e }},
	{"sensitive", func(s *Step, e hcl.Expression) { s.Sensitive = &e }},
	{"env", func(s *Step, e hcl.Expression) { s.Env = e }},
}

// stepBodySchema defines the expected structure of a `step` block's body.
var stepBodySchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "enabled"}, {Name: "description"}, {Name: "tags"}, {Name: "scope"},
		{Name: "depends_on"}, {Name: "uses"}, {Name: "count"}, {Name: "for_each"},
		{Name: "priority"}, {Name: "delay_before"}, {Name: "delay_after"},
		{Name: "continue_on_failure"}, {Name: "idempotency_key"},
		{Name: "sensitive"}, {Name: "env"},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "arguments"}, {Type: "timeouts"}, {Type: "concurrency"},
		{Type: "rate_limit"}, {Type: "retry"}, {Type: "on_error"},
		{Type: "cache"}, {Type: "dedupe"}, {Type: "tracing"},
		{Type: "metrics"}, {Type: "placement"},
	},
}
