package executor

import (
	"context"
	"fmt"

	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/dag"
)

// worker is the core processing loop for a single concurrent worker.
func (e *Executor) worker(ctx context.Context, readyChan chan *dag.Node, cancel context.CancelFunc, workerID int) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Worker started.", "workerID", workerID)

	for node := range readyChan {
		workerLogger := logger.With("workerID", workerID, "nodeID", node.ID)

		if ctx.Err() != nil {
			node.Skip(ctx.Err(), &e.wg)
			continue
		}

		workerLogger.Debug("Worker picked up node for execution.")
		node.SetState(dag.Running)
		var err error
		switch node.Type {
		case dag.ResourceNode:
			err = e.runResourceNode(ctx, node)
		case dag.StepNode:
			err = e.runStepNode(ctx, node)
		}

		if err != nil {
			workerLogger.Error("Node execution failed.", "error", err)
			node.SetState(dag.Failed)
			node.Error = err
			cancel()
			e.skipDependents(ctx, node)
			e.wg.Done()
			continue
		}

		workerLogger.Debug("Node execution succeeded.")
		node.SetState(dag.Done)

		for _, dependent := range node.Dependents {
			if dependent.DecrementDepCount() == 0 {
				workerLogger.Debug("Unlocking dependent node.", "dependentID", dependent.ID)
				readyChan <- dependent
			}
		}

		if node.Type == dag.StepNode {
			for _, dep := range node.Deps {
				if dep.Type == dag.ResourceNode {
					if dep.DecrementDescendantCount() == 0 {
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

// skipDependents recursively marks all downstream nodes as failed and decrements the WaitGroup.
func (e *Executor) skipDependents(ctx context.Context, node *dag.Node) {
	logger := ctxlog.FromContext(ctx)
	for _, dependent := range node.Dependents {
		err := fmt.Errorf("skipped due to upstream failure of '%s'", node.ID)
		wasSkipped := dependent.Skip(err, &e.wg)
		if wasSkipped {
			logger.Warn("Skipping dependent node due to upstream failure.", "nodeID", dependent.ID, "dependency", node.ID)
			e.skipDependents(ctx, dependent)
		}
	}
}
