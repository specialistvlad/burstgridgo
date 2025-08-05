package type_system_test

import (
	"context"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/testutil"
)

// --- Structs and Module for Generic Object Test ---

type GenericObjectInput struct {
	// As per ADR-011, a generic object should decode to map[string]any.
	Data map[string]any `bggo:"data"`
}

type genericObjectModule struct {
	capturedInput GenericObjectInput
	mu            sync.Mutex
}

func (m *genericObjectModule) Register(r *registry.Registry) {
	r.RegisterRunner("OnRunGenericObject", &registry.RegisteredRunner{
		NewInput:  func() any { return new(GenericObjectInput) },
		InputType: reflect.TypeOf(GenericObjectInput{}),
		NewDeps:   func() any { return new(struct{}) },
		Fn: func(_ context.Context, _ any, inputRaw any) (any, error) {
			m.mu.Lock()
			m.capturedInput = *inputRaw.(*GenericObjectInput)
			m.mu.Unlock()
			return nil, nil
		},
	})
}

// TestCoreExecution_GenericObject_Success validates that an input with type
// object({}) correctly decodes into a map[string]any in Go.
func TestCoreExecution_GenericObject_Success(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	// MANIFEST: Defines an input as a generic object.
	manifestHCL := `
		runner "generic_object_runner" {
			lifecycle { on_run = "OnRunGenericObject" }
			input "data" {
				type = object({})
			}
		}
	`

	// GRID: Provides an object with heterogeneous data types.
	gridHCL := `
		step "generic_object_runner" "test" {
			arguments {
				data = {
					name      = "dynamic payload"
					is_active = true
					retries   = 3
					tags      = ["a", "b"]
				}
			}
		}
	`

	files := map[string]string{
		filepath.Join("modules", "generic", "manifest.hcl"): manifestHCL,
		"main.hcl": gridHCL,
	}

	mockModule := &genericObjectModule{}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.NoError(t, result.Err, "Run failed unexpectedly for generic object test. Logs:\n%s", result.LogOutput)

	// Our converter now produces native Go types, which is the desired behavior.
	// The test's expectations are updated to reflect this.
	expectedData := map[string]any{
		"name":      "dynamic payload",
		"is_active": true,
		"retries":   float64(3), // Expect a native float64
		"tags":      []any{"a", "b"},
	}

	mockModule.mu.Lock()
	defer mockModule.mu.Unlock()

	// Using cmp.Diff provides the most robust and complete comparison.
	if diff := cmp.Diff(expectedData, mockModule.capturedInput.Data); diff != "" {
		t.Errorf("Captured generic object data mismatch (-want +got):\n%s", diff)
	}
}
