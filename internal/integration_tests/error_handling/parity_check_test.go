package integration_tests

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/testutil"
)

type mockParityCheckModule struct{}

func (m *mockParityCheckModule) Register(r *registry.Registry) {
	type runnerInput struct {
		GoOnlyField string `bggo:"go_only_field"`
	}
	r.RegisterRunner("OnRunMismatched", &registry.RegisteredRunner{
		NewInput:  func() any { return new(runnerInput) },
		InputType: reflect.TypeOf(runnerInput{}),
		NewDeps:   func() any { return new(struct{}) },
		Fn:        func() {},
	})
}

// TestStartupValidation_ManifestImplementationMismatch_Fails validates that the app
// panics on startup if a manifest and Go struct are out of sync.
func TestStartupValidation_ManifestImplementationMismatch_Fails(t *testing.T) {
	t.Parallel()
	// --- Arrange ---
	mismatchedManifest := `
		runner "mismatched_runner" {
			lifecycle {
				on_run = "OnRunMismatched"
			}
			input "hcl_only_field" {
				type = string
			}
		}
	`
	files := map[string]string{
		"modules/mismatched_runner/manifest.hcl": mismatchedManifest,
	}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, &mockParityCheckModule{})

	// --- Assert ---
	require.Error(t, result.Err, "app.New() should have panicked, but it did not")
	errStr := result.Err.Error()

	expectedGoError := "Go struct has field for input 'go_only_field' which is not declared in manifest"
	require.True(t, strings.Contains(errStr, expectedGoError))

	expectedHclError := "manifest declares input 'hcl_only_field' which is not found in Go struct"
	require.True(t, strings.Contains(errStr, expectedHclError))
}
