package integration_tests

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vk/burstgridgo/internal/testutil"
)

func TestErrorHandling_AmbiguousImplicitDependency(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	// The runner manifests are minimal, as we only need them to pass
	// registry validation. The logic we are testing is in the grid HCL.
	sourceManifest := `
        runner "source" {
          lifecycle { on_run = "NoOp" }
		  // We must declare an output for the grid HCL to be valid.
          output "output" { type = any }
        }
    `
	consumerManifest := `
        runner "consumer" {
          lifecycle { on_run = "NoOp" }
		  // The "NoOp" runner takes no inputs, so we declare none here.
        }
    `
	// The grid config contains the ambiguous reference that should cause an error.
	gridHCL := `
        step "source" "many" {
          count = 3
        }

        step "consumer" "one" {
          // This reference to "step.source.many" is ambiguous because "many"
          // has multiple instances. It should fail validation.
          arguments {
            input_val = step.source.many.output
          }
        }
    `
	files := map[string]string{
		"modules/source.hcl":   sourceManifest,
		"modules/consumer.hcl": consumerManifest,
		"main.hcl":             gridHCL,
	}

	// The NoOpModule provides a valid "NoOp" handler for the manifests.
	mockModule := &testutil.NoOpModule{}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	// We expect an error to have occurred during the DAG linking phase.
	require.Error(t, result.Err, "expected the run to fail with a validation error")

	// Check for our specific error message to ensure the right validation fired.
	expectedErrorSubstring := "ambiguous implicit dependency"
	require.Contains(t, result.Err.Error(), expectedErrorSubstring, "error message should indicate an ambiguous dependency")
}

func TestErrorHandling_AmbiguousExplicitDependency(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	// Minimal runner manifests are sufficient, as the error should occur
	// during graph linking, well before any runner execution.
	sourceManifest := `
        runner "source" {
          lifecycle { on_run = "NoOp" }
        }
    `
	consumerManifest := `
        runner "consumer" {
          lifecycle { on_run = "NoOp" }
        }
    `
	// The grid config contains the ambiguous explicit dependency.
	gridHCL := `
        step "source" "many" {
          count = 3
        }

        step "consumer" "one" {
          // This explicit "depends_on" reference is ambiguous because "many"
          // has multiple instances. It should fail validation.
          depends_on = ["source.many"]
        }
    `
	files := map[string]string{
		"modules/source.hcl":   sourceManifest,
		"modules/consumer.hcl": consumerManifest,
		"main.hcl":             gridHCL,
	}

	mockModule := &testutil.NoOpModule{}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.Error(t, result.Err, "expected the run to fail with a validation error")

	// Check for our specific error message from the explicit dependency linker.
	expectedErrorSubstring := "ambiguous dependency"
	require.Contains(t, result.Err.Error(), expectedErrorSubstring, "error message should indicate an ambiguous dependency")
}
