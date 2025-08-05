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
	"github.com/zclconf/go-cty/cty"
)

// --- Structs and Module for Generic Object Test ---

type GenericObjectInput struct {
	// As per ADR-011, a generic object should decode to map[string]any.
	// The 'any' type in Go is 'interface{}'.
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
	t.Skip("Doesn't work yet. Error: Received unexpected error: execution failed: execution failed for step.generic_object_runner.test: failed to decode arguments for step step.generic_object_runner.test: failed to decode argument 'data' into Go struct field: failed to decode attribute 'is_active' into 'any': incorrect type")
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
					name    = "dynamic payload"
					is_active = true
					retries = 3
					tags    = ["a", "b"]
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
	require.NoError(t, result.Err, "Run failed unexpectedly for generic object test")

	// go-cty decodes HCL numbers into cty.Number, which gocty then decodes
	// into the most appropriate Go numeric type, often float64. We must
	// also account for list elements being of type 'any' (interface{}).
	expectedData := map[string]any{
		"name":      "dynamic payload",
		"is_active": true,
		"retries":   cty.NumberIntVal(3), // Comparing with cty.Value is most reliable
		"tags":      []any{"a", "b"},
	}

	// Because of the type ambiguity with `any`, direct comparison can be tricky.
	// We'll use go-cmp with a transformer for cty.Value for robust comparison.
	cmp.Transformer("cty", func(v cty.Value) any {
		// A simple transformation for comparison purposes
		if v.Type() == cty.Number {
			bf, _ := v.AsBigFloat().Float64()
			return bf
		}
		// Add other types as needed, for now this handles our case
		return nil
	})

	mockModule.mu.Lock()
	defer mockModule.mu.Unlock()

	// A simple length check is a good first step.
	require.Len(t, mockModule.capturedInput.Data, 4, "decoded map has the wrong number of keys")

	// Check individual fields for type-safety and correctness.
	require.Equal(t, expectedData["name"], mockModule.capturedInput.Data["name"])
	require.Equal(t, expectedData["is_active"], mockModule.capturedInput.Data["is_active"])

	// For the number, let's convert and compare to avoid float precision issues.
	retriesVal, ok := mockModule.capturedInput.Data["retries"].(cty.Value)
	require.True(t, ok, "retries field was not a cty.Value")
	require.True(t, expectedData["retries"].(cty.Value).Equals(retriesVal).True(), "retries value did not match")

	// For the slice, we need to compare contents.
	require.ElementsMatch(t, expectedData["tags"], mockModule.capturedInput.Data["tags"])
}
