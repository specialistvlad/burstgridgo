package executor

import (
	"context"

	"github.com/vk/burstgridgo/internal/builder"
	"github.com/vk/burstgridgo/internal/ctxlog"
)

// worker is the core processing loop for a single concurrent worker.
func (e *Executor) worker(ctx context.Context, readyChan chan *builder.Node, cancel context.CancelFunc, workerID int) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Worker started.", "workerID", workerID)

	for node := range readyChan {
		workerLogger := logger.With("workerID", workerID, "nodeID", node.ID)

		if ctx.Err() != nil {
			node.Skip(ctx.Err(), &e.wg)
			continue
		}

		workerLogger.Debug("Worker picked up node for execution.")
		node.SetState(builder.Running)
		var err error

		// Main logic fork: handle placeholders differently from normal nodes.
		if node.IsPlaceholder {
			err = e.runPlaceholderNode(ctx, node)
		} else {
			switch node.Type {
			case builder.ResourceNode:
				err = e.runResourceNode(ctx, node)
			case builder.StepNode:
				err = e.runStepNode(ctx, node)
			}
		}

		if err != nil {
			workerLogger.Error("Node execution failed.", "error", err)
			node.SetState(builder.Failed)
			node.Error = err
			cancel()
			e.skipDependents(ctx, node)
			e.wg.Done()
			continue
		}

		workerLogger.Debug("Node execution succeeded.")
		node.SetState(builder.Done)

		// --- REFACTORED SECTION 1: Unlocking dependents ---
		// We now query the graph for the authoritative list of dependents.
		dependents, err := e.Graph.Dependents(node.ID)
		if err != nil {
			workerLogger.Error("Failed to get dependents for completed node", "nodeID", node.ID, "error", err)
		} else {
			for _, dependent := range dependents {
				if dependent.DecrementDepCount() == 0 {
					workerLogger.Debug("Unlocking dependent node.", "dependentID", dependent.ID)
					readyChan <- dependent
				}
			}
		}

		// Don't check for resource destruction on placeholder nodes
		if node.Type == builder.StepNode && !node.IsPlaceholder {
			// --- REFACTORED SECTION 2: Checking dependencies for resource cleanup ---
			// We query the graph for the authoritative list of dependencies.
			dependencies, err := e.Graph.Dependencies(node.ID)
			if err != nil {
				workerLogger.Error("Failed to get dependencies for completed node", "nodeID", node.ID, "error", err)
			} else {
				for _, dep := range dependencies {
					if dep.Type == builder.ResourceNode {
						if dep.DecrementDescendantCount() == 0 {
							workerLogger.Debug("Scheduling efficient destruction for resource.", "resourceID", dep.ID)
							go e.destroyResource(ctx, dep)
						}
					}
				}
			}
		}

		e.wg.Done()
	}
	logger.Debug("Worker finished.", "workerID", workerID)
}
