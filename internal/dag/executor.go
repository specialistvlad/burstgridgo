package dag

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/vk/burstgridgo/internal/ctxlog"
)

// Run executes the entire graph concurrently and returns an error if any node fails.
// It respects the cancellation signal from the provided context.
func (e *Executor) Run(ctx context.Context) error {
	logger := ctxlog.FromContext(ctx)
	defer e.executeCleanupStack(ctx)

	readyChan := make(chan *Node, len(e.Graph.Nodes))
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger.Debug("Initializing executor, finding root nodes...")
	rootNodeCount := 0
	for _, node := range e.Graph.Nodes {
		if node.depCount.Load() == 0 {
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
		if node.State.Load() == int32(Failed) {
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

// skipDependents recursively marks all downstream nodes as failed and decrements the WaitGroup.
func (e *Executor) skipDependents(ctx context.Context, node *Node) {
	logger := ctxlog.FromContext(ctx)
	for _, dependent := range node.Dependents {
		dependent.skipOnce.Do(func() {
			logger.Warn("Skipping dependent node due to upstream failure.", "nodeID", dependent.ID, "dependency", node.ID)
			dependent.State.Store(int32(Failed))
			dependent.Error = fmt.Errorf("skipped due to upstream failure of '%s'", node.ID)
			e.wg.Done()
			e.skipDependents(ctx, dependent)
		})
	}
}

// worker is the core processing loop for a single concurrent worker.
func (e *Executor) worker(ctx context.Context, readyChan chan *Node, cancel context.CancelFunc, workerID int) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Worker started.", "workerID", workerID)

	for node := range readyChan {
		workerLogger := logger.With("workerID", workerID, "nodeID", node.ID)

		if ctx.Err() != nil {
			node.skipOnce.Do(func() {
				workerLogger.Warn("Context canceled, skipping node execution.")
				node.State.Store(int32(Failed))
				node.Error = ctx.Err()
				e.wg.Done()
			})
			continue
		}

		workerLogger.Debug("Worker picked up node for execution.")
		node.State.Store(int32(Running))
		var err error
		switch node.Type {
		case ResourceNode:
			err = e.executeResourceNode(ctx, node)
		case StepNode:
			err = e.executeStepNode(ctx, node)
		}

		if err != nil {
			workerLogger.Error("Node execution failed.", "error", err)
			node.State.Store(int32(Failed))
			node.Error = err
			cancel()
			e.skipDependents(ctx, node)
			e.wg.Done()
			continue
		}

		workerLogger.Debug("Node execution succeeded.")
		node.State.Store(int32(Done))

		for _, dependent := range node.Dependents {
			if dependent.depCount.Add(-1) == 0 {
				workerLogger.Debug("Unlocking dependent node.", "dependentID", dependent.ID)
				readyChan <- dependent
			}
		}

		if node.Type == StepNode {
			for _, dep := range node.Deps {
				if dep.Type == ResourceNode {
					if dep.descendantCount.Add(-1) == 0 {
						workerLogger.Debug("Scheduling efficient destruction for resource.", "resourceID", dep.ID)
						go e.destroyResource(ctx, dep)
					}
				}
			}
		}

		e.wg.Done()
	}
	logger.Debug("Worker finished.", "workerID", workerID)
}
