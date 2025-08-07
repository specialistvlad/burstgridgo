package executor

import (
	"context"
	"regexp"
	"sort"
	"strconv"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/dag"
	"github.com/zclconf/go-cty/cty"
)

// instanceOutput is a helper struct to hold an instance's output along with its
// index, so we can sort them before adding to the HCL context.
type instanceOutput struct {
	index int
	value cty.Value
}

var (
	// nodeIndexRegex is used to efficiently parse the instance index from a node ID.
	nodeIndexRegex = regexp.MustCompile(`\[(\d+)\]$`)
)

// buildEvalContext creates the HCL evaluation context for a node.
func (e *Executor) buildEvalContext(ctx context.Context, node *dag.Node) *hcl.EvalContext {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Building HCL evaluation context.", "node", node.ID)
	vars := make(map[string]cty.Value)

	// map[runner_type] -> map[instance_name] -> []instanceOutput
	stepOutputsByRunner := make(map[string]map[string][]instanceOutput)

	// First pass: collect outputs from ALL successfully completed steps in the graph.
	// This provides a consistent, global view of the state for the HCL engine.
	logger.Debug("Starting to collect outputs from all completed graph nodes.")
	for _, graphNode := range e.Graph.Nodes {
		// Only consider step nodes that have finished successfully and have an output.
		if graphNode.Type != dag.StepNode || graphNode.GetState() != dag.Done || graphNode.Output == nil {
			continue
		}

		// Get runner type and instance name directly from the config for robustness.
		runnerType := graphNode.StepConfig.RunnerType
		instanceName := graphNode.StepConfig.Name

		// Parse the index from the node ID.
		matches := nodeIndexRegex.FindStringSubmatch(graphNode.ID)
		if len(matches) != 2 {
			logger.Warn("Could not parse instance index from graph node ID, skipping for HCL context.", "graph_node_id", graphNode.ID)
			continue
		}
		index, _ := strconv.Atoi(matches[1])

		// Initialize nested maps if they don't exist.
		if _, ok := stepOutputsByRunner[runnerType]; !ok {
			stepOutputsByRunner[runnerType] = make(map[string][]instanceOutput)
		}

		// Append the instance's output, wrapped in the 'output' object as requested.
		outputWithWrapper := cty.ObjectVal(map[string]cty.Value{
			"output": graphNode.Output.(cty.Value),
		})
		stepOutputsByRunner[runnerType][instanceName] = append(
			stepOutputsByRunner[runnerType][instanceName],
			instanceOutput{index: index, value: outputWithWrapper},
		)
		logger.Debug("Collected completed step output for instance.",
			"source_node_id", graphNode.ID,
			"runner", runnerType,
			"name", instanceName,
			"index", index,
		)
	}
	logger.Debug("Finished collecting completed step outputs.")

	// Second pass: build the final `step` object for the HCL context from the collected outputs.
	logger.Debug("Building final 'step' variable for HCL context.")
	finalStepOutputs := make(map[string]cty.Value)
	for runnerType, instancesMap := range stepOutputsByRunner {
		runnerMap := make(map[string]cty.Value)
		for instanceName, outputs := range instancesMap {
			if len(outputs) == 0 {
				continue
			}
			sort.Slice(outputs, func(i, j int) bool {
				return outputs[i].index < outputs[j].index
			})

			if len(outputs) > 1 {
				outputList := make([]cty.Value, len(outputs))
				for i, out := range outputs {
					outputList[i] = out.value
				}
				runnerMap[instanceName] = cty.ListVal(outputList)
			} else {
				runnerMap[instanceName] = outputs[0].value
			}

			if val, ok := runnerMap[instanceName]; ok {
				logger.Debug("Prepared HCL context value for step.",
					"runner", runnerType,
					"name", instanceName,
					"is_list", val.Type().IsListType(),
					"num_instances", len(outputs),
				)
			}
		}
		if len(runnerMap) > 0 {
			finalStepOutputs[runnerType] = cty.ObjectVal(runnerMap)
		}
	}
	vars["step"] = cty.ObjectVal(finalStepOutputs)
	logger.Debug("Final 'step' variable constructed.", "value_gostring", vars["step"].GoString())

	// Inject `count.index` for the *current* node that is executing.
	matches := nodeIndexRegex.FindStringSubmatch(node.ID)
	if len(matches) == 2 {
		index, err := strconv.Atoi(matches[1])
		if err == nil {
			logger.Debug("Injecting count.index into context.", "node", node.ID, "index", index)
			vars["count"] = cty.ObjectVal(map[string]cty.Value{
				"index": cty.NumberIntVal(int64(index)),
			})
		}
	}

	logger.Debug("Finished building HCL evaluation context.", "node", node.ID)
	return &hcl.EvalContext{Variables: vars}
}
