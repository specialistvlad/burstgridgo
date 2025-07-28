package dag

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/schema"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// formatValueForLogs converts a value to its loggable representation.
// For cty.Value, it's converted to a Go interface. Other types are passed through.
func formatValueForLogs(v any) any {
	if ctyVal, ok := v.(cty.Value); ok {
		converted, err := ctyValueToInterface(ctyVal)
		if err != nil {
			return fmt.Sprintf("[unloggable cty.Value: %v]", err)
		}
		return converted
	}
	return v
}

// executeResourceNode handles the creation of a stateful resource.
func (e *Executor) executeResourceNode(ctx context.Context, node *Node) error {
	logger := ctxlog.FromContext(ctx).With("resource", node.ID)
	logger.Info("▶️ Creating resource")
	logger.Debug("Executing resource node.")

	assetType := node.ResourceConfig.AssetType
	assetDef, ok := e.registry.AssetDefinitionRegistry[assetType]
	if !ok {
		return fmt.Errorf("unknown asset type '%s'", assetType)
	}
	createHandlerName := assetDef.Lifecycle.Create
	destroyHandlerName := assetDef.Lifecycle.Destroy

	assetHandler, ok := e.registry.AssetHandlerRegistry[createHandlerName]
	if !ok || assetHandler.CreateFn == nil {
		return fmt.Errorf("create handler '%s' not registered", createHandlerName)
	}

	destroyFn, ok := e.registry.AssetHandlerRegistry[destroyHandlerName]
	if !ok || destroyFn.DestroyFn == nil {
		return fmt.Errorf("destroy handler '%s' not registered", destroyHandlerName)
	}

	logger.Debug("Decoding resource arguments.")
	inputStruct := assetHandler.NewInput()
	evalCtx := e.buildEvalContext(ctx, node)
	if node.ResourceConfig.Arguments != nil {
		if diags := gohcl.DecodeBody(node.ResourceConfig.Arguments.Body, evalCtx, inputStruct); diags.HasErrors() {
			return diags
		}
	}

	logger.Debug("Calling resource create handler.", "handler", createHandlerName)
	handlerFunc := reflect.ValueOf(assetHandler.CreateFn)
	results := handlerFunc.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(inputStruct)})
	resourceObj, errResult := results[0].Interface(), results[1].Interface()
	if errResult != nil {
		return errResult.(error)
	}

	node.Output = resourceObj
	e.resourceInstances.Store(node.ID, resourceObj)
	e.pushCleanup(node, func() {
		logger.Info("🔥 Destroying resource")
		reflect.ValueOf(destroyFn.DestroyFn).Call([]reflect.Value{reflect.ValueOf(resourceObj)})
		e.resourceInstances.Delete(node.ID)
	})

	logger.Info("✅ Resource created")
	return nil
}

// executeStepNode handles the execution of a stateless step.
func (e *Executor) executeStepNode(ctx context.Context, node *Node) error {
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

// applyInputDefaults applies default values from a runner's manifest.
func applyInputDefaults(ctx context.Context, inputStruct any, runnerDef *schema.RunnerDefinition, userBody hcl.Body) error {
	logger := ctxlog.FromContext(ctx)
	if inputStruct == nil || runnerDef == nil || userBody == nil {
		return nil
	}

	userAttrs, _ := userBody.JustAttributes()
	userProvidedNames := make(map[string]struct{})
	for name := range userAttrs {
		userProvidedNames[name] = struct{}{}
	}

	structVal := reflect.ValueOf(inputStruct).Elem()
	structType := structVal.Type()

	for _, inputDef := range runnerDef.Inputs {
		if _, ok := userProvidedNames[inputDef.Name]; ok || inputDef.Default == nil {
			continue
		}
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			tagName := strings.Split(field.Tag.Get("hcl"), ",")[0]
			if tagName == inputDef.Name {
				fieldVal := structVal.Field(i)
				if fieldVal.CanSet() {
					logger.Debug("Applying default value.", "field", tagName, "value", *inputDef.Default)
					if err := gocty.FromCtyValue(*inputDef.Default, fieldVal.Addr().Interface()); err != nil {
						return fmt.Errorf("failed to apply default for '%s': %w", inputDef.Name, err)
					}
				}
				break
			}
		}
	}
	return nil
}

// ctyValueToInterface converts a cty.Value to a Go interface{}.
func ctyValueToInterface(val cty.Value) (any, error) {
	if !val.IsKnown() || val.IsNull() {
		return nil, nil
	}
	if val.Type().IsPrimitiveType() {
		switch val.Type() {
		case cty.String:
			return val.AsString(), nil
		case cty.Number:
			f, _ := val.AsBigFloat().Float64()
			return f, nil
		case cty.Bool:
			return val.True(), nil
		default:
			return nil, fmt.Errorf("unsupported primitive type: %s", val.Type().FriendlyName())
		}
	}
	if val.Type().IsObjectType() || val.Type().IsMapType() {
		out := make(map[string]any)
		for it := val.ElementIterator(); it.Next(); {
			k, v := it.Element()
			valInterface, err := ctyValueToInterface(v)
			if err != nil {
				return nil, err
			}
			out[k.AsString()] = valInterface
		}
		return out, nil
	}
	if val.Type().IsTupleType() || val.Type().IsListType() {
		var out []any
		for it := val.ElementIterator(); it.Next(); {
			_, v := it.Element()
			valInterface, err := ctyValueToInterface(v)
			if err != nil {
				return nil, err
			}
			out = append(out, valInterface)
		}
		return out, nil
	}
	return nil, fmt.Errorf("unsupported cty.Type for conversion: %s", val.Type().FriendlyName())
}
