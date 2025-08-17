// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Vladyslav Kazantsev
//
// // This file centralizes the parsing and validation for step looping constructs.
//
// Why centralize loop logic?
//
// A step can be executed multiple times using either `count` or `for_each`. These
// attributes are mutually exclusive and have specific type requirements. This file's
// purpose is to contain that specialized logic, ensuring that the looping
// configuration is valid at parse-time and providing clear errors for misconfiguration.
// This keeps the main `step.go` parser cleaner and focused on orchestration.
package model

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// parseCount finds the "count" attribute and performs static type validation on it.
func parseCount(attrs hcl.Attributes) (hcl.Expression, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	countAttr, exists := attrs["count"]
	if !exists {
		return nil, diags
	}

	// If the expression is a literal value, we can validate its type right now.
	if len(countAttr.Expr.Variables()) == 0 {
		val, valDiags := countAttr.Expr.Value(nil)
		diags = append(diags, valDiags...)
		if valDiags.HasErrors() {
			return countAttr.Expr, diags
		}

		if val.Type() != cty.Number {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid count value",
				Detail:   "The 'count' attribute must be a number.",
				Subject:  countAttr.Expr.Range().Ptr(),
			})
		} else {
			// Also, ensure the number is a whole number (no fractional part).
			bf := val.AsBigFloat()
			if !bf.IsInt() {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid count value",
					Detail:   "The 'count' attribute must be a whole number.",
					Subject:  countAttr.Expr.Range().Ptr(),
				})
			}
		}
	}

	return countAttr.Expr, diags
}

// parseForEach finds the "for_each" attribute and performs static type validation on it.
func parseForEach(attrs hcl.Attributes) (hcl.Expression, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	forEachAttr, exists := attrs["for_each"]
	if !exists {
		return nil, diags
	}

	// If the expression is a literal, we can validate its type.
	if len(forEachAttr.Expr.Variables()) == 0 {
		val, valDiags := forEachAttr.Expr.Value(nil)
		diags = append(diags, valDiags...)
		if valDiags.HasErrors() {
			return forEachAttr.Expr, diags
		}

		ty := val.Type()
		isCollection := ty.IsTupleType() || ty.IsListType() || ty.IsSetType() || ty.IsMapType()

		if !isCollection {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid for_each value",
				Detail:   "The 'for_each' attribute must be a map, a set of strings, or a list of strings.",
				Subject:  forEachAttr.Expr.Range().Ptr(),
			})
		} else if ty.IsTupleType() || ty.IsListType() {
			// If it's a list or tuple literal, all elements must be strings.
			it := val.ElementIterator()
			for it.Next() {
				_, v := it.Element()
				if v.Type() != cty.String {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid for_each value",
						Detail:   "When using a list or tuple for for_each, all elements must be strings.",
						Subject:  forEachAttr.Expr.Range().Ptr(),
					})
					// Break after the first error to avoid spamming diagnostics.
					break
				}
			}
		}
	}

	return forEachAttr.Expr, diags
}

// validateStepLoopingAttributes checks for conflicting looping attributes ('count' and 'for_each').
func validateStepLoopingAttributes(count, forEach hcl.Expression, body hcl.Body) hcl.Diagnostics {
	var diags hcl.Diagnostics
	if count != nil && forEach != nil {
		rng := body.MissingItemRange()
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Conflicting looping attributes",
			Detail:   "The 'count' and 'for_each' attributes cannot be used together in the same step.",
			Subject:  &rng,
		})
	}
	return diags
}
