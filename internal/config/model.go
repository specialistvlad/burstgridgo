package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// Model is the unified, format-agnostic representation of the entire
// application configuration, including all modules and the execution grid.
type Model struct {
	Runners map[string]*RunnerDefinition
	Assets  map[string]*AssetDefinition
	Grid    *Grid
	// TODO: Add support for variables, etc.
}

// Grid represents the user's execution graph definition.
type Grid struct {
	Steps     []*Step
	Resources []*Resource
}

// Step is the format-agnostic representation of a `step` block.
type Step struct {
	RunnerType string
	Name       string
	Arguments  map[string]hcl.Expression
	Uses       map[string]hcl.Expression
	DependsOn  []string
}

// Resource is the format-agnostic representation of a `resource` block.
type Resource struct {
	AssetType string
	Name      string
	Arguments map[string]hcl.Expression
	DependsOn []string
}

// --- Module Manifest Models ---

// RunnerDefinition is the format-agnostic representation of a runner's manifest.
type RunnerDefinition struct {
	Type        string
	Description string
	Lifecycle   *Lifecycle
	Inputs      map[string]*InputDefinition
	Outputs     map[string]*OutputDefinition
	Uses        map[string]*UsesDefinition
}

// AssetDefinition is the format-agnostic representation of an asset's manifest.
type AssetDefinition struct {
	Type        string
	Description string
	Lifecycle   *AssetLifecycle
	Inputs      map[string]*InputDefinition
	Outputs     map[string]*OutputDefinition
}

// Lifecycle maps a runner's events to Go handler names.
type Lifecycle struct {
	OnRun string
}

// AssetLifecycle maps an asset's events to Go handler names.
type AssetLifecycle struct {
	Create  string
	Destroy string
}

// InputDefinition defines a single input argument for a runner or asset.
type InputDefinition struct {
	Name        string
	Type        cty.Type // Will be populated in a future step (ADR-009)
	Description string
	Default     *cty.Value
	Optional    bool
}

// OutputDefinition defines a single output value from a runner.
type OutputDefinition struct {
	Name        string
	Type        cty.Type // Will be populated in a future step (ADR-009)
	Description string
}

// UsesDefinition defines an asset dependency for a runner.
type UsesDefinition struct {
	LocalName string
	AssetType string
}
