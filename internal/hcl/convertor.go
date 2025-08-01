package hcl

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/gocty"
)

// Converter is the HCL-specific implementation of the config.Converter interface.
type Converter struct{}

// NewConverter creates a new HCL converter.
func NewConverter() *Converter {
	return &Converter{}
}

// DecodeBody evaluates HCL expressions, applies defaults, and populates the
// provided Go struct using reflection.
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
		field := structType.Field(i)
		fieldVal := structVal.Field(i)

		if !fieldVal.CanSet() {
			continue
		}

		lookupName := field.Name
		if tag := field.Tag.Get("bggo"); tag != "" {
			lookupName = strings.Split(tag, ",")[0]
		}

		inputDef, defExists := defs[lookupName]
		if !defExists {
			continue
		}

		targetPtr := fieldVal.Addr().Interface()
		argExpr, argProvided := args[lookupName]

		if argProvided {
			val, diags := argExpr.Value(evalCtx)
			if diags.HasErrors() {
				return diags
			}
			if err := c.decode(ctx, val, targetPtr); err != nil {
				return fmt.Errorf("failed to decode argument '%s': %w", lookupName, err)
			}
		} else {
			if inputDef.Default == nil && !inputDef.Optional {
				return fmt.Errorf("missing required argument %q", lookupName)
			}

			if inputDef.Default != nil {
				if err := c.decode(ctx, *inputDef.Default, targetPtr); err != nil {
					return fmt.Errorf("failed to apply default for '%s': %w", lookupName, err)
				}
			}
		}
	}
	logger.Debug("Finished HCL body decoding successfully.")
	return nil
}

// decode handles the conversion and decoding of a cty.Value into a Go pointer.
func (c *Converter) decode(ctx context.Context, val cty.Value, goVal any) error {
	logger := ctxlog.FromContext(ctx)
	valPtr := reflect.ValueOf(goVal)
	if valPtr.Kind() != reflect.Ptr {
		return fmt.Errorf("target for decoding must be a pointer, got %T", goVal)
	}

	impliedType, err := gocty.ImpliedType(valPtr.Elem().Interface())
	if err != nil {
		logger.Debug("Could not imply cty.Type from Go type, attempting direct decoding.", "go_type", valPtr.Elem().Type().String(), "error", err)
		return gocty.FromCtyValue(val, goVal)
	}

	logger.Debug("Preparing to decode value.",
		"source_type", val.Type().FriendlyName(),
		"target_type", impliedType.FriendlyName(),
	)

	convertedVal, err := convert.Convert(val, impliedType)
	if err != nil {
		return fmt.Errorf("cannot convert %s to required type %s: %w", val.Type().FriendlyName(), impliedType.FriendlyName(), err)
	}

	if !val.Type().Equals(convertedVal.Type()) {
		logger.Debug("Implicitly converted value type.",
			"from", val.Type().FriendlyName(),
			"to", convertedVal.Type().FriendlyName(),
		)
	}

	return gocty.FromCtyValue(convertedVal, goVal)
}

// ToCtyValue converts a native Go value into its corresponding cty.Value.
func (c *Converter) ToCtyValue(v any) (cty.Value, error) {
	if v == nil {
		return cty.NilVal, nil
	}
	ty, err := gocty.ImpliedType(v)
	if err != nil {
		return cty.NilVal, fmt.Errorf("unable to infer cty.Type: %w", err)
	}
	return gocty.ToCtyValue(v, ty)
}
