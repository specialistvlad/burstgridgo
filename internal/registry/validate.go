package registry

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// ValidateRegistry performs a strict parity check between manifests and Go code.
// It checks both the presence of inputs and the compatibility of their types.
func (r *Registry) ValidateRegistry(ctx context.Context) error {
	var errs []string
	logger := ctxlog.FromContext(ctx)

	for runnerType, def := range r.DefinitionRegistry {
		handler, ok := r.HandlerRegistry[def.Lifecycle.OnRun]
		if !ok {
			continue
		}

		if handler.InputType == nil {
			if len(def.Inputs) > 0 {
				errs = append(errs, fmt.Sprintf("runner '%s': manifest declares inputs, but Go handler has no input struct", runnerType))
			}
			continue
		}

		hclInputs := make(map[string]struct{})
		for name := range def.Inputs {
			hclInputs[name] = struct{}{}
		}

		goInputs := make(map[string]reflect.StructField)
		inputType := handler.InputType
		for i := 0; i < inputType.NumField(); i++ {
			field := inputType.Field(i)
			if !field.IsExported() {
				continue
			}
			tag := field.Tag.Get("bggo")
			tagName := strings.Split(tag, ",")[0]
			if tagName != "" && tagName != "-" {
				goInputs[tagName] = field
			}
		}

		// Check for presence mismatches
		for name := range goInputs {
			if _, ok := hclInputs[name]; !ok {
				errs = append(errs, fmt.Sprintf("runner '%s': Go struct has field for input '%s' which is not declared in manifest", runnerType, name))
			}
		}
		for name := range hclInputs {
			if _, ok := goInputs[name]; !ok {
				errs = append(errs, fmt.Sprintf("runner '%s': manifest declares input '%s' which is not found in Go struct", runnerType, name))
			}
		}

		// Check for type mismatches
		for name, inputDef := range def.Inputs {
			goField, ok := goInputs[name]
			if !ok {
				continue // Already handled by presence check
			}

			manifestType := inputDef.Type
			if manifestType.Equals(cty.DynamicPseudoType) {
				logger.Warn("Manifest for runner has input with 'type = any', which disables static type checking. Consider using a specific type like 'string', 'number', or 'bool'.", "runner", runnerType, "input", name)
				continue
			}

			// Infer type from the Go field
			goFieldType, err := gocty.ImpliedType(reflect.Zero(goField.Type).Interface())
			if err != nil {
				errs = append(errs, fmt.Sprintf("runner '%s', input '%s': could not imply cty type from Go field type %s: %v", runnerType, name, goField.Type, err))
				continue
			}

			// The core type check
			if !manifestType.Equals(goFieldType) {
				errs = append(errs, fmt.Sprintf("runner '%s', input '%s': type mismatch. Manifest requires '%s' but Go struct field '%s' provides compatible type '%s'",
					runnerType, name, manifestType.FriendlyName(), goField.Name, goFieldType.FriendlyName()))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("registry validation failed:\n- %s", strings.Join(errs, "\n- "))
	}

	return nil
}
