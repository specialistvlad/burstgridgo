package dag

import (
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
	Graph     *Graph
	wg        sync.WaitGroup
	nodeMutex sync.Mutex
}

func NewExecutor(graph *Graph) *Executor {
	return &Executor{Graph: graph}
}

// Run executes the entire graph concurrently and returns an error if any node fails.
func (e *Executor) Run() error {
	readyChan := make(chan *Node, len(e.Graph.Nodes))
	defer close(readyChan)

	for _, node := range e.Graph.Nodes {
		if len(node.Deps) == 0 {
			readyChan <- node
		}
	}

	e.wg.Add(len(e.Graph.Nodes))

	const numWorkers = 4 // This could be configurable later
	for i := 0; i < numWorkers; i++ {
		go e.worker(readyChan)
	}

	e.wg.Wait()

	var failedModules []string
	for _, node := range e.Graph.Nodes {
		node.mu.RLock()
		if node.State == Failed {
			failedModules = append(failedModules, node.Name)
		}
		node.mu.RUnlock()
	}

	if len(failedModules) > 0 {
		return fmt.Errorf("modules failed: %s", strings.Join(failedModules, ", "))
	}

	return nil
}

func (e *Executor) worker(readyChan chan *Node) {
	for node := range readyChan {
		if err := e.executeNode(node); err != nil {
			continue
		}

		e.nodeMutex.Lock()
		for _, dependent := range node.Dependents {
			if e.areDepsMet(dependent) {
				readyChan <- dependent
			}
		}
		e.nodeMutex.Unlock()
	}
}

func (e *Executor) executeNode(node *Node) error {
	defer e.wg.Done()

	logger := slog.With("module", node.Name, "runner", node.Module.Runner)
	logger.Info("▶️ Starting module")

	node.mu.Lock()
	node.State = Running
	node.mu.Unlock()

	ctx := e.buildEvalContext(node)

	runner, ok := engine.Registry[node.Module.Runner]
	if !ok {
		err := fmt.Errorf("unknown runner type '%s'", node.Module.Runner)
		logger.Error("Module execution failed", "error", err)
		node.mu.Lock()
		node.State = Failed
		node.Error = err
		node.mu.Unlock()
		return err
	}

	output, err := runner.Run(*node.Module, ctx)
	if err != nil {
		logger.Error("Module execution failed", "error", err)
		node.mu.Lock()
		node.State = Failed
		node.Error = err
		node.mu.Unlock()
		return err
	}

	node.mu.Lock()
	node.Output = output
	node.State = Done
	node.mu.Unlock()
	logger.Info("✅ Finished module")
	return nil
}

// areDepsMet checks if all dependencies of a given node are in the Done state.
func (e *Executor) areDepsMet(node *Node) bool {
	for _, dep := range node.Deps {
		dep.mu.RLock()
		state := dep.State
		dep.mu.RUnlock()
		if state != Done {
			return false
		}
	}
	return true
}

// buildEvalContext creates the HCL evaluation context for a node.
func (e *Executor) buildEvalContext(node *Node) *hcl.EvalContext {
	vars := make(map[string]cty.Value)
	moduleOutputs := make(map[string]cty.Value)

	for depName, depNode := range node.Deps {
		depNode.mu.RLock()
		moduleOutputs[depName] = depNode.Output
		depNode.mu.RUnlock()
	}

	vars["module"] = cty.ObjectVal(moduleOutputs)

	return &hcl.EvalContext{
		Variables: vars,
	}
}
