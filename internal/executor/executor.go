package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/dag"
	"github.com/vk/burstgridgo/internal/registry"
)

// Executor runs the tasks in a graph concurrently.
type Executor struct {
	Graph             *dag.Graph
	wg                sync.WaitGroup
	resourceInstances sync.Map
	cleanupStack      []func()
	cleanupMutex      sync.Mutex
	registry          *registry.Registry
	numWorkers        int
}

// New creates a new graph executor.
func New(
	graph *dag.Graph,
	numWorkers int,
	reg *registry.Registry,
) *Executor {
	if numWorkers <= 0 {
		numWorkers = 10 // Default to 10 if an invalid number is provided.
	}
	return &Executor{
		Graph:      graph,
		numWorkers: numWorkers,
		registry:   reg,
	}
}

// Run executes the entire graph concurrently and returns an error if any node fails.
// It respects the cancellation signal from the provided context.
func (e *Executor) Run(ctx context.Context) error {
	logger := ctxlog.FromContext(ctx)
	defer e.executeCleanupStack(ctx)

	readyChan := make(chan *dag.Node, len(e.Graph.Nodes))
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger.Debug("Initializing executor, finding root nodes...")
	rootNodeCount := 0
	for _, node := range e.Graph.Nodes {
		if node.DepCount() == 0 {
			logger.Debug("Found root node.", "nodeID", node.ID)
			readyChan <- node
			rootNodeCount++
		}
	}
	logger.Debug("Found all root nodes.", "count", rootNodeCount)

	e.wg.Add(len(e.Graph.Nodes))

	logger.Debug("Starting worker pool.", "workers", e.numWorkers)
	for i := 0; i < e.numWorkers; i++ {
		go e.worker(runCtx, readyChan, cancel, i)
	}

	logger.Info("Waiting for all nodes to complete...")
	e.wg.Wait()
	logger.Info("All nodes completed.")
	close(readyChan)

	var failedNodes []string
	var rootCauseError error
	for _, node := range e.Graph.Nodes {
		if node.GetState() == dag.Failed {
			logger.Error("Node failed execution.", "nodeID", node.ID, "error", node.Error)
			// Check if this node's error is a potential root cause.
			// A "skipped" error is a symptom, not a cause.
			if node.Error != nil && !strings.HasPrefix(node.Error.Error(), "skipped") && !errors.Is(node.Error, context.Canceled) {
				failedNodes = append(failedNodes, node.ID)
				// Prioritize the first "real" error as the root cause.
				if rootCauseError == nil {
					rootCauseError = node.Error
				}
			}
		}
	}

	if rootCauseError != nil {
		// Use %w to wrap the identified root cause error.
		return fmt.Errorf("execution failed for %s: %w", strings.Join(failedNodes, ", "), rootCauseError)
	}

	return nil
}
