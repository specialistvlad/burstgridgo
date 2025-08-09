package hcl_adapter

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/gocty"
)

// decode is a recursive function that populates a Go value from a cty.Value,
// guided by a manifest-derived cty.Type.
func (c *Converter) decode(ctx context.Context, val cty.Value, manifestType cty.Type, goVal any) error {
	valPtr := reflect.ValueOf(goVal)
	goPtr := valPtr.Elem()
	goType := goPtr.Type()
	logger := ctxlog.FromContext(ctx).With("go_kind", goType.Kind().String())

	// If the target field in the Go struct is of type cty.Value, we don't need
	// to decode it further. We just assign the value directly.
	if goType == reflect.TypeOf(cty.Value{}) {
		logger.Debug("Target is cty.Value, performing direct assignment.")
		if val.IsKnown() { // Ensure we don't assign an unknown value directly
			goPtr.Set(reflect.ValueOf(val))
		}
		return nil
	}

	if !val.IsKnown() || val.IsNull() {
		logger.Debug("Skipping decode for null or unknown value.")
		return nil // Nothing to decode.
	}

	switch goType.Kind() {
	case reflect.Struct:
		logger.Debug("Decoding as struct.")
		if !val.Type().IsObjectType() && val.Type() != cty.DynamicPseudoType {
			return fmt.Errorf("type mismatch: cannot decode cty value of type %s into Go struct %s", val.Type().FriendlyName(), goType.String())
		}
		if !manifestType.IsObjectType() && manifestType != cty.DynamicPseudoType {
			return fmt.Errorf("type mismatch: manifest expected an object for Go struct %s, but got %s", goType.String(), manifestType.FriendlyName())
		}

		isManifestObject := manifestType.IsObjectType()
		attrMap := val.AsValueMap()

		for i := 0; i < goType.NumField(); i++ {
			fieldDef := goType.Field(i)
			fieldVal := goPtr.Field(i)

			if !fieldDef.IsExported() || !fieldVal.CanSet() {
				continue
			}

			tagName := fieldDef.Tag.Get("cty")
			tagName = strings.Split(tagName, ",")[0]
			if tagName == "" || tagName == "-" {
				continue
			}

			attrVal, ok := attrMap[tagName]
			if !ok {
				continue
			}

			var attrManifestType cty.Type
			if isManifestObject {
				attrManifestType = manifestType.AttributeTypes()[tagName]
			} else {
				attrManifestType = attrVal.Type()
			}

			if err := c.decode(ctx, attrVal, attrManifestType, fieldVal.Addr().Interface()); err != nil {
				return fmt.Errorf("in attribute '%s': %w", tagName, err)
			}
		}
		return nil

	case reflect.Interface: // This handles 'any'
		logger.Debug("Decoding as interface (any).")
		nativeVal, err := ctyToNative(val)
		if err != nil {
			return err
		}
		if nativeVal != nil {
			goPtr.Set(reflect.ValueOf(nativeVal))
		}
		return nil

	case reflect.Map:
		return c.decodeMap(ctx, val, manifestType, goPtr)

	case reflect.Slice:
		logger.Debug("Decoding as slice.")
		if !val.Type().IsListType() && !val.Type().IsTupleType() {
			return fmt.Errorf("type mismatch: cannot decode cty.%s into Go slice %s", val.Type().FriendlyName(), goType.String())
		}
		if (!manifestType.IsListType() && !manifestType.IsTupleType()) && manifestType != cty.DynamicPseudoType {
			return fmt.Errorf("type mismatch: manifest expected a list for Go slice %s, but got %s", goType.String(), manifestType.FriendlyName())
		}

		if val.Type().IsTupleType() {
			logger.Debug("Value is a tuple, converting to list before decoding to slice.")
			goElemType := goType.Elem()
			ctyElemType, err := gocty.ImpliedType(reflect.Zero(goElemType).Interface())
			if err != nil {
				return fmt.Errorf("cannot imply cty type for slice element %s: %w", goElemType.String(), err)
			}

			listVal, err := convert.Convert(val, cty.List(ctyElemType))
			if err != nil {
				return fmt.Errorf("cannot convert tuple to a uniform list for slice %s: %w", goType.String(), err)
			}
			val = listVal
		}

		newSlice := reflect.MakeSlice(goType, val.LengthInt(), val.LengthInt())
		var elemManifestType cty.Type
		if manifestType.IsListType() || manifestType.IsTupleType() {
			elemManifestType = manifestType.ElementType()
		} else {
			elemManifestType = val.Type().ElementType()
		}

		it := val.ElementIterator()
		for i := 0; it.Next(); i++ {
			_, elemVal := it.Element()
			if err := c.decode(ctx, elemVal, elemManifestType, newSlice.Index(i).Addr().Interface()); err != nil {
				return fmt.Errorf("in slice element %d: %w", i, err)
			}
		}
		goPtr.Set(newSlice)
		return nil

	default: // Base cases for primitives (string, int, bool, float64, etc.)
		logger.Debug("Decoding as primitive.")
		convertedVal, err := convert.Convert(val, manifestType)
		if err != nil {
			return fmt.Errorf("cannot convert value of type %s to required manifest type %s: %w", val.Type().FriendlyName(), manifestType.FriendlyName(), err)
		}
		return gocty.FromCtyValue(convertedVal, goVal)
	}
}
