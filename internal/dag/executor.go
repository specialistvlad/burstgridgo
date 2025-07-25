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

	// Initial population of the ready channel
	for _, node := range e.Graph.Nodes {
		if node.depCount.Load() == 0 {
			readyChan <- node
		}
	}

	e.wg.Add(len(e.Graph.Nodes))

	const numWorkers = 10 // Increased worker pool size
	for i := 0; i < numWorkers; i++ {
		go e.worker(ctx, readyChan, cancel)
	}

	e.wg.Wait()
	close(readyChan)

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

func (e *Executor) worker(ctx context.Context, readyChan chan *Node, cancel context.CancelFunc) {
	for node := range readyChan {
		if ctx.Err() != nil {
			node.State.Store(int32(Failed))
			e.wg.Done()
			continue
		}

		var err error
		switch node.Type {
		case ResourceNode:
			err = e.executeResourceNode(ctx, node)
		case StepNode:
			err = e.executeStepNode(ctx, node)
		}

		if err != nil {
			node.State.Store(int32(Failed))
			node.Error = err
			// Fail-fast: cancel the context for all other workers.
			cancel()
			e.wg.Done() // wg.Done() is called once per node, even on failure.
			continue
		}

		node.State.Store(int32(Done))

		// Trigger dependents
		for _, dependent := range node.Dependents {
			if dependent.depCount.Add(-1) == 0 {
				readyChan <- dependent
			}
		}

		// After a step completes, decrement the counter on its resource dependencies.
		if node.Type == StepNode {
			for _, dep := range node.Deps {
				if dep.Type == ResourceNode {
					if dep.descendantCount.Add(-1) == 0 {
						// Efficiently destroy resource as soon as it's no longer needed.
						// This is fire-and-forget; the main cleanup stack is the safety net.
						go e.destroyResource(dep)
					}
				}
			}
		}
		e.wg.Done()
	}
}

// executeResourceNode handles the creation of a stateful resource.
func (e *Executor) executeResourceNode(ctx context.Context, node *Node) error {
	logger := slog.With("resource", node.ID)
	logger.Info("▶️ Creating resource")

	assetType := node.ResourceConfig.AssetType

	// 1. Check if the asset type itself is defined in a manifest.
	assetDef, ok := engine.AssetDefinitionRegistry[assetType]
	if !ok {
		return fmt.Errorf("unknown asset type '%s' referenced by resource '%s'", assetType, node.ID)
	}

	// 2. Check if the asset definition has a lifecycle block.
	if assetDef.Lifecycle == nil {
		return fmt.Errorf("asset type '%s' has no 'lifecycle' block defined in its manifest", assetType)
	}

	// 3. Check if the required 'create' and 'destroy' handlers are specified in the lifecycle.
	createHandlerName := assetDef.Lifecycle.Create
	destroyHandlerName := assetDef.Lifecycle.Destroy
	if createHandlerName == "" {
		return fmt.Errorf("asset type '%s' is missing 'create' handler name in its lifecycle", assetType)
	}
	if destroyHandlerName == "" {
		return fmt.Errorf("asset type '%s' is missing 'destroy' handler name in its lifecycle", assetType)
	}

	// 4. Find the Go functions for these handlers, checking overrides first.
	assetHandler, ok := e.assetHandlerOverrides[createHandlerName]
	if !ok {
		assetHandler, ok = engine.AssetHandlerRegistry[createHandlerName]
	}
	if !ok || assetHandler.CreateFn == nil {
		return fmt.Errorf("create handler '%s' for asset type '%s' is not registered or is nil", createHandlerName, assetType)
	}

	destroyFn, ok := e.assetHandlerOverrides[destroyHandlerName]
	if !ok {
		destroyFn, ok = engine.AssetHandlerRegistry[destroyHandlerName]
	}
	if !ok || destroyFn.DestroyFn == nil {
		return fmt.Errorf("destroy handler '%s' for asset type '%s' is not registered or is nil", destroyHandlerName, assetType)
	}

	// --- Validation Passed: Proceed with Execution ---

	// Decode arguments, if the block exists.
	inputStruct := assetHandler.NewInput()
	evalCtx := e.buildEvalContext(node)
	if node.ResourceConfig.Arguments != nil {
		if diags := gohcl.DecodeBody(node.ResourceConfig.Arguments.Body, evalCtx, inputStruct); diags.HasErrors() {
			return diags
		}
	}

	// Call Create handler
	handlerFunc := reflect.ValueOf(assetHandler.CreateFn)
	results := handlerFunc.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(inputStruct)})
	resourceObj := results[0].Interface()
	errResult := results[1].Interface()
	if errResult != nil {
		return errResult.(error)
	}

	// Store instance and register destroy function
	node.Output = resourceObj
	e.resourceInstances.Store(node.ID, resourceObj)
	e.pushCleanup(func() {
		logger.Info("🔥 Destroying resource via deferred cleanup")
		reflect.ValueOf(destroyFn.DestroyFn).Call([]reflect.Value{reflect.ValueOf(resourceObj)})
	})

	logger.Info("✅ Resource created")
	return nil
}

// executeStepNode handles the execution of a stateless step.
func (e *Executor) executeStepNode(ctx context.Context, node *Node) error {
	logger := slog.With("step", node.ID)
	logger.Info("▶️ Starting step")

	// Find runner definition and handler
	runnerDef, ok := engine.DefinitionRegistry[node.StepConfig.RunnerType]
	if !ok {
		return fmt.Errorf("unknown runner type '%s'", node.StepConfig.RunnerType)
	}
	handlerName := runnerDef.Lifecycle.OnRun
	registeredHandler, ok := e.handlerOverrides[handlerName] // Check overrides first
	if !ok {
		registeredHandler, ok = engine.HandlerRegistry[handlerName] // Fallback to global registry
	}
	if !ok {
		return fmt.Errorf("handler '%s' for runner '%s' not registered or provided as override", handlerName, runnerDef.Type)
	}

	// Decode 'arguments' block
	inputStruct := registeredHandler.NewInput()
	evalCtx := e.buildEvalContext(node)

	if inputStruct != nil && node.StepConfig.Arguments != nil {
		if diags := gohcl.DecodeBody(node.StepConfig.Arguments.Body, evalCtx, inputStruct); diags.HasErrors() {
			return diags
		}
		// After decoding user-provided args, apply defaults from the manifest for any missing fields.
		if err := applyInputDefaults(inputStruct, runnerDef, node.StepConfig.Arguments.Body); err != nil {
			return fmt.Errorf("error applying default values for step %s: %w", node.ID, err)
		}
	}

	// Build 'deps' struct from the 'uses' block
	depsStruct, err := e.buildDepsStruct(node, runnerDef, registeredHandler, evalCtx)
	if err != nil {
		return err
	}

	// Call handler
	handlerFunc := reflect.ValueOf(registeredHandler.Fn)
	callArgs := []reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(depsStruct),
	}
	// Handle nil input struct for handlers that don't take arguments
	if inputStruct == nil {
		// Pass the zero value for the handler's expected input type
		// For `any` (interface{}), the zero value is nil.
		inputType := handlerFunc.Type().In(2)
		callArgs = append(callArgs, reflect.Zero(inputType))
	} else {
		callArgs = append(callArgs, reflect.ValueOf(inputStruct))
	}

	results := handlerFunc.Call(callArgs)
	outputVal := results[0].Interface()
	errResult := results[1].Interface()

	if errResult != nil {
		return errResult.(error)
	}

	// Ensure output is a cty.Value, if not nil
	if outputVal == nil {
		node.Output = cty.NilVal
	} else {
		ctyOutput, ok := outputVal.(cty.Value)
		if !ok {
			return fmt.Errorf("handler for step %s returned a non-cty.Value type: %T", node.ID, outputVal)
		}
		node.Output = ctyOutput
	}

	logger.Info("✅ Finished step")
	return nil
}

// buildDepsStruct populates the `deps` struct for a step via reflection.
func (e *Executor) buildDepsStruct(node *Node, runnerDef *engine.RunnerDefinition, handler *engine.RegisteredHandler, evalCtx *hcl.EvalContext) (any, error) {
	depsStruct := handler.NewDeps()
	if node.StepConfig.Uses == nil {
		return depsStruct, nil // No `uses` block, return empty deps struct.
	}

	// The uses block maps local names (fields in Deps struct) to resource IDs.
	// e.g., { client = resource.http_client.shared }
	usesMap := make(map[string]hcl.Expression)
	attrs, diags := node.StepConfig.Uses.Body.JustAttributes()
	if diags.HasErrors() {
		return nil, diags
	}
	for _, attr := range attrs {
		usesMap[attr.Name] = attr.Expr
	}

	depsValue := reflect.ValueOf(depsStruct).Elem()
	for i := 0; i < depsValue.NumField(); i++ {
		field := depsValue.Type().Field(i)
		localName := field.Name

		// Find which resource this field maps to from the `uses` block
		resourceExpr, ok := usesMap[localName]
		if !ok {
			continue // This field in the Deps struct isn't set in the HCL.
		}

		// The expression should be a variable traversal, e.g., `resource.http_client.shared`
		vars := resourceExpr.Variables()
		if len(vars) != 1 {
			return nil, fmt.Errorf("field '%s' in 'uses' block must be a direct reference to a single resource", localName)
		}
		resourceID, err := traversableToID(vars[0])
		if err != nil {
			return nil, err
		}

		// Get the live resource object instance
		instance, found := e.resourceInstances.Load(resourceID)
		if !found {
			return nil, fmt.Errorf("step '%s' requires resource '%s' which has not been created", node.ID, resourceID)
		}

		// Type-check: ensure the live object implements the interface expected by the field.
		if !reflect.TypeOf(instance).Implements(field.Type) {
			return nil, fmt.Errorf("type mismatch for '%s': resource '%s' of type %T does not implement required interface %s", localName, resourceID, instance, field.Type)
		}

		// Set the field in the deps struct
		depsValue.Field(i).Set(reflect.ValueOf(instance))
	}

	return depsStruct, nil
}

// buildEvalContext creates the HCL evaluation context for a node.
func (e *Executor) buildEvalContext(node *Node) *hcl.EvalContext {
	vars := make(map[string]cty.Value)

	// Use a standard Go map for building the structure first.
	// The structure is: map[runnerType] -> map[instanceName] -> cty.Value
	stepOutputsByRunner := make(map[string]map[string]cty.Value)

	for _, depNode := range node.Deps {
		if depNode.Type == StepNode {
			// A step can only be a dependency if it's successfully completed.
			if depNode.State.Load() != int32(Done) || depNode.Output == nil {
				continue
			}
			runnerType := depNode.StepConfig.RunnerType
			instanceName := depNode.Name

			// Get or create the inner map for this runner type.
			if _, ok := stepOutputsByRunner[runnerType]; !ok {
				stepOutputsByRunner[runnerType] = make(map[string]cty.Value)
			}

			// Assign the output to the instance name in the inner map.
			// This is a simple, safe map assignment.
			stepOutputsByRunner[runnerType][instanceName] = cty.ObjectVal(map[string]cty.Value{
				"output": depNode.Output.(cty.Value),
			})
		}
	}

	// After building the Go map, convert it to the final cty.Value structure.
	finalStepOutputs := make(map[string]cty.Value)
	for runnerType, instancesMap := range stepOutputsByRunner {
		finalStepOutputs[runnerType] = cty.ObjectVal(instancesMap)
	}

	vars["step"] = cty.ObjectVal(finalStepOutputs)
	// Context for resources can be added here if they produce cty.Value outputs.
	return &hcl.EvalContext{Variables: vars}
}

func (e *Executor) pushCleanup(f func()) {
	e.cleanupMutex.Lock()
	defer e.cleanupMutex.Unlock()
	e.cleanupStack = append(e.cleanupStack, f)
}

func (e *Executor) executeCleanupStack() {
	e.cleanupMutex.Lock()
	defer e.cleanupMutex.Unlock()
	slog.Info("Executing cleanup stack...")
	for i := len(e.cleanupStack) - 1; i >= 0; i-- {
		e.cleanupStack[i]()
	}
	e.cleanupStack = nil // Clear the stack
}

func (e *Executor) destroyResource(node *Node) {
	instance, found := e.resourceInstances.Load(node.ID)
	if !found {
		return // Already destroyed or never created.
	}

	logger := slog.With("resource", node.ID)
	logger.Info("🔥 Destroying resource efficiently")

	assetDef := engine.AssetDefinitionRegistry[node.ResourceConfig.AssetType]
	if assetDef == nil || assetDef.Lifecycle == nil {
		logger.Warn("Cannot destroy resource efficiently: asset definition or lifecycle not found.")
		return
	}

	destroyHandlerName := assetDef.Lifecycle.Destroy
	destroyHandler, ok := e.assetHandlerOverrides[destroyHandlerName]
	if !ok {
		destroyHandler, ok = engine.AssetHandlerRegistry[destroyHandlerName]
	}

	if !ok || destroyHandler.DestroyFn == nil {
		logger.Warn("Cannot destroy resource efficiently: destroy handler not found or is nil.", "handler", destroyHandlerName)
		return
	}

	reflect.ValueOf(destroyHandler.DestroyFn).Call([]reflect.Value{reflect.ValueOf(instance)})
	e.resourceInstances.Delete(node.ID)
}

func traversableToID(v hcl.Traversal) (string, error) {
	if len(v) < 3 {
		return "", fmt.Errorf("invalid resource traversal")
	}
	root := v.RootName()
	if root != "resource" {
		return "", fmt.Errorf("expected a 'resource' traversal, got '%s'", root)
	}
	return fmt.Sprintf("resource.%s.%s", v[1].(hcl.TraverseAttr).Name, v[2].(hcl.TraverseAttr).Name), nil
}

// applyInputDefaults uses reflection to apply default values from a runner's
// manifest to an input struct for any fields the user did not provide.
func applyInputDefaults(inputStruct any, runnerDef *engine.RunnerDefinition, userBody hcl.Body) error {
	if inputStruct == nil || runnerDef == nil || userBody == nil {
		return nil
	}

	// Get the set of attributes the user explicitly provided in their HCL.
	userAttrs, _ := userBody.JustAttributes()
	userProvidedNames := make(map[string]struct{})
	for name := range userAttrs {
		userProvidedNames[name] = struct{}{}
	}

	structVal := reflect.ValueOf(inputStruct).Elem()
	structType := structVal.Type()

	// Iterate over the inputs defined in the manifest.
	for _, inputDef := range runnerDef.Inputs {
		// If the user already provided this input, or if there's no default, skip.
		if _, ok := userProvidedNames[inputDef.Name]; ok || inputDef.Default == nil {
			continue
		}

		// Find the corresponding field in the Go struct by its HCL tag name.
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			tag := field.Tag.Get("hcl")
			tagName := strings.Split(tag, ",")[0]

			if tagName == inputDef.Name {
				// We found the matching field. Apply the default value.
				fieldVal := structVal.Field(i)
				if fieldVal.CanSet() {
					// Use gocty to convert the default cty.Value to the field's native Go type.
					if err := gocty.FromCtyValue(*inputDef.Default, fieldVal.Addr().Interface()); err != nil {
						return fmt.Errorf("failed to apply default for input '%s': %w", inputDef.Name, err)
					}
				}
				break // Move to the next input definition.
			}
		}
	}

	return nil
}
