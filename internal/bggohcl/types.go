package bggohcl

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// HCLTypeToCtyType converts an HCL expression that represents a type (e.g., the `string`
// keyword) into its corresponding cty.Type. This function is comprehensive, meaning it
// recognizes all HCL type keywords but will panic if a type is not yet implemented.
func HCLTypeToCtyType(expr hcl.Expression) (cty.Type, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	// We expect a simple identifier like `string`, not a complex expression.
	// AbsTraversalForExpr is the right tool to validate this structure.
	traversal, hclDiags := hcl.AbsTraversalForExpr(expr)
	if hclDiags.HasErrors() || len(traversal) != 1 {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid type specification",
			Detail:   "The 'type' attribute must be a simple type keyword like 'string', 'number', or 'bool', not a complex expression.",
			Subject:  expr.Range().Ptr(),
		})
		return cty.NilType, diags
	}

	typeName := traversal.RootName()
	if typeName == "" {
		// This should be unreachable given the checks above, but serves as a safeguard.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid type specification",
			Detail:   "Internal error: Could not read type keyword.",
			Subject:  expr.Range().Ptr(),
		})
		return cty.NilType, diags
	}

	switch typeName {
	// --- Implemented Primitive Types ---
	case "string":
		return cty.String, diags
	case "number":
		return cty.Number, diags
	case "bool":
		return cty.Bool, diags
	case "any":
		// While 'any' is a valid constraint, we may want to be more strict in runners.
		// For now, let's consider it unimplemented for explicit input types.
		panic(fmt.Sprintf("PANIC: The type '%s' is not yet implemented.", typeName))

	// --- Unimplemented Complex Types (Placeholders) ---
	case "list", "map", "set", "object", "tuple":
		// These are valid HCL types, but we are explicitly not supporting them in this phase.
		// A panic is appropriate here to halt development if they are used by mistake,
		// signaling that a new feature implementation is required.
		panic(fmt.Sprintf("PANIC: The complex type '%s' is not yet implemented.", typeName))

	// --- Invalid Type Keyword ---
	default:
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsupported type",
			Detail:   fmt.Sprintf("The keyword '%s' is not a valid type. Supported types are: string, number, bool.", typeName),
			Subject:  expr.Range().Ptr(),
		})
		return cty.NilType, diags
	}
}
