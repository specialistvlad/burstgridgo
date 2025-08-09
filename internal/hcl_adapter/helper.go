package hcl_adapter

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/zclconf/go-cty/cty"
)

// isExprDefined checks if an HCL expression was actually present in the source
// code. The HCL decoder often populates optional fields with non-nil, zero-width
// expression objects, so a simple nil check is insufficient. This helper
// provides a robust way to check for genuine user-provided attributes.
func isExprDefined(ctx context.Context, expr hcl.Expression, attrName string) bool {
	logger := ctxlog.FromContext(ctx)

	if expr == nil {
		logger.Debug("Expression is nil, considering it undefined.", "attribute", attrName)
		return false
	}

	// The most reliable check is to see if the expression's source range has a
	// physical size. A real attribute occupies bytes in the file, while a
	// placeholder for an omitted optional attribute has a zero-width range
	// where the start and end byte are the same.
	exprRange := expr.Range()
	isDefined := exprRange.End.Byte > exprRange.Start.Byte

	logger.Debug("Checking if HCL attribute was explicitly defined.",
		"attribute", attrName,
		"hcl_range", exprRange.String(),
		"start_byte", exprRange.Start.Byte,
		"end_byte", exprRange.End.Byte,
		"is_defined", isDefined,
	)

	return isDefined
}

// extractBodyAttributes converts block bodies into a map of expressions.
func (l *Loader) extractBodyAttributes(block interface{}) map[string]hcl.Expression {
	if block == nil {
		return nil
	}
	var body hcl.Body
	switch b := block.(type) {
	case *StepArgs:
		if b == nil {
			return nil
		}
		body = b.Body
	case *UsesBlock:
		if b == nil {
			return nil
		}
		body = b.Body
	default:
		return nil
	}
	if body == nil {
		return nil
	}
	attrs, _ := body.JustAttributes()
	if attrs == nil {
		return nil
	}
	exprMap := make(map[string]hcl.Expression)
	for name, attr := range attrs {
		exprMap[name] = attr.Expr
	}
	return exprMap
}

// translateInputDefinition is a helper that processes a single HCL input
// block, handling its default value and type parsing.
func translateInputDefinition(ctx context.Context, in *InputDefinition, ownerKind, ownerName string) (*config.InputDefinition, error) {
	var defaultVal *cty.Value
	var isOptional bool

	if in.Default != nil {
		val, diags := in.Default.Value(nil)
		if diags.HasErrors() {
			return nil, fmt.Errorf("invalid default value for input '%s' in %s '%s': %w", in.Name, ownerKind, ownerName, diags)
		}
		if !val.IsNull() {
			defaultVal = &val
			isOptional = true
		}
	}

	parsedType, err := typeExprToCtyType(ctx, in.Type)
	if err != nil {
		return nil, fmt.Errorf("in %s '%s', input '%s': %w", ownerKind, ownerName, in.Name, err)
	}

	return &config.InputDefinition{
		Name:        in.Name,
		Type:        parsedType,
		Description: in.Description,
		Default:     defaultVal,
		Optional:    isOptional,
	}, nil
}
