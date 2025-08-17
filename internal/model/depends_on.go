// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Vladyslav Kazantsev
//
// This file contains the specific parsing and validation logic for the `depends_on` attribute.
//
// Why a special parser for depends_on?
//
// Unlike simple value attributes, `depends_on` has a critical structural role:
// it defines explicit edges in the execution graph. Its purpose is to allow users
// to enforce an ordering dependency between steps that don't have an implicit
// data dependency (i.e., one step's argument referencing another's output).
// This parser ensures the attribute's value is a list literal, as required for
// building the dependency graph.
package model

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// parseDependsOn finds the "depends_on" attribute and returns its raw expression
// for dependency analysis. It also validates that the expression is a list.
func parseDependsOn(attrs hcl.Attributes) (hcl.Expression, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	dependsOnAttr, exists := attrs["depends_on"]
	if !exists {
		// The attribute is optional, so it's not an error if it's missing.
		return nil, diags
	}
	expr := dependsOnAttr.Expr

	// The expression must be a tuple constructor, i.e., a list literal like `[...]`.
	if syntaxExpr, ok := expr.(hclsyntax.Expression); ok {
		if _, isTuple := syntaxExpr.(*hclsyntax.TupleConsExpr); !isTuple {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid depends_on value",
				Detail:   "The 'depends_on' attribute must be a list of step references.",
				Subject:  expr.Range().Ptr(),
			})
		}
	}

	return expr, diags
}
