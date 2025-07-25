package engine

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// --- Grid File Structs (User's HCL) ---

// StepArgs represents the content of the 'arguments' block within a step.
type StepArgs struct {
	Body hcl.Body `hcl:",remain"`
}

// UsesBlock represents the content of the 'uses' block within a step.
type UsesBlock struct {
	Body hcl.Body `hcl:",remain"`
}

// Step represents a step block from a user's grid file.
type Step struct {
	RunnerType string     `hcl:"runner_type,label"`
	Name       string     `hcl:"instance_name,label"`
	Arguments  *StepArgs  `hcl:"arguments,block"`
	Uses       *UsesBlock `hcl:"uses,block"` // Corrected: removed ",optional"
	DependsOn  []string   `hcl:"depends_on,optional"`
}

// Resource represents a resource block from a user's grid file.
type Resource struct {
	AssetType string    `hcl:"asset_type,label"`
	Name      string    `hcl:"instance_name,label"`
	Arguments *StepArgs `hcl:"arguments,block"`
	DependsOn []string  `hcl:"depends_on,optional"`
}

// GridConfig represents the top-level structure of a user's grid file.
type GridConfig struct {
	Steps     []*Step     `hcl:"step,block"`
	Resources []*Resource `hcl:"resource,block"`
	Body      hcl.Body    `hcl:",remain"`
}

// --- Module Manifest Structs (Module's HCL) ---

// Lifecycle defines the Go handlers for a runner's lifecycle events.
type Lifecycle struct {
	OnRun string `hcl:"on_run,optional"`
}

// AssetLifecycle defines Go handlers for a resource's lifecycle.
type AssetLifecycle struct {
	Create  string `hcl:"create"`
	Destroy string `hcl:"destroy"`
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

// UsesDefinition defines an asset dependency for a runner.
type UsesDefinition struct {
	LocalName string `hcl:"local_name,label"` // The key in the 'uses' block, e.g., "DB"
	AssetType string `hcl:"asset_type"`       // The required asset type, e.g., "database"
}

// RunnerDefinition represents the HCL manifest for a runner type.
type RunnerDefinition struct {
	Type        string              `hcl:"type,label"`
	Description string              `hcl:"description,optional"`
	Lifecycle   *Lifecycle          `hcl:"lifecycle,block"`
	Inputs      []*InputDefinition  `hcl:"input,block"`
	Outputs     []*OutputDefinition `hcl:"output,block"`
	Uses        []*UsesDefinition   `hcl:"uses,block"`
}

// AssetDefinition represents the HCL manifest for an asset type.
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
