// This file contains the logic for converting an arbitrary cty.Value into its
// native Go representation (interface{}). This is necessary for handling
// generic types like object({}) which decode into map[string]any.

package hcl_adapter

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// ctyToNative recursively converts a cty.Value to its most natural Go counterpart.
// It is the engine for decoding generic objects and 'any' types.
func ctyToNative(v cty.Value) (any, error) {
	// A nil or unknown value becomes a nil interface{}.
	if v.IsNull() || !v.IsKnown() {
		return nil, nil
	}

	ty := v.Type()

	switch {
	case ty == cty.String:
		return v.AsString(), nil

	case ty == cty.Number:
		// For a generic 'any' target, float64 is the most sensible and common
		// representation for a number.
		var f float64
		if err := gocty.FromCtyValue(v, &f); err != nil {
			return nil, fmt.Errorf("could not convert cty.Number to float64: %w", err)
		}
		return f, nil

	case ty == cty.Bool:
		var b bool
		if err := gocty.FromCtyValue(v, &b); err != nil {
			// This path should be theoretically unreachable if our type switch is correct,
			// but it's the safe way to implement this.
			return nil, fmt.Errorf("internal error: failed to convert cty.Bool to bool: %w", err)
		}
		return b, nil

	case ty.IsListType() || ty.IsTupleType():
		slice := make([]any, 0)
		it := v.ElementIterator()
		for it.Next() {
			_, val := it.Element()
			nativeVal, err := ctyToNative(val)
			if err != nil {
				return nil, err // Propagate errors from recursive calls
			}
			slice = append(slice, nativeVal)
		}
		return slice, nil

	case ty.IsObjectType() || ty.IsMapType():
		goMap := make(map[string]any)
		it := v.ElementIterator()
		for it.Next() {
			key, val := it.Element()
			keyStr := key.AsString()
			nativeVal, err := ctyToNative(val)
			if err != nil {
				// Add context to the error
				return nil, fmt.Errorf("in attribute '%s': %w", keyStr, err)
			}
			goMap[keyStr] = nativeVal
		}
		return goMap, nil

	default:
		return nil, fmt.Errorf("unsupported cty type for 'any' conversion: %s", ty.FriendlyName())
	}
}

// ToCtyValue converts a native Go value into its corresponding cty.Value.
func ToCtyValue(v any) (cty.Value, error) {
	if v == nil {
		return cty.NilVal, nil
	}
	ty, err := gocty.ImpliedType(v)
	if err != nil {
		return cty.NilVal, fmt.Errorf("unable to infer cty.Type: %w", err)
	}
	return gocty.ToCtyValue(v, ty)
}
