package old_executor

import (
	"context"
	"reflect"

	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/specialistvlad/burstgridgo/internal/node"
)

// pushCleanup adds a function to the LIFO cleanup stack.
func (e *Executor) pushCleanup(node *node.Node, f func()) {
	e.cleanupMutex.Lock()
	defer e.cleanupMutex.Unlock()
	e.cleanupStack = append(e.cleanupStack, func() {
		node.Destroy(f)
	})
}

// executeCleanupStack runs all registered cleanup functions in LIFO order.
func (e *Executor) executeCleanupStack(ctx context.Context) {
	logger := ctxlog.FromContext(ctx)
	e.cleanupMutex.Lock()
	defer e.cleanupMutex.Unlock()
	logger.Info("Executing cleanup stack...")
	for i := len(e.cleanupStack) - 1; i >= 0; i-- {
		e.cleanupStack[i]()
	}
	e.cleanupStack = nil // Clear the stack
}

// destroyResource handles the efficient, runtime destruction of a resource.
func (e *Executor) destroyResource(ctx context.Context, node *node.Node) {
	logger := ctxlog.FromContext(ctx)
	instance, found := e.resourceInstances.Load(node.ID())
	if !found {
		return
	}

	resourceLogger := logger.With("resource", node.ID())
	assetDef := e.registry.AssetDefinitionRegistry[node.ResourceConfig.AssetType]
	if assetDef == nil || assetDef.Lifecycle == nil {
		resourceLogger.Warn("Cannot destroy resource efficiently: asset definition or lifecycle not found.")
		return
	}

	destroyHandlerName := assetDef.Lifecycle.Destroy
	destroyHandler, ok := e.registry.AssetHandlerRegistry[destroyHandlerName]

	if !ok || destroyHandler.DestroyFn == nil {
		resourceLogger.Warn("Cannot destroy resource efficiently: destroy handler not found or is nil.", "handler", destroyHandlerName)
		return
	}

	cleanupFunc := func() {
		resourceLogger.Info("🔥 Destroying resource")
		reflect.ValueOf(destroyHandler.DestroyFn).Call([]reflect.Value{reflect.ValueOf(instance)})
		e.resourceInstances.Delete(node.ID())
	}

	node.Destroy(cleanupFunc)
}
