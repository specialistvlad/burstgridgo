package integration_tests

import (
	"context"
	"reflect"
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/registry"
	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestCoreExecution_Count_Static validates that a step with a static `count`
// meta-argument is expanded into N distinct nodes in the graph and all are executed.
func TestCoreExecution_Count_Static(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	manifestHCL := `
		runner "print" {
			lifecycle { on_run = "OnRunPrint" }
		}
	`
	gridHCL := `
		step "print" "A" {
			count = 3
			arguments {}
		}
	`
	files := map[string]string{
		"modules/print/manifest.hcl": manifestHCL,
		"main.hcl":                   gridHCL,
	}

	mockModule := &testutil.SimpleModule{
		RunnerName: "OnRunPrint",
		Runner: &registry.RegisteredRunner{
			NewInput:  func() any { return new(struct{}) },
			InputType: reflect.TypeOf(struct{}{}),
			NewDeps:   func() any { return new(struct{}) },
			Fn:        func(context.Context, any, any) (any, error) { return nil, nil },
		},
	}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.NoError(t, result.Err, "app.Run() returned an unexpected error")

	// Note: We're temporarily using direct `require.Contains` here.
	// In a future step, we can enhance our `testutil.AssertStepRan` helper
	// to be more flexible for these indexed checks.
	require.Contains(t, result.LogOutput, "step=step.print.A[0]", "log for instance [0] not found")
	require.Contains(t, result.LogOutput, "step=step.print.A[1]", "log for instance [1] not found")
	require.Contains(t, result.LogOutput, "step=step.print.A[2]", "log for instance [2] not found")
}
