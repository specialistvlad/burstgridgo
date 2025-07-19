package engine

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// Runner defines the interface that all modules must implement to be executable.
type Runner interface {
	// Run now accepts an EvalContext to resolve inputs and returns a cty.Value for its output.
	Run(m Module, ctx *hcl.EvalContext) (cty.Value, error)
}

// Registry is a map that holds all the registered module runners,
// keyed by the runner name specified in the HCL config.
var Registry = make(map[string]Runner)
