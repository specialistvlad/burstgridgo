package dag

import (
	"context"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
)

// TestMain runs setup before any tests in the package are executed.
func TestMain(m *testing.M) {
	// Configure slog for test-time debugging.
	// Run with `BGGO_TEST_LOG_LEVEL=debug go test -v ./...` to enable.
	logLevel := slog.LevelInfo
	if os.Getenv("BGGO_TEST_LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	slog.SetDefault(slog.New(handler))

	// Setup: Discover all modules once to avoid redundant loads and warnings.
	slog.Info("Setting up test suite: discovering modules...")
	modulePath, err := filepath.Abs("../../modules")
	if err != nil {
		log.Fatalf("failed to resolve module path for testing: %v", err)
	}
	if err := engine.DiscoverModules(modulePath); err != nil {
		log.Fatalf("failed to discover modules for testing: %v", err)
	}
	slog.Info("Test suite setup complete.")

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
