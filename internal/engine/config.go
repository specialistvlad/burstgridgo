package engine

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// Step represents a step block from a user's grid file.
// It corresponds to the HCL: step "runner_type" "instance_name" { ... }
type Step struct {
	RunnerType string   `hcl:"runner_type,label"`
	Name       string   `hcl:"instance_name,label"`
	Arguments  hcl.Body `hcl:"arguments,block"`
	DependsOn  []string `hcl:"depends_on,optional"`
}

// GridConfig represents the top-level structure of a user's grid file.
type GridConfig struct {
	Steps []*Step `hcl:"step,block"`
}

// --- Runner Definition Structs ---

// Lifecycle defines the Go handlers for a runner's lifecycle events.
type Lifecycle struct {
	OnStart string `hcl:"on_start,optional"`
	OnRun   string `hcl:"on_run,optional"`
	OnEnd   string `hcl:"on_end,optional"`
}

// InputDefinition defines a single input variable for a runner.
type InputDefinition struct {
	Name        string         `hcl:"name,label"`
	Type        hcl.Expression `hcl:"type"`
	Description string         `hcl:"description,optional"`
	Optional    bool           `hcl:"optional,optional"`
	Default     *cty.Value     `hcl:"default,optional"`
}

// OutputDefinition defines a single output value for a runner.
type OutputDefinition struct {
	Name        string         `hcl:"name,label"`
	Type        hcl.Expression `hcl:"type"`
	Description string         `hcl:"description,optional"`
}

// RunnerDefinition represents the HCL manifest for a runner type.
// It corresponds to the HCL: runner "type" { ... }
type RunnerDefinition struct {
	Type        string              `hcl:"type,label"`
	Description string              `hcl:"description,optional"`
	Version     string              `hcl:"version,optional"`
	Lifecycle   *Lifecycle          `hcl:"lifecycle,block"`
	Inputs      []*InputDefinition  `hcl:"input,block"`
	Outputs     []*OutputDefinition `hcl:"output,block"`
}

// DefinitionConfig represents the top-level structure of a runner manifest file.
type DefinitionConfig struct {
	Runner *RunnerDefinition `hcl:"runner,block"`
}
