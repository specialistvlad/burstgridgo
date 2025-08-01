package integration_tests

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/testutil"
)

// TestErrorHandling_RequiredArgumentMissing_FailsRun validates that the application
// fails when a step omits a required argument defined in the manifest.
func TestErrorHandling_RequiredArgumentMissing_FailsRun(t *testing.T) {
	t.Parallel()
	// --- Arrange ---
	manifestHCL := `
		runner "required_arg_runner" {
			lifecycle {
				on_run = "OnRunRequiredArg"
			}
			input "name" {
				type = string
				# 'optional = false' is the default, so this is required.
			}
		}
	`
	// This grid HCL is invalid because the 'arguments' block is empty.
	gridHCL := `
		step "required_arg_runner" "A" {
			arguments {}
		}
	`
	files := map[string]string{
		"modules/required_arg_runner/manifest.hcl": manifestHCL,
		"main.hcl": gridHCL,
	}

	// Define the input struct and the mock module for the test.
	type runnerInput struct {
		Name string `bggo:"name"`
	}
	mockModule := &testutil.SimpleModule{
		RunnerName: "OnRunRequiredArg",
		Runner: &registry.RegisteredRunner{
			NewInput:  func() any { return new(runnerInput) },
			InputType: reflect.TypeOf(runnerInput{}),
			NewDeps:   func() any { return new(struct{}) },
			Fn:        func(context.Context, any, any) (any, error) { return nil, nil },
		},
	}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.Error(t, result.Err, "app.Run() should have returned an error for a missing required argument")

	expectedErrorSubstring := `missing required argument "name"`
	require.True(t, strings.Contains(result.Err.Error(), expectedErrorSubstring), "error message mismatch, got: %v", result.Err)
}
