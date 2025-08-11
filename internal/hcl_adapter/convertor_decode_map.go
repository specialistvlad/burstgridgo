package hcl_adapter

import (
	"context"
	"fmt"
	"reflect"

	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/zclconf/go-cty/cty"
)

// decodeMap handles the recursive decoding of a cty.Value into a Go map.
// It contains a fast path for generic map[string]any and a deep-decode path
// for typed maps, and correctly handles decoding cty objects into maps.
func (c *Converter) decodeMap(ctx context.Context, val cty.Value, manifestType cty.Type, goPtr reflect.Value) error {
	logger := ctxlog.FromContext(ctx).With("go_type", goPtr.Type().String(), "cty_type", val.Type().FriendlyName())
	logger.Debug("Decoding into Go map.")

	// Fast path for generic objects into map[string]any, which is a common case.
	if goPtr.Type() == reflect.TypeOf((map[string]any)(nil)) {
		logger.Debug("Using fast path for map[string]any via ctyToNative.")
		nativeVal, err := ctyToNative(val)
		if err != nil {
			return err
		}
		if nativeVal != nil {
			goPtr.Set(reflect.ValueOf(nativeVal))
		}
		return nil
	}

	logger.Debug("Performing deep decode for typed map.")
	newMap := reflect.MakeMap(goPtr.Type())
	it := val.ElementIterator()

	for it.Next() {
		key, elemVal := it.Element()
		keyStr := key.AsString()
		elemLogger := logger.With("map_key", keyStr)
		elemLogger.Debug("Processing map element.")

		var elemManifestType cty.Type
		if manifestType.IsMapType() {
			// If the parent manifest specifies a map, all elements must conform to its element type.
			elemManifestType = manifestType.ElementType()
		} else {
			// ** THE FIX **
			// If the manifest is 'any' or an object, the element's own type is
			// the "manifest" for the recursive call. This prevents panics when
			// val is a cty.Object.
			elemManifestType = elemVal.Type()
		}

		newElemPtr := reflect.New(goPtr.Type().Elem())
		if err := c.decode(ctx, elemVal, elemManifestType, newElemPtr.Interface()); err != nil {
			return fmt.Errorf("failed to decode map element '%s': %w", keyStr, err)
		}
		newMap.SetMapIndex(reflect.ValueOf(keyStr), newElemPtr.Elem())
	}
	goPtr.Set(newMap)
	logger.Debug("Successfully decoded into Go map.")
	return nil
}
