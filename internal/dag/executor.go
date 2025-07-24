package dag

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
		return fmt.Errorf("steps failed: %s", strings.Join(failedModules, ", "))
	}

	return nil
}

func (e *Executor) worker(ctx context.Context, readyChan chan *Node) {
	for node := range readyChan {
		if ctx.Err() != nil {
			node.State.Store(int32(Failed))
			e.wg.Done()
			continue
		}

		if err := e.executeNode(ctx, node); err != nil {
			// In the future, we could have a "fail-fast" mode that calls cancel() here.
			continue
		}

		for _, dependent := range node.Dependents {
			if dependent.depCount.Add(-1) == 0 {
				readyChan <- dependent
			}
		}
	}
}

func (e *Executor) executeNode(ctx context.Context, node *Node) error {
	defer e.wg.Done()

	logger := slog.With("step", node.Name, "runner", node.Step.RunnerType)
	logger.Info("▶️ Starting step")

	node.State.Store(int32(Running))

	// 1. Find the runner's definition from the registry.
	runnerDef, ok := engine.DefinitionRegistry[node.Step.RunnerType]
	if !ok {
		err := fmt.Errorf("unknown runner type '%s'", node.Step.RunnerType)
		logger.Error("Step execution failed", "error", err)
		node.Error = err
		node.State.Store(int32(Failed))
		return err
	}

	// For now, we only support the on_run lifecycle event.
	if runnerDef.Lifecycle == nil || runnerDef.Lifecycle.OnRun == "" {
		err := fmt.Errorf("runner '%s' has no on_run lifecycle handler defined", runnerDef.Type)
		logger.Error("Step execution failed", "error", err)
		node.Error = err
		node.State.Store(int32(Failed))
		return err
	}
	handlerName := runnerDef.Lifecycle.OnRun

	// 2. Find the registered Go handler function.
	registeredHandler, ok := engine.HandlerRegistry[handlerName]
	if !ok {
		err := fmt.Errorf("handler '%s' for runner '%s' is not registered", handlerName, runnerDef.Type)
		logger.Error("Step execution failed", "error", err)
		node.Error = err
		node.State.Store(int32(Failed))
		return err
	}

	var inputStruct any
	// Only create inputStruct and decode if the handler's NewInput function is provided.
	if registeredHandler.NewInput != nil {
		inputStruct = registeredHandler.NewInput()
		if inputStruct == nil {
			err := fmt.Errorf("handler '%s' NewInput function returned nil unexpectedly", handlerName)
			logger.Error("Step execution failed", "error", err)
			node.Error = err
			node.State.Store(int32(Failed))
			return err
		}

		// Check if the Arguments body is available before decoding
		var argsBody hcl.Body
		if node.Step.Arguments != nil { // Check if the Arguments struct exists
			argsBody = node.Step.Arguments.Body
		}
		evalCtx := e.buildEvalContext(node)
		if diags := gohcl.DecodeBody(argsBody, evalCtx, inputStruct); diags.HasErrors() {
			logger.Error("Step execution failed", "error", diags)
			node.Error = diags
			node.State.Store(int32(Failed))
			return diags
		}
	}

	// 4. Call the handler function using reflection.
	handlerFunc := reflect.ValueOf(registeredHandler.Fn)
	callArgs := []reflect.Value{
		reflect.ValueOf(ctx),
	}

	// Add the input argument only if the handler's signature expects it (i.e., has more than 1 argument).
	// This makes it compatible with handlers that have no HCL input struct (like 'help').
	if handlerFunc.Type().NumIn() > 1 {
		// If inputStruct is nil (because NewInput was nil), pass the zero value of the expected type.
		// For 'any', this will be a nil interface value.
		if inputStruct == nil {
			callArgs = append(callArgs, reflect.Zero(handlerFunc.Type().In(1)))
		} else {
			callArgs = append(callArgs, reflect.ValueOf(inputStruct))
		}
	}

	// This is a simplified call for now. It assumes a signature of func(ctx, *Input) (*Output, error)
	// and doesn't yet handle the state object.
	results := handlerFunc.Call(callArgs)
	outputStruct := results[0].Interface()
	errResult := results[1].Interface()

	if errResult != nil {
		err := errResult.(error)
		logger.Error("Step execution failed", "error", err)
		node.Error = err
		node.State.Store(int32(Failed))
		return err
	}

	// 5. Convert the Go output struct back to a cty.Value for downstream steps.
	if outputStruct != nil && !reflect.ValueOf(outputStruct).IsNil() {
		outputVal, err := gocty.ToCtyValue(outputStruct, cty.DynamicPseudoType)
		if err != nil {
			err := fmt.Errorf("failed to convert runner output to HCL value: %w", err)
			logger.Error("Step execution failed", "error", err)
			node.Error = err
			node.State.Store(int32(Failed))
			return err
		}
		node.Output = outputVal
	} else {
		node.Output = cty.NilVal
	}

	node.State.Store(int32(Done))
	logger.Info("✅ Finished step")
	return nil
}

// buildEvalContext creates the HCL evaluation context for a node.
func (e *Executor) buildEvalContext(node *Node) *hcl.EvalContext {
	vars := make(map[string]cty.Value)
	stepOutputs := make(map[string]cty.Value)

	for depName, depNode := range node.Deps {
		stepOutputs[depName] = depNode.Output
	}

	vars["step"] = cty.ObjectVal(stepOutputs)

	return &hcl.EvalContext{
		Variables: vars,
	}
}
