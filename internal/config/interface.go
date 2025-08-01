package config

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// Loader is the interface for a format-specific configuration loader.
type Loader interface {
	// Load reads configuration from a given path, translates it into the
	// format-agnostic model, and returns a matching Converter.
	Load(ctx context.Context, paths ...string) (*Model, Converter, error)
}

// Converter is the interface for a format-specific data binding and type
// conversion implementation. It acts as the bridge between the raw configuration
// and the Go types used by modules.
type Converter interface {
	// DecodeBody decodes a raw configuration body (e.g., an 'arguments'
	// block) into a target Go struct, applying defaults and validations.
	DecodeBody(
		ctx context.Context,
		inputStruct any,
		args map[string]hcl.Expression,
		defs map[string]*InputDefinition,
		evalCtx *hcl.EvalContext,
	) error

	// ToCtyValue converts a native Go value (like a map[string]any from a pure
	// Go module) into its equivalent cty.Value for the engine's internal use.
	ToCtyValue(v any) (cty.Value, error)
}
