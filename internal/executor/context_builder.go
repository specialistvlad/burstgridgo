package executor

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/dag"
	"github.com/zclconf/go-cty/cty"
)

// buildEvalContext creates the HCL evaluation context for a node.
func (e *Executor) buildEvalContext(ctx context.Context, node *dag.Node) *hcl.EvalContext {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Building HCL evaluation context.", "node", node.ID)
	vars := make(map[string]cty.Value)

	stepOutputsByRunner := make(map[string]map[string]cty.Value)

	for _, depNode := range node.Deps {
		if depNode.Type == dag.StepNode {
			if depNode.GetState() != dag.Done || depNode.Output == nil {
				continue
			}
			runnerType := depNode.StepConfig.RunnerType
			instanceName := depNode.Name
			if _, ok := stepOutputsByRunner[runnerType]; !ok {
				stepOutputsByRunner[runnerType] = make(map[string]cty.Value)
			}
			stepOutputsByRunner[runnerType][instanceName] = cty.ObjectVal(map[string]cty.Value{
				"output": depNode.Output.(cty.Value),
			})
		}
	}

	finalStepOutputs := make(map[string]cty.Value)
	for runnerType, instancesMap := range stepOutputsByRunner {
		finalStepOutputs[runnerType] = cty.ObjectVal(instancesMap)
	}

	vars["step"] = cty.ObjectVal(finalStepOutputs)
	logger.Debug("Finished building HCL evaluation context.", "node", node.ID, "vars_count", len(vars))
	return &hcl.EvalContext{Variables: vars}
}
