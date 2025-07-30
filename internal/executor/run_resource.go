package executor

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/dag"
)

// runResourceNode handles the creation of a stateful resource.
func (e *Executor) runResourceNode(ctx context.Context, node *dag.Node) error {
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
