package dag

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// buildEvalContext creates the HCL evaluation context for a node, populating it
// with the outputs of its dependencies under the `step` variable.
func (e *Executor) buildEvalContext(node *Node) *hcl.EvalContext {
	vars := make(map[string]cty.Value)

	// Use a standard Go map for building the structure first.
	// The structure is: map[runnerType] -> map[instanceName] -> cty.Value
	stepOutputsByRunner := make(map[string]map[string]cty.Value)

	for _, depNode := range node.Deps {
		if depNode.Type == StepNode {
			// A step can only be a dependency if it's successfully completed.
			if depNode.State.Load() != int32(Done) || depNode.Output == nil {
				continue
			}
			runnerType := depNode.StepConfig.RunnerType
			instanceName := depNode.Name

			// Get or create the inner map for this runner type.
			if _, ok := stepOutputsByRunner[runnerType]; !ok {
				stepOutputsByRunner[runnerType] = make(map[string]cty.Value)
			}

			// Assign the output to the instance name in the inner map.
			stepOutputsByRunner[runnerType][instanceName] = cty.ObjectVal(map[string]cty.Value{
				"output": depNode.Output.(cty.Value),
			})
		}
	}

	// After building the Go map, convert it to the final cty.Value structure.
	finalStepOutputs := make(map[string]cty.Value)
	for runnerType, instancesMap := range stepOutputsByRunner {
		finalStepOutputs[runnerType] = cty.ObjectVal(instancesMap)
	}

	vars["step"] = cty.ObjectVal(finalStepOutputs)
	// Context for resources can be added here if they produce cty.Value outputs.
	return &hcl.EvalContext{Variables: vars}
}
