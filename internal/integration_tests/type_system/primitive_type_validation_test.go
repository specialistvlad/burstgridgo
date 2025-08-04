package type_system_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/testutil"
)

// --- Test for Startup Validation (Manifest vs. Go Mismatch) ---

type mockMismatchModule struct{}

func (m *mockMismatchModule) Register(r *registry.Registry) {
	type mismatchInput struct {
		Value string `bggo:"value"` // Go is 'string'
	}
	r.RegisterRunner("OnRunMismatch", &registry.RegisteredRunner{
		NewInput:  func() any { return new(mismatchInput) },
		InputType: reflect.TypeOf(mismatchInput{}),
		NewDeps:   func() any { return new(struct{}) },
		Fn:        func(context.Context, any, any) (any, error) { return nil, nil },
	})
}

// TestStartupValidation_ManifestGoTypeMismatch_Fails validates that the app
// fails to start if a manifest's type is incompatible with the Go struct's type.
func TestStartupValidation_ManifestGoTypeMismatch_Fails(t *testing.T) {
	t.Parallel()
	// --- Arrange ---
	mismatchedManifest := `
		runner "mismatch_runner" {
			lifecycle {
				on_run = "OnRunMismatch"
			}
			input "value" {
				type = number // Manifest wants 'number'
			}
		}
	`
	files := map[string]string{
		"modules/mismatch_runner/manifest.hcl": mismatchedManifest,
	}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, &mockMismatchModule{})

	// --- Assert ---
	require.Error(t, result.Err, "app.New() should have panicked, but it did not")
	errStr := result.Err.Error()
	require.Contains(t, errStr, "application startup panicked", "Error should indicate a panic")
	require.Contains(t, errStr, "registry validation failed", "Error should indicate registry validation failure")
	require.Contains(t, errStr, "type mismatch. Manifest requires 'number' but Go struct field 'Value' provides compatible type 'string'", "Error message is incorrect")
}

// --- Test for Runtime Validation (Invalid User Input) ---

type mockTypeCheckModule struct{}

func (m *mockTypeCheckModule) Register(r *registry.Registry) {
	type typeCheckInput struct {
		Value int `bggo:"value"` // Go is 'int', compatible with 'number'
	}
	r.RegisterRunner("OnRunTypeCheck", &registry.RegisteredRunner{
		NewInput:  func() any { return new(typeCheckInput) },
		InputType: reflect.TypeOf(typeCheckInput{}),
		NewDeps:   func() any { return new(struct{}) },
		Fn:        func(context.Context, any, any) (any, error) { return nil, nil },
	})
}

// TestErrorHandling_TypeMismatch_FailsRun validates that a run fails if a user
// provides a value that cannot be converted to the manifest's declared type.
func TestErrorHandling_TypeMismatch_FailsRun(t *testing.T) {
	t.Parallel()
	// --- Arrange ---
	manifestHCL := `
		runner "type_checker" {
			lifecycle {
				on_run = "OnRunTypeCheck"
			}
			input "value" {
				type = number
			}
		}
	`
	// This grid HCL is invalid because "not-a-number" cannot be converted to a number.
	gridHCL := `
		step "type_checker" "A" {
			arguments {
				value = "not-a-number"
			}
		}
	`
	files := map[string]string{
		"modules/type_checker/manifest.hcl": manifestHCL,
		"main.hcl":                          gridHCL,
	}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, &mockTypeCheckModule{})

	// --- Assert ---
	require.Error(t, result.Err, "app.Run() should have failed due to a type conversion error")
	errStr := result.Err.Error()
	// This is the actual, user-friendly error from the cty library.
	expectedErrorSubstring := "a number is required"
	require.True(t, strings.Contains(errStr, expectedErrorSubstring), "error message mismatch, expected it to contain '%s', but got: %v", expectedErrorSubstring, result.Err)
}
