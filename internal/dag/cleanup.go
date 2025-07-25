package dag

import (
	"log/slog"
	"reflect"

	"github.com/vk/burstgridgo/internal/engine"
)

// pushCleanup adds a function to the LIFO cleanup stack, guarded by the node's sync.Once.
func (e *Executor) pushCleanup(node *Node, f func()) {
	e.cleanupMutex.Lock()
	defer e.cleanupMutex.Unlock()
	// The function we store on the stack now calls the real logic via sync.Once
	e.cleanupStack = append(e.cleanupStack, func() {
		node.destroyOnce.Do(f)
	})
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

	// Create a closure containing the actual destruction logic.
	cleanupFunc := func() {
		logger.Info("🔥 Destroying resource")
		reflect.ValueOf(destroyHandler.DestroyFn).Call([]reflect.Value{reflect.ValueOf(instance)})
		e.resourceInstances.Delete(node.ID)
	}

	// Use sync.Once to ensure this logic runs at most once.
	node.destroyOnce.Do(cleanupFunc)
}
