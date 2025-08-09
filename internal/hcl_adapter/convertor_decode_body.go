package hcl_adapter

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/zclconf/go-cty/cty"
)

// DecodeBody iterates through the fields of a Go struct, finds the corresponding
// HCL arguments, and uses the recursive `decode` helper to populate them.
func (c *Converter) DecodeBody(
	ctx context.Context,
	inputStruct any,
	args map[string]hcl.Expression,
	defs map[string]*config.InputDefinition,
	evalCtx *hcl.EvalContext,
) error {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Starting HCL body decoding.")

	structVal := reflect.ValueOf(inputStruct)
	if structVal.Kind() != reflect.Ptr || structVal.IsNil() {
		return fmt.Errorf("inputStruct must be a non-nil pointer")
	}
	structVal = structVal.Elem()
	structType := structVal.Type()

	for i := 0; i < structType.NumField(); i++ {
		fieldDef := structType.Field(i)
		fieldVal := structVal.Field(i)

		if !fieldDef.IsExported() || !fieldVal.CanSet() {
			continue
		}

		tagName := fieldDef.Tag.Get("bggo")
		tagName = strings.Split(tagName, ",")[0]
		if tagName == "" || tagName == "-" {
			continue
		}

		inputDef, ok := defs[tagName]
		if !ok {
			continue // No definition for this field, skip.
		}

		var valueToDecode cty.Value
		argExpr, provided := args[tagName]

		if provided {
			val, diags := argExpr.Value(evalCtx)
			if diags.HasErrors() {
				return diags
			}
			valueToDecode = val
		} else {
			if inputDef.Default != nil {
				valueToDecode = *inputDef.Default
			} else if inputDef.Optional {
				continue
			} else {
				return fmt.Errorf("missing required argument %q", tagName)
			}
		}

		if err := c.decode(ctx, valueToDecode, inputDef.Type, fieldVal.Addr().Interface()); err != nil {
			return fmt.Errorf("failed to decode argument '%s': %w", tagName, err)
		}
	}
	logger.Debug("Finished HCL body decoding successfully.")
	return nil
}
