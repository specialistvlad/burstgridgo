package dag

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
)

// An empty struct to satisfy the handler signature for stateless runners under test.
type Deps struct{}

// TestExecutor_Integration_StatelessWorkflow tests the end-to-end execution of a
// stateless workflow using handler overrides to mock inputs and spy on outputs.
func TestExecutor_Integration_StatelessWorkflow(t *testing.T) {
	// -- Arrange --

	// 1. Define the "Spy" mock for the 'print' runner.
	// This will capture the input it receives so we can assert on it later.
	var capturedInput map[string]string
	var captureMutex sync.Mutex

	// The 'print' runner's Go handler expects an input struct with a 'Value' field.
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
	// This will inject a known, deterministic map, replacing the real os.Environ().
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
	// The keys must match the 'on_run' values in the respective runner manifests.
	overrides := map[string]*engine.RegisteredHandler{
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

	// 4. Discover real module definitions (we need them for schemas).
	// The test assumes it runs from the 'dag' package directory.
	modulePath, err := filepath.Abs("../../modules")
	require.NoError(t, err)
	err = engine.DiscoverModules(modulePath)
	require.NoError(t, err)

	// 5. Parse the HCL grid file that defines the workflow topology.
	gridPath, err := filepath.Abs("../../examples/display_env_vars.hcl")
	require.NoError(t, err)
	hclFiles, err := engine.ResolveGridPath(gridPath)
	require.NoError(t, err)
	require.NotEmpty(t, hclFiles)
	gridConfig, err := engine.DecodeGridFile(hclFiles[0])
	require.NoError(t, err)

	// 6. Build the dependency graph.
	graph, err := NewGraph(gridConfig)
	require.NoError(t, err)

	// -- Act --

	// 7. Create an executor with the mock handlers and run the graph.
	executor := NewExecutor(graph, overrides)
	runErr := executor.Run()

	// -- Assert --

	// 8. Verify the execution succeeded and the "Spy" captured the correct data.
	require.NoError(t, runErr, "Executor.Run() should not return an error")

	captureMutex.Lock()
	defer captureMutex.Unlock()

	expected := map[string]string{
		"BURSTGRID_TEST_VAR": "this-is-a-test",
		"ANOTHER_VAR":        "42",
	}
	assert.Equal(t, expected, capturedInput, "The 'print' spy should have captured the data from the 'env_vars' source mock.")
}
