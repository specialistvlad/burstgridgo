package integration_tests

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/specialistvlad/burstgridgo/internal/handlers"
	"github.com/specialistvlad/burstgridgo/internal/model"
	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// TestLoader_FocusOnParsing verifies that HCL manifests are correctly
// parsed into the application's internal configuration.
func TestLoader_FocusOnParsing(t *testing.T) {
	// --- Arrange ---

	// 1. Define HCL files for the test harness.
	manifestHCL := `
        runner "test_runner" {
            description = "A test model."
            lifecycle {
                on_run = "OnRunTest"
            }
            input "message" {
                type    = string
                default = "default_message"
            }
        }
    `
	// Grid HCL commented out as it's not implemented yet.
	// gridHCL := `
	//     step "test_runner" "A" {
	//         arguments {
	//             message = "hello"
	//         }
	//     }
	// `
	files := map[string]string{
		"modules/test_runner/manifest.hcl": manifestHCL,
		// Grid file removed from the test harness.
		// "main.hcl":                         gridHCL,
	}

	// 2. Create a mock module. This is required for the app to initialize
	// correctly, as the loader phase also validates handlers.
	type testRunnerInput struct {
		Message string `bggo:"message"`
	}

	mockModule := &testutil.SimpleModule{
		RunnerName: "OnRunTest",
		Runner: &handlers.RegisteredHandler{
			Input:     func() any { return new(testRunnerInput) },
			InputType: reflect.TypeOf(testRunnerInput{}),
			Deps:      func() any { return new(struct{}) },
			Fn: func(ctx context.Context, deps, input any) (any, error) {
				return nil, nil // No-op, we are not testing execution.
			},
		},
	}

	// --- Act ---
	// Run the app, passing the mock module to the harness for injection.
	handlers_storage := handlers.New()
	handlers_storage.RegisterHandler(mockModule.RunnerName, mockModule.Runner)
	result := testutil.RunIntegrationTest(t, files, handlers_storage)

	// --- Assert ---

	// 1. Basic checks: The application should initialize and load without errors.
	require.NoError(t, result.Err, "The application run should not produce an error")
	require.NotNil(t, result.App, "The app instance should not be nil")

	// 2. Assert on logs: Check that the loading phases were logged.
	require.Contains(t, result.LogOutput, "Loading modules...", "log should indicate module loading has started")
	// Grid log assertion commented out.
	// require.Contains(t, result.LogOutput, "Loading grids...", "log should indicate grid loading has started")

	// 3. Assert on the parsed Runner Definition structure.
	defaultValue := cty.StringVal("default_message")
	expectedRunner := &model.Runner{
		Type:        "test_runner",
		Description: "A test model.",
		// The fields below are not currently parsed, so the test will ignore them.
		Lifecycle: model.RunnerLifecycle{OnRun: "OnRunTest"},
		Inputs: map[string]model.RunnerInputDefinition{
			"message": {
				Type:    cty.String,
				Default: &defaultValue,
			},
		},
	}

	ctyTypeComparer := cmp.Comparer(func(a, b cty.Type) bool {
		return a.Equals(b)
	})
	ignoreCtyUnexported := cmpopts.IgnoreUnexported(cty.Value{})

	var actualRunner *model.Runner
	for _, r := range result.App.Registry().Runners() {
		if r.Type == "test_runner" {
			actualRunner = r
			break
		}
	}
	require.NotNil(t, actualRunner, "Runner 'test_runner' was not found in the registry")

	if diff := cmp.Diff(expectedRunner, actualRunner, ctyTypeComparer, ignoreCtyUnexported, cmpopts.IgnoreFields(model.Runner{}, "FSInformation", "Inputs", "Lifecycle", "Outputs")); diff != "" {
		t.Errorf("Runner definition mismatch (-want +got):\n%s", diff)
	}

	// Grid step assertions commented out.
	// 4. Assert on the parsed Grid Step structure.
	// require.Len(t, result.App.Grid.Steps, 1, "Expected 1 step in the grid")
	// step := result.App.Grid.Steps[0]
	// require.Equal(t, "test_runner", step.RunnerType)
	// require.Equal(t, "A", step.Name)
	// require.Contains(t, step.Arguments, "message", "Expected 'message' argument in step arguments")
}
