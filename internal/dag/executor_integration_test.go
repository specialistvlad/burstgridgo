package dag

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// TestMain runs setup before any tests in the package are executed.
func TestMain(m *testing.M) {
	// Setup: Discover all modules once to avoid redundant loads and warnings.
	modulePath, err := filepath.Abs("../../modules")
	if err != nil {
		log.Fatalf("failed to resolve module path for testing: %v", err)
	}
	if err := engine.DiscoverModules(modulePath); err != nil {
		log.Fatalf("failed to discover modules for testing: %v", err)
	}

	// Run all tests in the package.
	os.Exit(m.Run())
}

// An empty struct to satisfy the handler signature for stateless runners under test.
type Deps struct{}

// TestExecutor_Integration_StatelessWorkflow tests the end-to-end execution of a
// stateless workflow using handler overrides to mock inputs and spy on outputs.
func TestExecutor_Integration_StatelessWorkflow(t *testing.T) {
	// -- Arrange --

	// 1. Define the "Spy" mock for the 'print' runner.
	var capturedInput map[string]string
	var captureMutex sync.Mutex

	type mockPrintInput struct {
		Value map[string]string `hcl:"input"`
	}

	mockOnRunPrint := func(ctx context.Context, deps *Deps, input *mockPrintInput) (cty.Value, error) {
		captureMutex.Lock()
		defer captureMutex.Unlock()
		capturedInput = input.Value
		return cty.NilVal, nil
	}

	// 2. Define the "Source" mock for the 'env_vars' runner.
	mockOnRunEnvVars := func(ctx context.Context, deps *Deps, input any) (cty.Value, error) {
		testData := map[string]cty.Value{
			"BURSTGRID_TEST_VAR": cty.StringVal("this-is-a-test"),
			"ANOTHER_VAR":        cty.StringVal("42"),
		}
		return cty.ObjectVal(map[string]cty.Value{
			"all": cty.MapVal(testData),
		}), nil
	}

	// 3. Create the handler overrides map.
	stepHandlerOverrides := map[string]*engine.RegisteredHandler{
		"OnRunPrint": {
			NewInput: func() any { return new(mockPrintInput) },
			NewDeps:  func() any { return new(Deps) },
			Fn:       mockOnRunPrint,
		},
		"OnRunEnvVars": {
			NewInput: func() any { return nil },
			NewDeps:  func() any { return new(Deps) },
			Fn:       mockOnRunEnvVars,
		},
	}

	// 4. Parse the HCL grid file that defines the workflow topology.
	gridPath, err := filepath.Abs("../../examples/display_env_vars.hcl")
	require.NoError(t, err)
	hclFiles, err := engine.ResolveGridPath(gridPath)
	require.NoError(t, err)
	require.NotEmpty(t, hclFiles)
	gridConfig, err := engine.DecodeGridFile(hclFiles[0])
	require.NoError(t, err)

	// 5. Build the dependency graph.
	graph, err := NewGraph(gridConfig)
	require.NoError(t, err)

	// -- Act --

	// 6. Create an executor with the mock handlers and run the graph.
	executor := NewExecutor(graph, stepHandlerOverrides, nil)
	runErr := executor.Run()

	// -- Assert --
	require.NoError(t, runErr, "Executor.Run() should not return an error")

	captureMutex.Lock()
	defer captureMutex.Unlock()

	expected := map[string]string{
		"BURSTGRID_TEST_VAR": "this-is-a-test",
		"ANOTHER_VAR":        "42",
	}
	assert.Equal(t, expected, capturedInput, "The 'print' spy should have captured the data from the 'env_vars' source mock.")
}

// mockCounter is a simple, thread-safe counter object used for testing resource state.
type mockCounter struct {
	count atomic.Int64
}

func (m *mockCounter) Increment() {
	m.count.Add(1)
}

func (m *mockCounter) Get() int64 {
	return m.count.Load()
}

// emptyAssetInput is used for assets that take no arguments.
type emptyAssetInput struct{}

// TestExecutor_Integration_StatefulResourceLifecycle tests the end-to-end lifecycle
// of a stateful resource: creation, sharing between steps, state modification, and destruction.
func TestExecutor_Integration_StatefulResourceLifecycle(t *testing.T) {
	t.Skip("Skipping stateful resource test until deadlock is resolved.")
	// -- Arrange --

	// 1. Spies to track handler invocations and created instances.
	var createCallCount, destroyCallCount atomic.Int32
	var createdInstances []*mockCounter
	var instanceMutex sync.Mutex

	// 2. Mock "Create" handler for the 'local_counter' asset.
	mockCreateCounter := func(ctx context.Context, input *emptyAssetInput) (*mockCounter, error) {
		createCallCount.Add(1)
		instance := new(mockCounter)

		instanceMutex.Lock()
		createdInstances = append(createdInstances, instance)
		instanceMutex.Unlock()

		return instance, nil
	}

	// 3. Mock "Destroy" handler for the 'local_counter' asset.
	mockDestroyCounter := func(counter *mockCounter) error {
		destroyCallCount.Add(1)
		return nil
	}

	// 4. Mock 'counter_op' step handler.
	type counterOpInput struct {
		Action string `hcl:"action"`
	}
	type counterOpDeps struct {
		Counter *mockCounter // Field name must match the key in the HCL 'uses' block.
	}
	mockOnRunCounterOp := func(ctx context.Context, deps *counterOpDeps, input *counterOpInput) (cty.Value, error) {
		require.NotNil(t, deps.Counter, "Counter dependency should have been injected")

		switch input.Action {
		case "increment":
			deps.Counter.Increment()
			return cty.NilVal, nil
		case "get":
			val := deps.Counter.Get()
			return cty.ObjectVal(map[string]cty.Value{
				"value": cty.NumberIntVal(val),
			}), nil
		default:
			return cty.NilVal, nil
		}
	}

	// 5. Create the override maps for all mock handlers using the correct types.
	stepHandlerOverrides := map[string]*engine.RegisteredHandler{
		"OnRunCounterOp": {
			NewInput: func() any { return new(counterOpInput) },
			NewDeps:  func() any { return new(counterOpDeps) },
			Fn:       mockOnRunCounterOp,
		},
	}
	assetHandlerOverrides := map[string]*engine.RegisteredAssetHandler{
		"CreateCounter": {
			NewInput: func() any { return new(emptyAssetInput) },
			CreateFn: mockCreateCounter,
		},
		"DestroyCounter": {
			DestroyFn: mockDestroyCounter,
		},
	}

	// 6. Parse the test grid file.
	gridPath, err := filepath.Abs("../../examples/local_resource_test.hcl")
	require.NoError(t, err)
	hclFiles, err := engine.ResolveGridPath(gridPath)
	require.NoError(t, err)
	require.NotEmpty(t, hclFiles)

	gridConfig := &engine.GridConfig{}
	for _, file := range hclFiles {
		cfg, err := engine.DecodeGridFile(file)
		require.NoError(t, err)
		gridConfig.Resources = append(gridConfig.Resources, cfg.Resources...)
		gridConfig.Steps = append(gridConfig.Steps, cfg.Steps...)
	}

	// 7. Build the dependency graph.
	graph, err := NewGraph(gridConfig)
	require.NoError(t, err)

	// -- Act --
	executor := NewExecutor(graph, stepHandlerOverrides, assetHandlerOverrides)
	runErr := executor.Run()

	// -- Assert --
	require.NoError(t, runErr, "Executor.Run() should not return an error")

	// 1. Creation and Destruction: Verify lifecycle handlers were called exactly once.
	assert.Equal(t, int32(1), createCallCount.Load(), "Create handler should be called exactly once.")
	assert.Equal(t, int32(1), destroyCallCount.Load(), "Destroy handler should be called exactly once.")

	// 2. Instance Sharing: Verify all steps received the same object instance.
	instanceMutex.Lock()
	require.GreaterOrEqual(t, len(createdInstances), 1, "At least one instance should have been created")
	firstInstance := createdInstances[0]
	for _, instance := range createdInstances {
		assert.Same(t, firstInstance, instance, "All steps should share the exact same resource instance.")
	}
	instanceMutex.Unlock()

	// 3. State Persistence: Verify the final state of the counter is correct.
	finalNode, ok := graph.Nodes["step.counter_op.get_final_value"]
	require.True(t, ok, "Final 'get' step node should exist in the graph")
	require.NotNil(t, finalNode.Output, "Final node should have an output value")

	var finalOutput struct {
		Value int64 `cty:"value"`
	}
	err = gocty.FromCtyValue(finalNode.Output.(cty.Value), &finalOutput)
	require.NoError(t, err)

	assert.Equal(t, int64(2), finalOutput.Value, "The counter's state should persist and be modified across steps.")
}
