package integration_tests

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/registry"
	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
)

type mockManifestValidationModule struct{}

func (m *mockManifestValidationModule) Register(r *registry.Registry) {
	type runnerInput struct {
		Name string `bggo:"name"`
	}
	r.RegisterRunner("OnRunManifest", &registry.RegisteredRunner{
		NewInput:  func() any { return new(runnerInput) },
		InputType: reflect.TypeOf(runnerInput{}),
		NewDeps:   func() any { return new(struct{}) },
		Fn:        func(context.Context, any, any) (any, error) { return nil, nil },
	})
}

// TestErrorHandling_ReferenceToUndeclaredOutput_FailsRun validates that the app
// fails when a step references an output that is not declared in the manifest.
func TestErrorHandling_ReferenceToUndeclaredOutput_FailsRun(t *testing.T) {
	t.Parallel()
	// --- Arrange ---
	manifestHCL := `
		runner "manifest_runner" {
			lifecycle {
				on_run = "OnRunManifest"
			}
			input "name" {
				type = string
			}
		}
	`
	gridHCL := `
		step "manifest_runner" "A" {
			arguments {
				name = "step A"
			}
		}

		step "manifest_runner" "B" {
			arguments {
				# This references an output that is not declared in the manifest.
				name = step.manifest_runner.A.output.undeclared_value
			}
		}
	`
	files := map[string]string{
		"modules/manifest_runner/manifest.hcl": manifestHCL,
		"main.hcl":                             gridHCL,
	}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, &mockManifestValidationModule{})

	// --- Assert ---
	require.Error(t, result.Err, "run should have failed for a reference to an undeclared output")
	require.True(t, strings.Contains(result.Err.Error(), "undeclared output"), "error message mismatch")
}
