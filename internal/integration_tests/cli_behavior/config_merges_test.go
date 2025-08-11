package integration_tests

import (
	"context"
	"reflect"
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/registry"
	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestCLI_MergesHCL_FromDirectoryPath validates that the loader correctly
// discovers and merges all HCL files from a given directory path.
func TestCLI_MergesHCL_FromDirectoryPath(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	manifestHCL := `
		runner "print" {
			lifecycle {
				on_run = "OnRunPrint"
			}
		}
	`
	hclFileA := `
		step "print" "step_A" {
			arguments {}
		}
	`
	hclFileB := `
		step "print" "step_B" {
			arguments {}
		}
	`
	// The harness will create these in the same directory structure.
	files := map[string]string{
		"modules/print/manifest.hcl": manifestHCL,
		"grids/a.hcl":                hclFileA,
		"grids/b.hcl":                hclFileB,
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
	// The harness configures the app to load from the root temporary
	// directory, discovering the module manifest and all grid files.
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.NoError(t, result.Err, "app.Run() returned an unexpected error")

	// Check that both steps, from both files, were executed using the resilient helper.
	testutil.AssertStepRan(t, result, "print", "step_A")
	testutil.AssertStepRan(t, result, "print", "step_B")
}
