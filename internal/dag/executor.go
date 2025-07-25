package dag

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/vk/burstgridgo/internal/engine"
)

// Executor runs the tasks in a graph.
type Executor struct {
	Graph                 *Graph
	wg                    sync.WaitGroup
	resourceInstances     sync.Map // Stores live resource objects, keyed by node.ID
	cleanupStack          []func() // LIFO stack of destroy functions
	cleanupMutex          sync.Mutex
	handlerOverrides      map[string]*engine.RegisteredHandler
	assetHandlerOverrides map[string]*engine.RegisteredAssetHandler
}

// NewExecutor creates a new graph executor.
func NewExecutor(graph *Graph, handlerOverrides map[string]*engine.RegisteredHandler, assetHandlerOverrides map[string]*engine.RegisteredAssetHandler) *Executor {
	return &Executor{
		Graph:                 graph,
		handlerOverrides:      handlerOverrides,
		assetHandlerOverrides: assetHandlerOverrides,
	}
}

// Run executes the entire graph concurrently and returns an error if any node fails.
func (e *Executor) Run() error {
	// Defer the cleanup stack execution to ensure resources are always released.
	defer e.executeCleanupStack()

	readyChan := make(chan *Node, len(e.Graph.Nodes))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initial population of the ready channel with nodes that have no dependencies.
	slog.Debug("Initializing executor, finding root nodes...")
	for _, node := range e.Graph.Nodes {
		if node.depCount.Load() == 0 {
			slog.Debug("Found root node", "nodeID", node.ID)
			readyChan <- node
		}
	}

	e.wg.Add(len(e.Graph.Nodes))

	// Start a pool of workers to process nodes from the ready channel.
	const numWorkers = 10
	slog.Debug("Starting worker pool", "workers", numWorkers)
	for i := 0; i < numWorkers; i++ {
		go e.worker(ctx, readyChan, cancel, i)
	}

	// Wait for all nodes to be processed.
	slog.Info("Waiting for all nodes to complete...")
	e.wg.Wait()
	slog.Info("All nodes completed.")
	close(readyChan)

	// Check for any failed nodes and report them.
	var failedNodes []string
	for _, node := range e.Graph.Nodes {
		if node.State.Load() == int32(Failed) {
			slog.Error("Node failed execution", "nodeID", node.ID, "error", node.Error)
			failedNodes = append(failedNodes, node.ID)
		}
	}
	if len(failedNodes) > 0 {
		return fmt.Errorf("execution failed for: %s", strings.Join(failedNodes, ", "))
	}
	return nil
}

// worker is the core processing loop for a single concurrent worker.
func (e *Executor) worker(ctx context.Context, readyChan chan *Node, cancel context.CancelFunc, workerID int) {
	slog.Debug("Worker started", "workerID", workerID)
	for node := range readyChan {
		logger := slog.With("workerID", workerID, "nodeID", node.ID, "nodeType", node.Type)

		// If the context was canceled, mark the node as failed and exit.
		if ctx.Err() != nil {
			logger.Warn("Context canceled, skipping node execution.")
			node.State.Store(int32(Failed))
			e.wg.Done()
			continue
		}

		logger.Debug("Worker picked up node for execution")
		// Execute the node based on its type.
		var err error
		switch node.Type {
		case ResourceNode:
			err = e.executeResourceNode(ctx, node)
		case StepNode:
			err = e.executeStepNode(ctx, node)
		}

		// If execution failed, mark it, record the error, and cancel all other workers.
		if err != nil {
			logger.Error("Node execution failed", "error", err)
			node.State.Store(int32(Failed))
			node.Error = err
			cancel() // Fail-fast
			e.wg.Done()
			continue
		}

		// If execution succeeded, mark it as Done.
		logger.Debug("Node execution succeeded")
		node.State.Store(int32(Done))

		// Decrement the dependency counter for all dependent nodes. If a dependent's
		// counter reaches zero, it is now ready to run.
		for _, dependent := range node.Dependents {
			if dependent.depCount.Add(-1) == 0 {
				logger.Debug("Unlocking dependent node", "dependentID", dependent.ID)
				readyChan <- dependent
			}
		}

		// After a step completes, decrement the descendant counter on its resource dependencies.
		// If a resource's counter reaches zero, it's no longer needed and can be destroyed.
		if node.Type == StepNode {
			for _, dep := range node.Deps {
				if dep.Type == ResourceNode {
					if dep.descendantCount.Add(-1) == 0 {
						logger.Debug("Scheduling efficient destruction for resource", "resourceID", dep.ID)
						go e.destroyResource(dep)
					}
				}
			}
		}

		e.wg.Done()
	}
	slog.Debug("Worker finished", "workerID", workerID)
}
