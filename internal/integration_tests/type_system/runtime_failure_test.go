package type_system_test

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/testutil"
)

// --- Structs and Module for Runtime Failure Test ---

// ConfigObject matches the manifest's object({ timeout = number, ... }) definition.
type ConfigObject struct {
	Timeout int  `cty:"timeout"`
	Enabled bool `cty:"enabled"`
}

// TypoInput is the top-level input struct for the Go handler.
type TypoInput struct {
	Config ConfigObject `bggo:"config"`
}

// TestErrorHandling_ObjectAttributeTypeMismatch_FailsRun validates that the
// run fails if a user provides a value of the wrong type for an object attribute.
func TestErrorHandling_ObjectAttributeTypeMismatch_FailsRun(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	// MANIFEST: Defines a 'config' input with a 'timeout' attribute of type 'number'.
	manifestHCL := `
		runner "object_typo_runner" {
			lifecycle { on_run = "OnRunObjectTypo" }
			input "config" {
				type = object({
					timeout = number
					enabled = bool
				})
			}
		}
	`

	// GRID: Intentionally provides a STRING for the 'timeout' attribute, which
	// should cause a runtime failure during argument decoding.
	gridHCL := `
		step "object_typo_runner" "test" {
			arguments {
				config = {
					timeout = "this is not a number"
					enabled = true
				}
			}
		}
	`

	// The Go module is structurally correct according to the manifest.
	// The error is purely from the user's input in gridHCL.
	mockModule := &testutil.SimpleModule{
		RunnerName: "OnRunObjectTypo",
		Runner: &registry.RegisteredRunner{
			NewInput:  func() any { return new(TypoInput) },
			InputType: reflect.TypeOf(TypoInput{}),
			NewDeps:   func() any { return new(struct{}) },
			Fn: func(_ context.Context, _, _ any) (any, error) {
				// This handler should never be called because decoding will fail first.
				t.Error("the runner handler was executed, but it should have failed on input decoding")
				return nil, nil
			},
		},
	}

	files := map[string]string{
		filepath.Join("modules", "runtime_fail", "manifest.hcl"): manifestHCL,
		"main.hcl": gridHCL,
	}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.Error(t, result.Err, "app.Run() should have failed due to a type conversion error, but it succeeded")

	errStr := result.Err.Error()
	require.Contains(t, errStr, "failed to decode argument 'config'", "Error message should specify the top-level argument that failed")
	require.Contains(t, errStr, "in attribute 'timeout'", "Error message should specify the exact object attribute that failed")
	require.Contains(t, errStr, "a number is required", "Error message should contain the underlying cty conversion error")
}
