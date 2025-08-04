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

		fieldLogger := logger.With("go_field", field.Name, "hcl_input", lookupName)

		inputDef, defExists := defs[lookupName]
		if !defExists {
			continue
		}

		targetPtr := fieldVal.Addr().Interface()
		argExpr, argProvided := args[lookupName]

		var valueToDecode cty.Value
		var valueSource string

		if argProvided {
			val, diags := argExpr.Value(evalCtx)
			if diags.HasErrors() {
				return diags
			}
			valueToDecode = val
			valueSource = "user-provided"
			fieldLogger.Debug("Received value from user.", "value_source", valueSource, "raw_value", val.GoString(), "raw_type", val.Type().FriendlyName())
		} else {
			if inputDef.Default == nil && !inputDef.Optional {
				return fmt.Errorf("missing required argument %q", lookupName)
			}
			if inputDef.Default != nil {
				valueToDecode = *inputDef.Default
				valueSource = "default"
				fieldLogger.Debug("Applying default value.", "value_source", valueSource, "default_value", valueToDecode.GoString(), "default_type", valueToDecode.Type().FriendlyName())
			} else {
				fieldLogger.Debug("Optional field not provided and no default exists. Skipping.")
				continue
			}
		}

		// Perform type conversion based on the manifest's definition.
		manifestType := inputDef.Type
		fieldLogger.Debug("Starting conversion based on manifest type.", "manifest_type", manifestType.FriendlyName())
		if !manifestType.Equals(cty.DynamicPseudoType) {
			var err error
			convertedValue, err := convert.Convert(valueToDecode, manifestType)
			if err != nil {
				return fmt.Errorf("failed to convert %s value for argument '%s': %w", valueSource, lookupName, err)
			}
			if !convertedValue.Type().Equals(valueToDecode.Type()) {
				fieldLogger.Debug("Value was converted to match manifest type.", "from_type", valueToDecode.Type().FriendlyName(), "to_type", convertedValue.Type().FriendlyName(), "new_value", convertedValue.GoString())
			}
			valueToDecode = convertedValue
		}

		if err := c.decode(ctx, valueToDecode, targetPtr); err != nil {
			return fmt.Errorf("failed to decode argument '%s' into Go struct field: %w", lookupName, err)
		}
	}
	logger.Debug("Finished HCL body decoding successfully.")
	return nil
}

// decode handles the final conversion and decoding of a cty.Value into a Go pointer.
func (c *Converter) decode(ctx context.Context, val cty.Value, goVal any) error {
	logger := ctxlog.FromContext(ctx)
	valPtr := reflect.ValueOf(goVal)
	if valPtr.Kind() != reflect.Ptr {
		return fmt.Errorf("target for decoding must be a pointer, got %T", goVal)
	}

	goType := valPtr.Elem().Type()
	decodeLogger := logger.With("target_go_type", goType.String())

	impliedType, err := gocty.ImpliedType(valPtr.Elem().Interface())
	if err != nil {
		decodeLogger.Debug("Could not imply cty.Type from Go type, attempting direct unsafe decoding.", "error", err)
		return gocty.FromCtyValue(val, goVal)
	}
	decodeLogger.Debug("Implied cty.Type from Go type.", "implied_cty_type", impliedType.FriendlyName())

	convertedVal, err := convert.Convert(val, impliedType)
	if err != nil {
		return fmt.Errorf("cannot convert value of type %s to required Go type %s (cty type %s): %w", val.Type().FriendlyName(), goType.String(), impliedType.FriendlyName(), err)
	}

	if !val.Type().Equals(convertedVal.Type()) {
		decodeLogger.Debug("Value was converted to match final Go type.",
			"from_type", val.Type().FriendlyName(),
			"to_type", convertedVal.Type().FriendlyName(),
		)
	}

	decodeLogger.Debug("Final value decoding into Go struct.")
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
