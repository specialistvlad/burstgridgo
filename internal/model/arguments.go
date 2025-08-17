// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Vladyslav Kazantsev
//
// This file provides a dedicated parser for a step's `arguments` block.
//
// Why a dedicated arguments parser?
//
// The `arguments` block is the primary mechanism for a user to pass data into
// a Runner. Its contents are not fixed; they are defined by the specific Runner
// being used. This function's purpose is to parse all attributes within the block
// into a generic map of names to their raw HCL expressions. A later validation
// step will then compare this map against the Runner's formal input schema
// (`RunnerInputDefinition`) to ensure correctness.
package model

import (
	"github.com/hashicorp/hcl/v2"
)

// parseArguments parses the attributes from a single "arguments" block.
// It returns a map of the argument names to their raw HCL expressions.
func parseArguments(block *hcl.Block) (map[string]hcl.Expression, hcl.Diagnostics) {
	attrs, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return nil, diags
	}

	args := make(map[string]hcl.Expression, len(attrs))
	for name, attr := range attrs {
		args[name] = attr.Expr
	}

	return args, diags
}
