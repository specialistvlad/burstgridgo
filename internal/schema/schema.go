package schema

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// --- Primary Grid Structures ---

// StepArgs represents the content of the 'arguments' block within a step.
type StepArgs struct {
	Body hcl.Body `hcl:",remain"`
}

// UsesBlock represents the content of the 'uses' block within a step.
type UsesBlock struct {
	Body hcl.Body `hcl:",remain"`
}

// Step represents a `step` block from a user's grid file. It is a runnable
// instance of a defined runner.
type Step struct {
	RunnerType string     `hcl:"runner_type,label"`
	Name       string     `hcl:"instance_name,label"`
	Arguments  *StepArgs  `hcl:"arguments,block"`
	Uses       *UsesBlock `hcl:"uses,block"`
	DependsOn  []string   `hcl:"depends_on,optional"`
}

// Resource represents a `resource` block from a user's grid file. It is a
// managed, stateful instance of a defined asset.
type Resource struct {
	AssetType string    `hcl:"asset_type,label"`
	Name      string    `hcl:"instance_name,label"`
	Arguments *StepArgs `hcl:"arguments,block"`
	DependsOn []string  `hcl:"depends_on,optional"`
}

// GridConfig represents the top-level structure of a user's grid file,
// containing all defined steps and resources.
type GridConfig struct {
	Steps     []*Step     `hcl:"step,block"`
	Resources []*Resource `hcl:"resource,block"`
	Body      hcl.Body    `hcl:",remain"`
}

// --- Module Manifest Schemas ---

// Lifecycle defines the mapping from a runner's lifecycle event to a
// registered Go handler function.
type Lifecycle struct {
	OnRun string `hcl:"on_run,optional"`
}

// AssetLifecycle defines the mapping from a resource's lifecycle events
// (create, destroy) to registered Go handler functions.
type AssetLifecycle struct {
	Create  string `hcl:"create"`
	Destroy string `hcl:"destroy"`
}

// InputDefinition defines a single input variable for a runner or asset.
type InputDefinition struct {
	Name        string         `hcl:"name,label"`
	Type        hcl.Expression `hcl:"type"`
	Description string         `hcl:"description,optional"`
	Default     *cty.Value     `hcl:"default,optional"`
}

// OutputDefinition defines a single output value produced by a runner or asset.
type OutputDefinition struct {
	Name        string         `hcl:"name,label"`
	Type        hcl.Expression `hcl:"type"`
	Description string         `hcl:"description,optional"`
}

// UsesDefinition defines an asset dependency required by a runner.
type UsesDefinition struct {
	LocalName string `hcl:"local_name,label"`
	AssetType string `hcl:"asset_type"`
}

// RunnerDefinition represents the HCL manifest for a runnable `runner` type.
type RunnerDefinition struct {
	Type        string              `hcl:"type,label"`
	Description string              `hcl:"description,optional"`
	Lifecycle   *Lifecycle          `hcl:"lifecycle,block"`
	Inputs      []*InputDefinition  `hcl:"input,block"`
	Outputs     []*OutputDefinition `hcl:"output,block"`
	Uses        []*UsesDefinition   `hcl:"uses,block"`
}

// AssetDefinition represents the HCL manifest for a stateful `asset` type.
type AssetDefinition struct {
	Type        string              `hcl:"type,label"`
	Description string              `hcl:"description,optional"`
	Lifecycle   *AssetLifecycle     `hcl:"lifecycle,block"`
	Inputs      []*InputDefinition  `hcl:"input,block"`
	Outputs     []*OutputDefinition `hcl:"output,block"`
}

// DefinitionConfig represents the top-level structure of a module manifest file.
type DefinitionConfig struct {
	Runner *RunnerDefinition `hcl:"runner,block"`
	Asset  *AssetDefinition  `hcl:"asset,block"`
	Body   hcl.Body          `hcl:",remain"`
}
