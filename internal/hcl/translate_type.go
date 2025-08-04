// This file contains the logic for parsing HCL type expressions (e.g., `string`,
// `list(number)`) into their corresponding cty.Type objects.

package hcl

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/zclconf/go-cty/cty"
)

// typeExprToCtyType converts an HCL type expression into its cty.Type equivalent.
func typeExprToCtyType(ctx context.Context, expr hcl.Expression) (cty.Type, error) {
	logger := ctxlog.FromContext(ctx)

	if expr == nil {
		logger.Debug("Type expression is nil, defaulting to any.")
		return cty.DynamicPseudoType, nil
	}

	// Using a type switch is the correct way to handle the various concrete
	// expression types that implement the hcl.Expression interface.
	switch v := expr.(type) {
	case *hclsyntax.FunctionCallExpr:
		logger.Debug("Parsing type expression as a function call.", "call", v.Name)
		if len(v.Args) != 1 {
			return cty.DynamicPseudoType, fmt.Errorf("type constructors (list, map, set) require exactly one argument, got %d", len(v.Args))
		}

		// Recursively parse the inner type.
		elementType, err := typeExprToCtyType(ctx, v.Args[0])
		if err != nil {
			return cty.DynamicPseudoType, err
		}
		if elementType == cty.DynamicPseudoType {
			return cty.DynamicPseudoType, fmt.Errorf("collection types cannot contain type 'any'")
		}
		logger.Debug("Parsed collection element type.", "type", elementType.FriendlyName())

		switch v.Name {
		case "list":
			return cty.List(elementType), nil
		case "map":
			return cty.Map(elementType), nil
		case "set":
			return cty.Set(elementType), nil
		default:
			return cty.DynamicPseudoType, fmt.Errorf("unknown type constructor function %q", v.Name)
		}

	case *hclsyntax.ScopeTraversalExpr:
		// This handles primitive type identifiers like `string` or `number`.
		if len(v.Traversal) != 1 {
			return cty.DynamicPseudoType, fmt.Errorf("invalid type keyword: traversal path is not a single identifier")
		}
		rootName := v.Traversal.RootName()
		logger.Debug("Parsing type expression as a primitive.", "keyword", rootName)
		switch rootName {
		case "string":
			return cty.String, nil
		case "number":
			return cty.Number, nil
		case "bool":
			return cty.Bool, nil
		case "any":
			return cty.DynamicPseudoType, nil
		default:
			return cty.DynamicPseudoType, fmt.Errorf("unknown primitive type %q", rootName)
		}

	default:
		// Fallback for any other kind of expression.
		return cty.DynamicPseudoType, fmt.Errorf("unsupported expression for type definition: %T", v)
	}
}
