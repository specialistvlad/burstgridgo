package executor

import (
	"context"
	"fmt"
	"reflect"

	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/dag"
)

// runStepNode handles the execution of a stateless step.
func (e *Executor) runStepNode(ctx context.Context, node *dag.Node) error {
	logger := ctxlog.FromContext(ctx).With("step", node.ID)
	logger.Info("▶️ Starting step")
	logger.Debug("Executing step node.")

	runnerDef, ok := e.registry.DefinitionRegistry[node.StepConfig.RunnerType]
	if !ok {
		return fmt.Errorf("unknown runner type '%s'", node.StepConfig.RunnerType)
	}
	handlerName := runnerDef.Lifecycle.OnRun
	registeredHandler, ok := e.registry.HandlerRegistry[handlerName]
	if !ok {
		return fmt.Errorf("handler '%s' not registered", handlerName)
	}

	// Use the robust decoding logic via the converter interface.
	inputStruct := registeredHandler.NewInput()
	if inputStruct != nil {
		evalCtx := e.buildEvalContext(ctx, node)
		// Pass the context down to the decoder.
		err := e.converter.DecodeBody(ctx, inputStruct, node.StepConfig.Arguments, runnerDef.Inputs, evalCtx)
		if err != nil {
			return fmt.Errorf("failed to decode arguments for step %s: %w", node.ID, err)
		}
	}
	logger.Debug("Step Input:", "data", formatValueForLogs(inputStruct))

	logger.Debug("Building step dependencies.")
	depsStruct, err := e.buildDepsStruct(ctx, node, registeredHandler)
	if err != nil {
		return err
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
		return errResult.(error)
	}

	// Convert the native Go return value to a cty.Value for the engine.
	ctyOutput, err := e.converter.ToCtyValue(nativeOutput)
	if err != nil {
		return fmt.Errorf("failed to convert handler output to cty.Value for step %s: %w", node.ID, err)
	}
	node.Output = ctyOutput

	logger.Debug("Step Output:", "data", node.Output)

	logger.Info("✅ Finished step")
	return nil
}
