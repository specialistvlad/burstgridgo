package executor

import (
	"context"

	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/specialistvlad/burstgridgo/internal/node"
)

// worker is the core processing loop for a single concurrent worker.
func (e *Executor) worker(ctx context.Context, readyChan chan *node.Node, cancel context.CancelFunc, workerID int) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Worker started.", "workerID", workerID)

	for n := range readyChan {
		workerLogger := logger.With("workerID", workerID, "nodeID", n.ID())

		if ctx.Err() != nil {
			n.Skip(ctx.Err(), &e.wg)
			continue
		}

		workerLogger.Debug("Worker picked up node for execution.")
		n.SetState(node.Running)
		var err error

		// Main logic fork: handle placeholders differently from normal nodes.
		if n.IsPlaceholder {
			err = e.runPlaceholderNode(ctx, n)
		} else {
			switch n.Type {
			case node.ResourceNode:
				err = e.runResourceNode(ctx, n)
			case node.StepNode:
				err = e.runStepNode(ctx, n)
			}
		}

		if err != nil {
			workerLogger.Error("Node execution failed.", "error", err)
			n.SetState(node.Failed)
			n.Error = err
			cancel()
			e.skipDependents(ctx, n)
			e.wg.Done()
			continue
		}

		workerLogger.Debug("Node execution succeeded.")
		n.SetState(node.Done)

		// --- REFACTORED SECTION 1: Unlocking dependents ---
		// We now query the graph for the authoritative list of dependents.
		dependents, err := e.Graph.Dependents(n.ID())
		if err != nil {
			workerLogger.Error("Failed to get dependents for completed node", "nodeID", n.ID(), "error", err)
		} else {
			for _, dependent := range dependents {
				if dependent.DecrementDepCount() == 0 {
					workerLogger.Debug("Unlocking dependent node.", "dependentID", dependent.ID())
					readyChan <- dependent
				}
			}
		}

		// Don't check for resource destruction on placeholder nodes
		if n.Type == node.StepNode && !n.IsPlaceholder {
			// --- REFACTORED SECTION 2: Checking dependencies for resource cleanup ---
			// We query the graph for the authoritative list of dependencies.
			dependencies, err := e.Graph.Dependencies(n.ID())
			if err != nil {
				workerLogger.Error("Failed to get dependencies for completed node", "nodeID", n.ID(), "error", err)
			} else {
				for _, dep := range dependencies {
					if dep.Type == node.ResourceNode {
						if dep.DecrementDescendantCount() == 0 {
							workerLogger.Debug("Scheduling efficient destruction for resource.", "resourceID", dep.ID())
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
