package type_system_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/registry"
	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestStartupValidation_ObjectManifestMismatch_Fails validates that the app
// fails to start if a manifest's object type is incompatible with the Go struct's type.
func TestStartupValidation_ObjectManifestMismatch_Fails(t *testing.T) {
	t.Parallel()

	// --- Arrange ---

	// MANIFEST: Defines an object with a boolean 'enabled' field.
	manifestHCL := `
		runner "mismatch_runner" {
			lifecycle { on_run = "OnRunMismatch" }
			input "config" {
				type = object({
					enabled = bool
				})
			}
		}
	`

	// GO STRUCT: Defines the 'enabled' field as a string, creating a mismatch.
	type MismatchedConfig struct {
		Enabled string `cty:"enabled"`
	}
	type MismatchInput struct {
		Config MismatchedConfig `bggo:"config"`
	}

	mockModule := &testutil.SimpleModule{
		RunnerName: "OnRunMismatch",
		Runner: &registry.RegisteredRunner{
			NewInput:  func() any { return new(MismatchInput) },
			InputType: reflect.TypeOf(MismatchInput{}),
			Fn:        func() {}, // The function itself never runs
		},
	}

	files := map[string]string{
		filepath.Join("modules", "mismatch", "manifest.hcl"): manifestHCL,
	}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.Error(t, result.Err, "app.New() should have panicked due to a manifest/Go mismatch, but it did not")
	errStr := result.Err.Error()

	require.Contains(t, errStr, "registry validation failed", "Error should indicate registry validation failure")

	// This is the specific error we want our new validation logic to produce.
	expectedErrorSubstring := "input 'config': attribute 'enabled' type mismatch: manifest requires 'bool', but Go struct field 'Enabled' provides 'string'"
	require.Contains(t, errStr, expectedErrorSubstring, "The error message did not clearly state the attribute mismatch")
}
