package integration_tests

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/specialistvlad/burstgridgo/internal/registry"
	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
)

type namedInput struct {
	Name string `bggo:"name"`
}

// TestCoreExecution_CountIndex_IsInjected validates that `count.index` is
// available for interpolation in a step's arguments.
func TestCoreExecution_CountIndex_IsInjected(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	manifestHCL := `
		runner "input_spy" {
			lifecycle { on_run = "OnRunSpy" }
			input "name" {
				type = string
			}
		}
	`
	gridHCL := `
		step "input_spy" "A" {
			count = 2
			arguments {
				name = "instance-${count.index}"
			}
		}
	`
	files := map[string]string{
		"modules/input_spy/manifest.hcl": manifestHCL,
		"main.hcl":                       gridHCL,
	}

	// A thread-safe map to capture the inputs received by each instance.
	var capturedInputs sync.Map

	mockModule := &testutil.SimpleModule{
		RunnerName: "OnRunSpy",
		Runner: &registry.RegisteredRunner{
			NewInput:  func() any { return new(namedInput) },
			InputType: reflect.TypeOf(namedInput{}),
			NewDeps:   func() any { return new(struct{}) },
			Fn: func(ctx context.Context, deps any, input any) (any, error) {
				in := input.(*namedInput)
				// Store the captured input, keyed by the name.
				capturedInputs.Store(in.Name, in)
				return nil, nil
			},
		},
	}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.NoError(t, result.Err, "app.Run() returned an unexpected error")

	// Verify that both instances ran with their correctly interpolated names.
	expectedInputs := map[string]namedInput{
		"instance-0": {Name: "instance-0"},
		"instance-1": {Name: "instance-1"},
	}

	actualInputs := make(map[string]namedInput)
	capturedInputs.Range(func(key, value any) bool {
		actualInputs[key.(string)] = *(value.(*namedInput))
		return true
	})

	if diff := cmp.Diff(expectedInputs, actualInputs); diff != "" {
		t.Errorf("Captured inputs mismatch (-want +got):\n%s", diff)
	}
}
