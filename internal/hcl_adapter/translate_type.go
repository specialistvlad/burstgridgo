// This file contains the logic for parsing HCL type expressions (e.g., `string`,
// `list(number)`) into their corresponding cty.Type objects.

package hcl_adapter

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

		if v.Name == "object" {
			logger.Debug("Parsing 'object' type constructor.")
			if len(v.Args) != 1 {
				return cty.DynamicPseudoType, fmt.Errorf("the object() type constructor requires exactly one argument (the object definition), got %d", len(v.Args))
			}

			objExpr, ok := v.Args[0].(*hclsyntax.ObjectConsExpr)
			if !ok {
				return cty.DynamicPseudoType, fmt.Errorf("the argument to object() must be an object literal like { key = type, ... }, got %T", v.Args[0])
			}

			if len(objExpr.Items) == 0 {
				logger.Debug("Detected generic 'object({})', creating empty object type.")
				return cty.Object(map[string]cty.Type{}), nil
			}

			attrTypes := make(map[string]cty.Type)
			logger.Debug("Parsing object attributes.", "count", len(objExpr.Items))

			for _, item := range objExpr.Items {
				logger.Debug("Inspecting key expression", "key_expr_type", fmt.Sprintf("%T", item.KeyExpr))
				var key string
				// 1. Check for the special wrapper type for object keys.
				if keyExpr, ok := item.KeyExpr.(*hclsyntax.ObjectConsKeyExpr); ok {
					// 2. Unwrap it and switch on the *actual* expression type inside.
					switch kexpr := keyExpr.Wrapped.(type) {
					case *hclsyntax.ScopeTraversalExpr:
						if len(kexpr.Traversal) == 1 {
							key = kexpr.Traversal.RootName()
						}
					case *hclsyntax.TemplateExpr:
						if len(kexpr.Parts) == 1 {
							if lit, isLit := kexpr.Parts[0].(*hclsyntax.LiteralValueExpr); isLit && lit.Val.Type().Equals(cty.String) {
								key = lit.Val.AsString()
							}
						}
					}
				}

				if key == "" {
					return cty.DynamicPseudoType, fmt.Errorf("invalid key in object type definition: keys must be simple identifiers or quoted strings, not complex expressions")
				}

				itemLogger := logger.With("attribute_name", key)
				itemLogger.Debug("Parsing attribute.")

				valueType, err := typeExprToCtyType(ctx, item.ValueExpr)
				if err != nil {
					return cty.DynamicPseudoType, fmt.Errorf("in object attribute '%s': %w", key, err)
				}
				itemLogger.Debug("Parsed attribute type.", "attribute_type", valueType.FriendlyName())
				attrTypes[key] = valueType
			}

			finalType := cty.Object(attrTypes)
			logger.Debug("Successfully constructed object type.", "final_type", finalType.FriendlyName())
			return finalType, nil
		}

		// Legacy logic for list, map, set
		if len(v.Args) != 1 {
			return cty.DynamicPseudoType, fmt.Errorf("type constructors (list, map, set) require exactly one argument, got %d", len(v.Args))
		}

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
		return cty.DynamicPseudoType, fmt.Errorf("unsupported expression for type definition: %T", v)
	}
}
