package dag

import (
	"context"
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
	// -- Arrange --

	// 1. Spies to track handler invocations and received instances.
	var createCallCount, destroyCallCount atomic.Int32
	var receivedInstances []*mockCounter
	var instanceMutex sync.Mutex

	// 2. Mock "Create" handler for the 'local_counter' asset.
	mockCreateCounter := func(ctx context.Context, input *emptyAssetInput) (*mockCounter, error) {
		newCount := createCallCount.Add(1)
		t.Logf("[CREATE SPY] Creating resource instance. New call count: %d", newCount)
		instance := new(mockCounter)
		return instance, nil
	}

	// 3. Mock "Destroy" handler for the 'local_counter' asset.
	mockDestroyCounter := func(counter *mockCounter) error {
		newCount := destroyCallCount.Add(1)
		t.Logf("[DESTROY SPY] Destroying resource instance. New call count: %d", newCount)
		return nil
	}

	// 4. Mock 'counter_op' step handler.
	type counterOpInput struct {
		Action string `hcl:"action"`
	}
	type counterOpDeps struct {
		Counter *mockCounter `hcl:"counter"` // This tag makes the injection explicit and robust.
	}
	mockOnRunCounterOp := func(ctx context.Context, deps *counterOpDeps, input *counterOpInput) (cty.Value, error) {
		t.Logf("[STEP SPY] Running counter_op with action '%s'", input.Action)
		require.NotNil(t, deps.Counter, "Counter dependency should have been injected")

		instanceMutex.Lock()
		receivedInstances = append(receivedInstances, deps.Counter)
		instanceMutex.Unlock()

		switch input.Action {
		case "increment":
			deps.Counter.Increment()
			return cty.NilVal, nil
		case "get":
			val := deps.Counter.Get()
			t.Logf("[STEP SPY] 'get' action returning value: %d", val)
			return cty.ObjectVal(map[string]cty.Value{
				"value": cty.NumberIntVal(val),
			}), nil
		default:
			return cty.NilVal, nil
		}
	}

	// 5. Create the override maps for all mock handlers.
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

	gridConfig, err := engine.DecodeGridFile(hclFiles[0])
	require.NoError(t, err)

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
	require.Len(t, receivedInstances, 3, "Exactly 3 steps should have received the counter instance.")
	firstInstance := receivedInstances[0]
	for i, instance := range receivedInstances {
		assert.Same(t, firstInstance, instance, "All steps should share the exact same resource instance. Mismatch at index %d.", i)
	}
	instanceMutex.Unlock()

	// 3. State Persistence: Verify the final state of the counter is correct.
	finalNode, ok := graph.Nodes["step.counter_op.get_final_value"]
	require.True(t, ok, "Final 'get' step node should exist in the graph")
	require.NotNil(t, finalNode.Output, "Final node should have an output value")

	var finalOutput struct {
		Value int64 `cty:"value"`
	}
	outputValue, ok := finalNode.Output.(cty.Value)
	require.True(t, ok, "Node output should be of type cty.Value")

	err = gocty.FromCtyValue(outputValue, &finalOutput)
	require.NoError(t, err)

	assert.Equal(t, int64(2), finalOutput.Value, "The counter's state should persist and be 2 after two increments.")
}
