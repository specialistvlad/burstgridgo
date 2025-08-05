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
	var allErrors []string
	logger := ctxlog.FromContext(ctx)

	for runnerType, def := range r.DefinitionRegistry {
		handler, ok := r.HandlerRegistry[def.Lifecycle.OnRun]
		if !ok {
			// This can happen for runners that are defined but not used by any module.
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

		// Check for type mismatches using the new recursive validator
		for name, inputDef := range def.Inputs {
			goField, ok := goInputFields[name]
			if !ok {
				continue // Already handled by presence check
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

// validateTypeParity recursively compares a manifest type with a Go field's type.
func (r *Registry) validateTypeParity(ctx context.Context, ownerName, inputPath string, manifestType cty.Type, goField reflect.StructField) error {
	logger := ctxlog.FromContext(ctx).With(
		"owner", ownerName,
		"input_path", inputPath,
		"manifest_type", manifestType.FriendlyName(),
		"go_type", goField.Type.String(),
		"go_field", goField.Name,
	)
	logger.Debug("Performing type parity validation.")

	// Case 1: The manifest type is 'any', which allows any Go type.
	if manifestType.Equals(cty.DynamicPseudoType) {
		logger.Debug("Manifest type is 'any', skipping strict type check.")
		return nil
	}

	// Case 2: The manifest type is an object. This requires recursive validation.
	if manifestType.IsObjectType() {
		logger.Debug("Validating as an object type.")
		goFieldType := goField.Type
		if goFieldType.Kind() == reflect.Pointer {
			goFieldType = goFieldType.Elem()
		}

		if goFieldType.Kind() != reflect.Struct {
			return fmt.Errorf("runner '%s', input '%s': type mismatch. Manifest requires an 'object', but Go field '%s' is not a struct (it's a %s)", ownerName, inputPath, goField.Name, goFieldType.Kind())
		}

		manifestAttrs := manifestType.AttributeTypes()
		goStructFields := getStructFieldsByCtyTag(goFieldType)

		// Check that all manifest attributes exist in the Go struct.
		for attrName := range manifestAttrs {
			if _, ok := goStructFields[attrName]; !ok {
				return fmt.Errorf("runner '%s', input '%s': attribute '%s' is required by the manifest but is missing from Go struct '%s'", ownerName, inputPath, attrName, goFieldType.Name())
			}
		}

		// Check that all Go struct fields are defined in the manifest.
		for tagName, field := range goStructFields {
			if _, ok := manifestAttrs[tagName]; !ok {
				return fmt.Errorf("runner '%s', input '%s': Go struct '%s' has field '%s' (cty tag '%s') which is not defined in the manifest object type", ownerName, inputPath, goFieldType.Name(), field.Name, tagName)
			}
		}

		// Recursively validate the types of each attribute.
		for attrName, manifestAttrType := range manifestAttrs {
			goStructField := goStructFields[attrName]
			err := r.validateTypeParity(ctx, ownerName, fmt.Sprintf("%s.%s", inputPath, attrName), manifestAttrType, goStructField)
			if err != nil {
				// We need to re-wrap the error to provide the full context path.
				return fmt.Errorf("runner '%s', input '%s': attribute '%s' type mismatch: %w", ownerName, inputPath, attrName, err)
			}
		}
		return nil
	}

	// Case 3: Primitive or collection types.
	logger.Debug("Validating as a primitive or collection type.")
	impliedGoType, err := gocty.ImpliedType(reflect.Zero(goField.Type).Interface())
	if err != nil {
		return fmt.Errorf("runner '%s', input '%s': could not imply cty type from Go field type %s: %w", ownerName, inputPath, goField.Type, err)
	}

	if !manifestType.Equals(impliedGoType) {
		// The base case for the recursion returns the most specific error.
		if strings.Contains(inputPath, ".") {
			return fmt.Errorf("manifest requires '%s', but Go struct field '%s' provides '%s'", manifestType.FriendlyName(), goField.Name, impliedGoType.FriendlyName())
		}
		return fmt.Errorf("runner '%s', input '%s': type mismatch. Manifest requires '%s' but Go struct field '%s' provides compatible type '%s'",
			ownerName, inputPath, manifestType.FriendlyName(), goField.Name, impliedGoType.FriendlyName())
	}

	return nil
}

// getStructFieldsByCtyTag inspects a struct type and returns a map of its
// fields keyed by their `cty:"..."` tag.
func getStructFieldsByCtyTag(structType reflect.Type) map[string]reflect.StructField {
	fields := make(map[string]reflect.StructField)
	if structType.Kind() != reflect.Struct {
		return fields
	}
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if !field.IsExported() {
			continue
		}
		tag := field.Tag.Get("cty")
		tagName := strings.Split(tag, ",")[0]
		if tagName != "" && tagName != "-" {
			fields[tagName] = field
		}
	}
	return fields
}
