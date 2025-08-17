// internal/testutil/runner_parser_harness.go
package testutil

import (
	"context"
	"reflect"
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/handlers"
	"github.com/specialistvlad/burstgridgo/internal/model"
	"github.com/stretchr/testify/require"
)

// RunRunnerParsingTest provides a standardized harness for testing the parsing of
// a runner manifest. It encapsulates the boilerplate of setting up file maps,
// registering a mock handler, and running the application loader.
// It returns the parsed runner and any error encountered during loading.
func RunRunnerParsingTest(t *testing.T, runnerFullHCL string) (*model.Runner, error) {
	t.Helper()

	const handlerName = "OnRunTest"
	const runnerType = "test" // We assume the HCL block uses "test" as the runner type label.

	files := map[string]string{
		"modules/test/manifest.hcl": runnerFullHCL,
	}

	// Create a complete, valid mock module. This ensures that the harness
	// can properly validate the runner, preventing unrelated type errors.
	mockModule := &SimpleModule{
		RunnerName: handlerName,
		Runner: &handlers.RegisteredHandler{
			Input:     func() any { return new(struct{}) },
			InputType: reflect.TypeOf(struct{}{}),
			Deps:      func() any { return new(struct{}) },
			Fn: func(ctx context.Context, deps any, input any) (any, error) {
				// This is a no-op as we are only testing the parsing phase.
				return nil, nil
			},
		},
	}

	// 1. Create a new Handlers store.
	handlerStorage := handlers.New()
	// 2. Register the mock handler with the store.
	handlerStorage.RegisterHandler(mockModule.RunnerName, mockModule.Runner)
	// 3. Pass the correctly typed handler storage to the harness.
	result := RunIntegrationTest(t, files, handlerStorage)

	if result.Err != nil {
		return nil, result.Err
	}

	// If loading succeeded, find the parsed runner in the registry.
	require.NotNil(t, result.App, "App should not be nil on successful load")
	registry := result.App.Registry()
	require.NotNil(t, registry, "Registry should not be nil")

	var actualRunner *model.Runner
	for _, r := range registry.Runners() {
		if r.Type == runnerType {
			actualRunner = r
			break
		}
	}

	require.NotNil(t, actualRunner, "Parsed runner '%s' not found in registry", runnerType)
	return actualRunner, nil
}
