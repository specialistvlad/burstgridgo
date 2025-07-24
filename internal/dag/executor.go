package dag

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
)

// Executor runs the tasks in a graph.
type Executor struct {
	Graph *Graph
	wg    sync.WaitGroup
}

// NewExecutor creates a new graph executor.
func NewExecutor(graph *Graph) *Executor {
	return &Executor{Graph: graph}
}

// Run executes the entire graph concurrently and returns an error if any node fails.
func (e *Executor) Run() error {
	readyChan := make(chan *Node, len(e.Graph.Nodes))
	defer close(readyChan)

	// Create a top-level context for the entire run.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cancellation is signaled on exit.

	// Find all nodes with no dependencies to start.
	for _, node := range e.Graph.Nodes {
		if node.depCount.Load() == 0 {
			readyChan <- node
		}
	}

	e.wg.Add(len(e.Graph.Nodes))

	const numWorkers = 4 // This could be configurable later
	for i := 0; i < numWorkers; i++ {
		go e.worker(ctx, readyChan)
	}

	e.wg.Wait()

	var failedModules []string
	for _, node := range e.Graph.Nodes {
		if node.State.Load() == int32(Failed) {
			failedModules = append(failedModules, node.Name)
		}
	}

	if len(failedModules) > 0 {
		return fmt.Errorf("modules failed: %s", strings.Join(failedModules, ", "))
	}

	return nil
}

func (e *Executor) worker(ctx context.Context, readyChan chan *Node) {
	for node := range readyChan {
		// Stop processing new nodes if the context has been canceled.
		if ctx.Err() != nil {
			node.State.Store(int32(Failed))
			e.wg.Done()
			continue
		}

		// If execution fails, we stop processing this branch of the graph.
		if err := e.executeNode(ctx, node); err != nil {
			continue
		}

		// Atomically decrement the counter of each dependent.
		// If a dependent's counter reaches zero, it's ready to run.
		for _, dependent := range node.Dependents {
			if dependent.depCount.Add(-1) == 0 {
				readyChan <- dependent
			}
		}
	}
}

func (e *Executor) executeNode(ctx context.Context, node *Node) error {
	defer e.wg.Done()

	logger := slog.With("module", node.Name, "runner", node.Module.Runner)
	logger.Info("▶️ Starting module")

	node.State.Store(int32(Running))

	evalCtx := e.buildEvalContext(node)

	runner, ok := engine.Registry[node.Module.Runner]
	if !ok {
		err := fmt.Errorf("unknown runner type '%s'", node.Module.Runner)
		logger.Error("Module execution failed", "error", err)
		node.Error = err
		node.State.Store(int32(Failed))
		return err
	}

	// Pass the context to the runner's Run method.
	output, err := runner.Run(ctx, *node.Module, evalCtx)
	if err != nil {
		logger.Error("Module execution failed", "error", err)
		node.Error = err
		node.State.Store(int32(Failed))
		return err
	}

	node.Output = output
	node.State.Store(int32(Done))
	logger.Info("✅ Finished module")
	return nil
}

// buildEvalContext creates the HCL evaluation context for a node.
func (e *Executor) buildEvalContext(node *Node) *hcl.EvalContext {
	vars := make(map[string]cty.Value)
	moduleOutputs := make(map[string]cty.Value)

	for depName, depNode := range node.Deps {
		moduleOutputs[depName] = depNode.Output
	}

	vars["module"] = cty.ObjectVal(moduleOutputs)

	return &hcl.EvalContext{
		Variables: vars,
	}
}
