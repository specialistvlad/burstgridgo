package executor

import (
	"context"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/specialistvlad/burstgridgo/internal/config"
	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/zclconf/go-cty/cty"
)

// instanceOutput is a helper struct to hold an instance's output along with its
// index and instancing mode.
type instanceOutput struct {
	index int
	value cty.Value
	mode  config.InstancingMode
}

// buildEvalContext creates the HCL evaluation context for a node.
func (e *Executor) buildEvalContext(ctx context.Context, currentNode *node.Node) *hcl.EvalContext {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Building HCL evaluation context.", "node", currentNode.ID())
	vars := make(map[string]cty.Value)

	// map[runner_type] -> map[instance_name] -> []instanceOutput
	stepOutputsByRunner := make(map[string]map[string][]instanceOutput)

	// First pass: collect outputs from ALL successfully completed steps in the graph.
	logger.Debug("Starting to collect outputs from all completed graph nodes.")
	for _, graphNode := range e.Graph.Nodes {
		if graphNode.Type != node.StepNode || graphNode.GetState() != node.Done || graphNode.Output == nil {
			continue
		}

		runnerType := graphNode.StepConfig.RunnerType
		instanceName := graphNode.StepConfig.Name

		if _, ok := stepOutputsByRunner[runnerType]; !ok {
			stepOutputsByRunner[runnerType] = make(map[string][]instanceOutput)
		}

		// ** THE FIX **: Handle placeholder and non-placeholder nodes differently.
		if graphNode.IsPlaceholder {
			// For a placeholder, its Output is the raw, un-wrapped list of instance outputs.
			// We store this list directly, using a sentinel index to identify it in the second pass.
			stepOutputsByRunner[runnerType][instanceName] = []instanceOutput{
				{
					index: -1, // Sentinel for a pre-aggregated placeholder list.
					value: graphNode.Output.(cty.Value),
					mode:  config.ModeInstanced,
				},
			}
		} else {
			// This is the original logic for non-placeholder (static count or single) nodes.
			lastSegment := graphNode.Address().Path[len(graphNode.Address().Path)-1]
			if !lastSegment.HasIndex() {
				logger.Warn("Could not parse instance index from graph node ID, skipping for HCL context.", "graph_node_id", graphNode.ID())
				continue
			}
			index := lastSegment.Index

			// For static instances, we wrap each individual output in an object.
			outputWithWrapper := cty.ObjectVal(map[string]cty.Value{
				"output": graphNode.Output.(cty.Value),
			})
			stepOutputsByRunner[runnerType][instanceName] = append(
				stepOutputsByRunner[runnerType][instanceName],
				instanceOutput{
					index: index,
					value: outputWithWrapper,
					mode:  graphNode.StepConfig.Instancing,
				},
			)
		}
	}
	logger.Debug("Finished collecting completed step outputs.")

	// Second pass: build the final `step` object for the HCL context.
	logger.Debug("Building final 'step' variable for HCL context.")
	finalStepOutputs := make(map[string]cty.Value)
	for runnerType, instancesMap := range stepOutputsByRunner {
		runnerMap := make(map[string]cty.Value)
		for instanceName, outputs := range instancesMap {
			if len(outputs) == 0 {
				continue
			}

			// ** THE FIX **: Check for the placeholder's sentinel value.
			if len(outputs) == 1 && outputs[0].index == -1 {
				// This is a dynamic-count placeholder. The `value` is the raw list (e.g., [val1, val2]).
				rawList := outputs[0].value

				// To support the splat operator, we must transform this raw list into a list of objects
				// of the shape `[ {output: val1}, {output: val2} ]`.
				wrappedOutputs := make([]cty.Value, 0, rawList.LengthInt())
				if rawList.IsKnown() && !rawList.IsNull() && rawList.LengthInt() > 0 {
					for it := rawList.ElementIterator(); it.Next(); {
						_, val := it.Element()
						wrappedObj := cty.ObjectVal(map[string]cty.Value{
							"output": val,
						})
						wrappedOutputs = append(wrappedOutputs, wrappedObj)
					}
				}

				if len(wrappedOutputs) > 0 {
					runnerMap[instanceName] = cty.ListVal(wrappedOutputs)
				} else {
					runnerMap[instanceName] = cty.EmptyTupleVal
				}
				continue
			}

			// This is the original logic for static-count and single steps.
			sort.Slice(outputs, func(i, j int) bool {
				return outputs[i].index < outputs[j].index
			})

			if outputs[0].mode == config.ModeInstanced {
				// For static-count steps, create a sparse list of the pre-wrapped objects.
				maxIndex := outputs[len(outputs)-1].index
				outputList := make([]cty.Value, maxIndex+1)
				outputType := outputs[0].value.Type()

				for i := 0; i <= maxIndex; i++ {
					outputList[i] = cty.NullVal(outputType)
				}
				for _, out := range outputs {
					outputList[out.index] = out.value
				}
				runnerMap[instanceName] = cty.ListVal(outputList)
			} else {
				// For singular steps, expose the single value directly.
				runnerMap[instanceName] = outputs[0].value
			}
		}
		if len(runnerMap) > 0 {
			finalStepOutputs[runnerType] = cty.ObjectVal(runnerMap)
		}
	}
	vars["step"] = cty.ObjectVal(finalStepOutputs)

	// Inject `count.index` for the *current* node that is executing.
	if !currentNode.IsPlaceholder {
		lastSegment := currentNode.Address().Path[len(currentNode.Address().Path)-1]
		if lastSegment.HasIndex() {
			vars["count"] = cty.ObjectVal(map[string]cty.Value{
				"index": cty.NumberIntVal(int64(lastSegment.Index)),
			})
		}
	}

	logger.Debug("Finished building HCL evaluation context.", "node", currentNode.ID())
	return &hcl.EvalContext{Variables: vars}
}
