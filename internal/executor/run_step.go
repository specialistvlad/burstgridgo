package executor

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/dag"
	"github.com/vk/burstgridgo/internal/schema"
	"github.com/zclconf/go-cty/cty"
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

	logger.Debug("Decoding step arguments.")
	inputStruct := registeredHandler.NewInput()
	evalCtx := e.buildEvalContext(ctx, node)
	if inputStruct != nil && node.StepConfig.Arguments != nil {
		// First, validate that all provided arguments are declared in the manifest.
		if err := e.validateArgumentsAgainstManifest(ctx, runnerDef, node.StepConfig.Arguments.Body); err != nil {
			return fmt.Errorf("invalid arguments for step %s: %w", node.ID, err)
		}

		if diags := gohcl.DecodeBody(node.StepConfig.Arguments.Body, evalCtx, inputStruct); diags.HasErrors() {
			return diags
		}
		if err := applyInputDefaults(ctx, inputStruct, runnerDef, node.StepConfig.Arguments.Body); err != nil {
			return fmt.Errorf("applying defaults for step %s: %w", node.ID, err)
		}
	}
	logger.Debug("Step Input:", "data", formatValueForLogs(inputStruct))

	logger.Debug("Building step dependencies.")
	depsStruct, err := e.buildDepsStruct(ctx, node, registeredHandler, evalCtx)
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
	outputVal, errResult := results[0].Interface(), results[1].Interface()
	if errResult != nil {
		return errResult.(error)
	}

	if outputVal == nil {
		node.Output = cty.NilVal
	} else if ctyOutput, ok := outputVal.(cty.Value); ok {
		node.Output = ctyOutput
	} else {
		return fmt.Errorf("handler for step %s returned non-cty.Value type: %T", node.ID, outputVal)
	}

	logger.Debug("Step Output:", "data", formatValueForLogs(node.Output))

	logger.Info("✅ Finished step")
	return nil
}

// validateArgumentsAgainstManifest checks that all arguments in the user's HCL
// are actually defined in the runner's manifest.
func (e *Executor) validateArgumentsAgainstManifest(ctx context.Context, runnerDef *schema.RunnerDefinition, userBody hcl.Body) error {
	if runnerDef == nil || userBody == nil {
		return nil
	}

	declaredInputs := make(map[string]struct{})
	for _, inputDef := range runnerDef.Inputs {
		declaredInputs[inputDef.Name] = struct{}{}
	}

	userAttrs, diags := userBody.JustAttributes()
	if diags.HasErrors() {
		return diags
	}

	for attrName := range userAttrs {
		if _, ok := declaredInputs[attrName]; !ok {
			return fmt.Errorf("undeclared argument %q", attrName)
		}
	}
	return nil
}
