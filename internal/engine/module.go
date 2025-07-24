package engine

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// Runner defines the interface that all modules must implement to be executable.
type Runner interface {
	// Run now accepts a context.Context for cancellation and deadlines.
	Run(ctx context.Context, m Module, evalCtx *hcl.EvalContext) (cty.Value, error)
}

// Registry is a map that holds all the registered module runners,
// keyed by the runner name specified in the HCL config.
var Registry = make(map[string]Runner)
