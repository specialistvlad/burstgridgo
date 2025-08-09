package hcl_adapter

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// Converter is the HCL-specific implementation of the config.Converter interface.
type Converter struct{}

// NewConverter creates a new HCL converter.
func NewConverter() *Converter {
	return &Converter{}
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
