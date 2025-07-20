package dag

import (
	"log"
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

// Run executes the entire graph concurrently.
func (e *Executor) Run() {
	// A channel to feed ready-to-run nodes to workers.
	readyChan := make(chan *Node, len(e.Graph.Nodes))

	// Add all root nodes (those with no dependencies) to the channel to start.
	for _, node := range e.Graph.Nodes {
		if len(node.Deps) == 0 {
			readyChan <- node
		}
	}

	e.wg.Add(len(e.Graph.Nodes))

	// Start worker goroutines.
	for i := 0; i < 4; i++ {
		go e.worker(readyChan)
	}

	e.wg.Wait()
	close(readyChan)
}

func (e *Executor) worker(readyChan chan *Node) {
	for node := range readyChan {
		e.executeNode(node)

		// After execution, check dependents to see if they are now ready.
		e.nodeMutex.Lock()
		for _, dependent := range node.Dependents {
			if e.areDepsMet(dependent) {
				readyChan <- dependent
			}
		}
		e.nodeMutex.Unlock()
	}
}

func (e *Executor) executeNode(node *Node) {
	defer e.wg.Done()

	log.Printf("  ▶️ Starting module '%s'...", node.Name)
	node.mu.Lock()
	node.State = Running
	node.mu.Unlock()

	// 1. Build the evaluation context from completed dependencies.
	ctx := e.buildEvalContext(node)

	// 2. Look up the runner and execute it.
	if runner, ok := engine.Registry[node.Module.Runner]; ok {
		output, err := runner.Run(*node.Module, ctx)
		if err != nil {
			log.Fatalf("    ❗️ Error executing module '%s': %v", node.Name, err)
			node.mu.Lock()
			node.State = Failed
			node.Error = err
			node.mu.Unlock()
			return // Don't trigger dependents if this node failed.
		}
		// Store the output on the node.
		node.mu.Lock()
		node.Output = output
		node.mu.Unlock()
	} else {
		log.Fatalf("    ❓ Unknown runner type '%s' for module '%s'", node.Module.Runner, node.Name)
		node.mu.Lock()
		node.State = Failed
		node.mu.Unlock()
		return
	}

	node.mu.Lock()
	node.State = Done
	node.mu.Unlock()
	log.Printf("  ✅ Finished module '%s'.", node.Name)
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

	// Populate with outputs from dependencies
	for depName, depNode := range node.Deps {
		depNode.mu.RLock()
		moduleOutputs[depName] = depNode.Output
		depNode.mu.RUnlock()
	}

	// Structure it for HCL: {"module": {"dep_name": ...}}
	vars["module"] = cty.ObjectVal(moduleOutputs)

	return &hcl.EvalContext{
		Variables: vars,
	}
}
