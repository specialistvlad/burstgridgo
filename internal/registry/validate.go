package registry

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/zclconf/go-cty/cty"
)

// ValidateRegistry performs a strict parity check between manifests and Go code.
func (r *Registry) ValidateRegistry(ctx context.Context) error {
	var allErrors []string
	logger := ctxlog.FromContext(ctx)

	for runnerType, def := range r.DefinitionRegistry {
		handler, ok := r.HandlerRegistry[def.Lifecycle.OnRun]
		if !ok {
			continue
		}

		if handler.InputType == nil {
			if len(def.Inputs) > 0 {
				allErrors = append(allErrors, fmt.Sprintf("runner '%s': manifest declares inputs, but Go handler has no input struct", runnerType))
			}
			continue
		}

		runnerLogger := logger.With("runner", runnerType, "go_handler", def.Lifecycle.OnRun)
		runnerLogger.Debug("Validating runner manifest against Go implementation.")

		manifestInputs := make(map[string]struct{})
		for name := range def.Inputs {
			manifestInputs[name] = struct{}{}
		}

		goInputFields := make(map[string]reflect.StructField)
		inputType := handler.InputType
		for i := 0; i < inputType.NumField(); i++ {
			field := inputType.Field(i)
			if !field.IsExported() {
				continue
			}
			tag := field.Tag.Get("bggo")
			tagName := strings.Split(tag, ",")[0]
			if tagName != "" && tagName != "-" {
				goInputFields[tagName] = field
			}
		}

		// Check for presence mismatches
		for name := range goInputFields {
			if _, ok := manifestInputs[name]; !ok {
				allErrors = append(allErrors, fmt.Sprintf("runner '%s': Go struct has field for input '%s' which is not declared in manifest", runnerType, name))
			}
		}
		for name := range manifestInputs {
			if _, ok := goInputFields[name]; !ok {
				allErrors = append(allErrors, fmt.Sprintf("runner '%s': manifest declares input '%s' which is not found in Go struct", runnerType, name))
			}
		}

		// Check for type mismatches using the refactored dispatcher.
		for name, inputDef := range def.Inputs {
			goField, ok := goInputFields[name]
			if !ok {
				continue
			}
			if err := r.validateTypeParity(ctx, runnerType, name, inputDef.Type, goField); err != nil {
				allErrors = append(allErrors, err.Error())
			}
		}
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("registry validation failed:\n- %s", strings.Join(allErrors, "\n- "))
	}

	logger.Debug("Registry validation completed successfully.")
	return nil
}

// validateTypeParity is the main dispatcher for type validation.
func (r *Registry) validateTypeParity(ctx context.Context, ownerName, inputPath string, manifestType cty.Type, goField reflect.StructField) error {
	logger := ctxlog.FromContext(ctx).With("input_path", inputPath)

	if manifestType.Equals(cty.DynamicPseudoType) {
		logger.Debug("Manifest type is 'any', skipping strict type check.")
		return nil
	}

	if manifestType.IsObjectType() {
		return r.validateObjectType(ctx, ownerName, inputPath, manifestType, goField)
	}

	return r.validatePrimitiveOrCollectionType(ctx, ownerName, inputPath, manifestType, goField)
}
