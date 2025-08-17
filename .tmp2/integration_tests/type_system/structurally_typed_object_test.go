package type_system_test

import (
	"context"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/specialistvlad/burstgridgo/internal/registry"
	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
)

// --- Structs and Module for Object Test ---

// UserDetails is a nested struct that uses 'cty' tags as per ADR-011.
type UserDetails struct {
	Name string `cty:"name"`
	Age  int    `cty:"age"`
}

// ObjectRunnerInput is the top-level input struct using the 'bggo' tag.
type ObjectRunnerInput struct {
	UserDetails UserDetails `bggo:"user_details"`
}

// objectRunnerModule is the mock module for this test.
type objectRunnerModule struct {
	capturedInput ObjectRunnerInput
	mu            sync.Mutex
}

func (m *objectRunnerModule) Register(r *registry.Registry) {
	r.RegisterRunner("OnRunObjectRunner", &registry.RegisteredRunner{
		NewInput:  func() any { return new(ObjectRunnerInput) },
		InputType: reflect.TypeOf(ObjectRunnerInput{}),
		NewDeps:   func() any { return new(struct{}) },
		Fn: func(_ context.Context, _ any, inputRaw any) (any, error) {
			m.mu.Lock()
			m.capturedInput = *inputRaw.(*ObjectRunnerInput)
			m.mu.Unlock()
			return nil, nil
		},
	})
}

// TestCoreExecution_StructurallyTypedObject_Success validates that a valid,
// structurally-typed object can be parsed from HCL and decoded into the
// corresponding Go structs.
func TestCoreExecution_StructurallyTypedObject_Success(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	manifestHCL := `
		runner "object_runner" {
			lifecycle {
				on_run = "OnRunObjectRunner"
			}
			input "user_details" {
				description = "A structured object representing a user."
				type = object({
					name = string
					age  = number
				})
			}
		}
	`

	gridHCL := `
		step "object_runner" "test" {
			arguments {
				user_details = {
					name = "John Doe"
					age  = 42
				}
			}
		}
	`

	files := map[string]string{
		filepath.Join("modules", "objects", "manifest.hcl"): manifestHCL,
		"main.hcl": gridHCL,
	}

	mockModule := &objectRunnerModule{}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.NoError(t, result.Err, "Expected the run to succeed, but it failed. Full logs:\n%s", result.LogOutput)

	expectedInput := ObjectRunnerInput{
		UserDetails: UserDetails{
			Name: "John Doe",
			Age:  42,
		},
	}

	mockModule.mu.Lock()
	defer mockModule.mu.Unlock()
	if diff := cmp.Diff(expectedInput, mockModule.capturedInput); diff != "" {
		t.Errorf("Captured object input mismatch (-want +got):\n%s", diff)
	}
}
