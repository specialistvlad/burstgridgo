package dag

import (
	"log/slog"
	"reflect"

	"github.com/vk/burstgridgo/internal/engine"
)

// pushCleanup adds a function to the LIFO cleanup stack.
func (e *Executor) pushCleanup(f func()) {
	e.cleanupMutex.Lock()
	defer e.cleanupMutex.Unlock()
	e.cleanupStack = append(e.cleanupStack, f)
}

// executeCleanupStack runs all registered cleanup functions in LIFO order.
// This is deferred in Run() to guarantee execution.
func (e *Executor) executeCleanupStack() {
	e.cleanupMutex.Lock()
	defer e.cleanupMutex.Unlock()
	slog.Info("Executing cleanup stack...")
	for i := len(e.cleanupStack) - 1; i >= 0; i-- {
		e.cleanupStack[i]()
	}
	e.cleanupStack = nil // Clear the stack
}

// destroyResource handles the efficient, runtime destruction of a resource
// as soon as it's no longer needed by any downstream steps.
func (e *Executor) destroyResource(node *Node) {
	instance, found := e.resourceInstances.Load(node.ID)
	if !found {
		return // Already destroyed or never created.
	}

	logger := slog.With("resource", node.ID)
	logger.Info("🔥 Destroying resource efficiently")

	assetDef := engine.AssetDefinitionRegistry[node.ResourceConfig.AssetType]
	if assetDef == nil || assetDef.Lifecycle == nil {
		logger.Warn("Cannot destroy resource efficiently: asset definition or lifecycle not found.")
		return
	}

	destroyHandlerName := assetDef.Lifecycle.Destroy
	destroyHandler, ok := e.assetHandlerOverrides[destroyHandlerName]
	if !ok {
		destroyHandler, ok = engine.AssetHandlerRegistry[destroyHandlerName]
	}

	if !ok || destroyHandler.DestroyFn == nil {
		logger.Warn("Cannot destroy resource efficiently: destroy handler not found or is nil.", "handler", destroyHandlerName)
		return
	}

	reflect.ValueOf(destroyHandler.DestroyFn).Call([]reflect.Value{reflect.ValueOf(instance)})
	e.resourceInstances.Delete(node.ID)
}
