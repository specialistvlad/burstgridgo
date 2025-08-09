package executor

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/builder"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/zclconf/go-cty/cty"
)

// runPlaceholderNode handles the runtime expansion and execution of a dynamic step.
func (e *Executor) runPlaceholderNode(ctx context.Context, node *builder.Node) error {
	logger := ctxlog.FromContext(ctx).With("step", node.ID)
	logger.Info("▶️ Expanding dynamic step")

	// 1. Evaluate the count expression
	evalCtx := e.buildEvalContext(ctx, node)
	val, diags := node.StepConfig.Count.Value(evalCtx)
	if diags.HasErrors() {
		return diags
	}
	if val.Type() != cty.Number {
		return fmt.Errorf("count for step %s must be a number, but got %s", node.ID, val.Type().FriendlyName())
	}
	countBf, _ := val.AsBigFloat().Int64()
	count := int(countBf)

	if count < 0 {
		return fmt.Errorf("count for step %s cannot be negative, got %d", node.ID, count)
	}

	logger.Debug("Dynamic count resolved.", "count", count)
	if count == 0 {
		logger.Info("✅ Finished expanding dynamic step (0 instances).")
		// Set output to an empty list for downstream consumers.
		node.Output = cty.EmptyTupleVal
		return nil
	}

	// 2. Loop and execute each instance, collecting outputs.
	outputs := make([]cty.Value, 0, count)
	for i := 0; i < count; i++ {
		instanceID := fmt.Sprintf("%s[%d]", node.ID, i)
		instanceEvalCtx := evalCtx.NewChild()
		instanceEvalCtx.Variables = make(map[string]cty.Value)
		instanceEvalCtx.Variables["count"] = cty.ObjectVal(map[string]cty.Value{
			"index": cty.NumberIntVal(int64(i)),
		})

		// Execute the core logic for this single instance.
		output, err := e.executeStepLogic(ctx, node, instanceEvalCtx, instanceID)
		if err != nil {
			return fmt.Errorf("instance %s of step %s failed: %w", instanceID, node.ID, err)
		}
		outputs = append(outputs, output.(cty.Value))
	}

	// 3. Aggregate outputs and set them on the placeholder node.
	node.Output = cty.ListVal(outputs)

	logger.Info("✅ Finished expanding dynamic step.", "instances_created", count)
	return nil
}

// runStepNode handles the execution of a single, non-placeholder step node.
func (e *Executor) runStepNode(ctx context.Context, node *builder.Node) error {
	evalCtx := e.buildEvalContext(ctx, node)
	output, err := e.executeStepLogic(ctx, node, evalCtx, node.ID)
	if err != nil {
		return err
	}
	node.Output = output
	return nil
}

// executeStepLogic contains the shared logic for running a step's handler.
func (e *Executor) executeStepLogic(ctx context.Context, node *builder.Node, evalCtx *hcl.EvalContext, instanceID string) (any, error) {
	logger := ctxlog.FromContext(ctx).With("step", instanceID)
	logger.Info("▶️ Starting step instance")

	runnerDef, ok := e.registry.DefinitionRegistry[node.StepConfig.RunnerType]
	if !ok {
		return nil, fmt.Errorf("unknown runner type '%s'", node.StepConfig.RunnerType)
	}
	handlerName := runnerDef.Lifecycle.OnRun
	registeredHandler, ok := e.registry.HandlerRegistry[handlerName]
	if !ok {
		return nil, fmt.Errorf("handler '%s' not registered", handlerName)
	}

	inputStruct := registeredHandler.NewInput()
	if inputStruct != nil {
		err := e.converter.DecodeBody(ctx, inputStruct, node.StepConfig.Arguments, runnerDef.Inputs, evalCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to decode arguments for step instance %s: %w", instanceID, err)
		}
	}
	logger.Debug("Step instance input:", "data", formatValueForLogs(inputStruct))

	depsStruct, err := e.buildDepsStruct(ctx, node, registeredHandler)
	if err != nil {
		return nil, err
	}

	logger.Debug("Calling step run handler.", "handler", handlerName)
	handlerFunc := reflect.ValueOf(registeredHandler.Fn)
	callArgs := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(depsStruct)}

	if inputStruct == nil {
		inputType := handlerFunc.Type().In(2)
		callArgs = append(callArgs, reflect.Zero(inputType))
	} else {
		callArgs = append(callArgs, reflect.ValueOf(inputStruct))
	}

	results := handlerFunc.Call(callArgs)
	nativeOutput, errResult := results[0].Interface(), results[1].Interface()
	if errResult != nil {
		return nil, errResult.(error)
	}

	ctyOutput, err := e.converter.ToCtyValue(nativeOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to convert handler output to cty.Value for step instance %s: %w", instanceID, err)
	}

	logger.Info("✅ Finished step instance")
	return ctyOutput, nil
}
