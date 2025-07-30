package executor

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/schema"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// applyInputDefaults applies default values from a runner's manifest.
func applyInputDefaults(ctx context.Context, inputStruct any, runnerDef *schema.RunnerDefinition, userBody hcl.Body) error {
	logger := ctxlog.FromContext(ctx)
	if inputStruct == nil || runnerDef == nil || userBody == nil {
		return nil
	}

	userAttrs, _ := userBody.JustAttributes()
	userProvidedNames := make(map[string]struct{})
	for name := range userAttrs {
		userProvidedNames[name] = struct{}{}
	}

	structVal := reflect.ValueOf(inputStruct).Elem()
	structType := structVal.Type()

	for _, inputDef := range runnerDef.Inputs {
		if _, ok := userProvidedNames[inputDef.Name]; ok || inputDef.Default == nil {
			continue
		}
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			tagName := strings.Split(field.Tag.Get("hcl"), ",")[0]
			if tagName == inputDef.Name {
				fieldVal := structVal.Field(i)
				if fieldVal.CanSet() {
					logger.Debug("Applying default value.", "field", tagName, "value", *inputDef.Default)
					if err := gocty.FromCtyValue(*inputDef.Default, fieldVal.Addr().Interface()); err != nil {
						return fmt.Errorf("failed to apply default for '%s': %w", inputDef.Name, err)
					}
				}
				break
			}
		}
	}
	return nil
}

// ctyValueToInterface converts a cty.Value to a Go interface{}.
func ctyValueToInterface(val cty.Value) (any, error) {
	if !val.IsKnown() || val.IsNull() {
		return nil, nil
	}
	if val.Type().IsPrimitiveType() {
		switch val.Type() {
		case cty.String:
			return val.AsString(), nil
		case cty.Number:
			f, _ := val.AsBigFloat().Float64()
			return f, nil
		case cty.Bool:
			return val.True(), nil
		default:
			return nil, fmt.Errorf("unsupported primitive type: %s", val.Type().FriendlyName())
		}
	}
	if val.Type().IsObjectType() || val.Type().IsMapType() {
		out := make(map[string]any)
		for it := val.ElementIterator(); it.Next(); {
			k, v := it.Element()
			valInterface, err := ctyValueToInterface(v)
			if err != nil {
				return nil, err
			}
			out[k.AsString()] = valInterface
		}
		return out, nil
	}
	if val.Type().IsTupleType() || val.Type().IsListType() {
		var out []any
		for it := val.ElementIterator(); it.Next(); {
			_, v := it.Element()
			valInterface, err := ctyValueToInterface(v)
			if err != nil {
				return nil, err
			}
			out = append(out, valInterface)
		}
		return out, nil
	}
	return nil, fmt.Errorf("unsupported cty.Type for conversion: %s", val.Type().FriendlyName())
}

// formatValueForLogs converts a value to its loggable representation.
// For cty.Value, it's converted to a Go interface. Other types are passed through.
func formatValueForLogs(v any) any {
	if ctyVal, ok := v.(cty.Value); ok {
		converted, err := ctyValueToInterface(ctyVal)
		if err != nil {
			return fmt.Sprintf("[unloggable cty.Value: %v]", err)
		}
		return converted
	}
	return v
}
