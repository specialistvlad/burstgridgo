package registry

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// validateObjectType handles validation for both generic and structured objects.
func (r *Registry) validateObjectType(ctx context.Context, ownerName, inputPath string, manifestType cty.Type, goField reflect.StructField) error {
	logger := ctxlog.FromContext(ctx).With("input_path", inputPath)
	isGenericObject := len(manifestType.AttributeTypes()) == 0

	if isGenericObject {
		logger.Debug("Validating as a generic object, expecting map[string]any.")
		expectedGoType := reflect.TypeOf((map[string]any)(nil))
		if goField.Type != expectedGoType {
			return fmt.Errorf("runner '%s', input '%s': manifest requires a generic object 'object({})', but Go field '%s' is not a 'map[string]any' (it's a '%s')", ownerName, inputPath, goField.Name, goField.Type.String())
		}
		return nil
	}

	logger.Debug("Validating as a structurally-typed object, expecting a struct.")
	goFieldType := goField.Type
	if goFieldType.Kind() == reflect.Pointer {
		goFieldType = goFieldType.Elem()
	}

	if goFieldType.Kind() != reflect.Struct {
		return fmt.Errorf("runner '%s', input '%s': type mismatch. Manifest requires a structured 'object', but Go field '%s' is not a struct (it's a %s)", ownerName, inputPath, goField.Name, goFieldType.Kind())
	}

	manifestAttrs := manifestType.AttributeTypes()
	goStructFields := getStructFieldsByCtyTag(goFieldType)

	for attrName := range manifestAttrs {
		if _, ok := goStructFields[attrName]; !ok {
			return fmt.Errorf("runner '%s', input '%s': attribute '%s' is required by the manifest but is missing from Go struct '%s'", ownerName, inputPath, attrName, goFieldType.Name())
		}
	}

	for tagName, field := range goStructFields {
		if _, ok := manifestAttrs[tagName]; !ok {
			return fmt.Errorf("runner '%s', input '%s': Go struct '%s' has field '%s' (cty tag '%s') which is not defined in the manifest object type", ownerName, inputPath, goFieldType.Name(), field.Name, tagName)
		}
	}

	for attrName, manifestAttrType := range manifestAttrs {
		goStructField := goStructFields[attrName]
		err := r.validateTypeParity(ctx, ownerName, fmt.Sprintf("%s.%s", inputPath, attrName), manifestAttrType, goStructField)
		if err != nil {
			return fmt.Errorf("runner '%s', input '%s': attribute '%s' type mismatch: %w", ownerName, inputPath, attrName, err)
		}
	}
	return nil
}

// validatePrimitiveOrCollectionType handles validation for non-object types.
func (r *Registry) validatePrimitiveOrCollectionType(ctx context.Context, ownerName, inputPath string, manifestType cty.Type, goField reflect.StructField) error {
	logger := ctxlog.FromContext(ctx).With("input_path", inputPath)
	logger.Debug("Validating as a primitive or collection type.")

	impliedGoType, err := gocty.ImpliedType(reflect.Zero(goField.Type).Interface())
	if err != nil {
		return fmt.Errorf("runner '%s', input '%s': could not imply cty type from Go field type %s: %w", ownerName, inputPath, goField.Type, err)
	}

	if !manifestType.Equals(impliedGoType) {
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
