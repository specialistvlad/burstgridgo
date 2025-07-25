package dag

import (
	"context"
	"fmt"
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
	for _, node := range e.Graph.Nodes {
		if node.depCount.Load() == 0 {
			readyChan <- node
		}
	}

	e.wg.Add(len(e.Graph.Nodes))

	// Start a pool of workers to process nodes from the ready channel.
	const numWorkers = 10
	for i := 0; i < numWorkers; i++ {
		go e.worker(ctx, readyChan, cancel)
	}

	// Wait for all nodes to be processed.
	e.wg.Wait()
	close(readyChan)

	// Check for any failed nodes and report them.
	var failedNodes []string
	for _, node := range e.Graph.Nodes {
		if node.State.Load() == int32(Failed) {
			failedNodes = append(failedNodes, node.ID)
		}
	}
	if len(failedNodes) > 0 {
		return fmt.Errorf("execution failed for: %s", strings.Join(failedNodes, ", "))
	}
	return nil
}

// worker is the core processing loop for a single concurrent worker.
func (e *Executor) worker(ctx context.Context, readyChan chan *Node, cancel context.CancelFunc) {
	for node := range readyChan {
		// If the context was canceled, mark the node as failed and exit.
		if ctx.Err() != nil {
			node.State.Store(int32(Failed))
			e.wg.Done()
			continue
		}

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
			node.State.Store(int32(Failed))
			node.Error = err
			cancel() // Fail-fast
			e.wg.Done()
			continue
		}

		// If execution succeeded, mark it as Done.
		node.State.Store(int32(Done))

		// Decrement the dependency counter for all dependent nodes. If a dependent's
		// counter reaches zero, it is now ready to run.
		for _, dependent := range node.Dependents {
			if dependent.depCount.Add(-1) == 0 {
				readyChan <- dependent
			}
		}

		// After a step completes, decrement the descendant counter on its resource dependencies.
		// If a resource's counter reaches zero, it's no longer needed and can be destroyed.
		if node.Type == StepNode {
			for _, dep := range node.Deps {
				if dep.Type == ResourceNode {
					if dep.descendantCount.Add(-1) == 0 {
						go e.destroyResource(dep)
					}
				}
			}
		}

		e.wg.Done()
	}
}
