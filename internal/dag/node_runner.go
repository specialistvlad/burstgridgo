package dag

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// executeResourceNode handles the creation of a stateful resource.
func (e *Executor) executeResourceNode(ctx context.Context, node *Node) error {
	logger := slog.With("resource", node.ID)
	logger.Info("▶️ Creating resource")

	assetType := node.ResourceConfig.AssetType

	// 1. Find asset definition and its lifecycle handlers.
	assetDef, ok := engine.AssetDefinitionRegistry[assetType]
	if !ok {
		return fmt.Errorf("unknown asset type '%s' referenced by resource '%s'", assetType, node.ID)
	}
	if assetDef.Lifecycle == nil {
		return fmt.Errorf("asset type '%s' has no 'lifecycle' block defined in its manifest", assetType)
	}
	createHandlerName := assetDef.Lifecycle.Create
	destroyHandlerName := assetDef.Lifecycle.Destroy
	if createHandlerName == "" || destroyHandlerName == "" {
		return fmt.Errorf("asset type '%s' is missing 'create' or 'destroy' handler name in its lifecycle", assetType)
	}

	// 2. Find the Go functions for these handlers, checking overrides first.
	assetHandler, ok := e.assetHandlerOverrides[createHandlerName]
	if !ok {
		assetHandler, ok = engine.AssetHandlerRegistry[createHandlerName]
	}
	if !ok || assetHandler.CreateFn == nil {
		return fmt.Errorf("create handler '%s' for asset type '%s' is not registered or is nil", createHandlerName, assetType)
	}
	destroyFn, ok := e.assetHandlerOverrides[destroyHandlerName]
	if !ok {
		destroyFn, ok = engine.AssetHandlerRegistry[destroyHandlerName]
	}
	if !ok || destroyFn.DestroyFn == nil {
		return fmt.Errorf("destroy handler '%s' for asset type '%s' is not registered or is nil", destroyHandlerName, assetType)
	}

	// --- Validation Passed: Proceed with Execution ---

	// Decode arguments.
	inputStruct := assetHandler.NewInput()
	evalCtx := e.buildEvalContext(node)
	if node.ResourceConfig.Arguments != nil {
		if diags := gohcl.DecodeBody(node.ResourceConfig.Arguments.Body, evalCtx, inputStruct); diags.HasErrors() {
			return diags
		}
	}

	// Call Create handler via reflection.
	handlerFunc := reflect.ValueOf(assetHandler.CreateFn)
	results := handlerFunc.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(inputStruct)})
	resourceObj := results[0].Interface()
	errResult := results[1].Interface()
	if errResult != nil {
		return errResult.(error)
	}

	// Store instance and register destroy function for guaranteed cleanup.
	node.Output = resourceObj
	e.resourceInstances.Store(node.ID, resourceObj)
	e.pushCleanup(func() {
		logger.Info("🔥 Destroying resource via deferred cleanup")
		reflect.ValueOf(destroyFn.DestroyFn).Call([]reflect.Value{reflect.ValueOf(resourceObj)})
	})

	logger.Info("✅ Resource created")
	return nil
}

// executeStepNode handles the execution of a stateless step.
func (e *Executor) executeStepNode(ctx context.Context, node *Node) error {
	logger := slog.With("step", node.ID)
	logger.Info("▶️ Starting step")

	// Find runner definition and handler
	runnerDef, ok := engine.DefinitionRegistry[node.StepConfig.RunnerType]
	if !ok {
		return fmt.Errorf("unknown runner type '%s'", node.StepConfig.RunnerType)
	}
	handlerName := runnerDef.Lifecycle.OnRun
	registeredHandler, ok := e.handlerOverrides[handlerName] // Check overrides first
	if !ok {
		registeredHandler, ok = engine.HandlerRegistry[handlerName] // Fallback to global registry
	}
	if !ok {
		return fmt.Errorf("handler '%s' for runner '%s' not registered or provided as override", handlerName, runnerDef.Type)
	}

	// Decode 'arguments' block
	inputStruct := registeredHandler.NewInput()
	evalCtx := e.buildEvalContext(node)
	if inputStruct != nil && node.StepConfig.Arguments != nil {
		if diags := gohcl.DecodeBody(node.StepConfig.Arguments.Body, evalCtx, inputStruct); diags.HasErrors() {
			return diags
		}
		if err := applyInputDefaults(inputStruct, runnerDef, node.StepConfig.Arguments.Body); err != nil {
			return fmt.Errorf("error applying default values for step %s: %w", node.ID, err)
		}
	}

	// Build 'deps' struct from the 'uses' block
	depsStruct, err := e.buildDepsStruct(node, runnerDef, registeredHandler, evalCtx)
	if err != nil {
		return err
	}

	// Call handler via reflection
	handlerFunc := reflect.ValueOf(registeredHandler.Fn)
	callArgs := []reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(depsStruct),
	}
	if inputStruct == nil {
		inputType := handlerFunc.Type().In(2)
		callArgs = append(callArgs, reflect.Zero(inputType))
	} else {
		callArgs = append(callArgs, reflect.ValueOf(inputStruct))
	}

	results := handlerFunc.Call(callArgs)
	outputVal := results[0].Interface()
	errResult := results[1].Interface()
	if errResult != nil {
		return errResult.(error)
	}

	// Store output
	if outputVal == nil {
		node.Output = cty.NilVal
	} else if ctyOutput, ok := outputVal.(cty.Value); ok {
		node.Output = ctyOutput
	} else {
		return fmt.Errorf("handler for step %s returned a non-cty.Value type: %T", node.ID, outputVal)
	}

	logger.Info("✅ Finished step")
	return nil
}

// applyInputDefaults uses reflection to apply default values from a runner's
// manifest to an input struct for any fields the user did not provide.
func applyInputDefaults(inputStruct any, runnerDef *engine.RunnerDefinition, userBody hcl.Body) error {
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
					if err := gocty.FromCtyValue(*inputDef.Default, fieldVal.Addr().Interface()); err != nil {
						return fmt.Errorf("failed to apply default for input '%s': %w", inputDef.Name, err)
					}
				}
				break
			}
		}
	}
	return nil
}
