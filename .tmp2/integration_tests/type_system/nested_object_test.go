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

// --- Structs and Module for Nested Object Test ---

// Metadata is the inner, nested struct. As per ADR-011, it uses 'cty' tags.
type Metadata struct {
	CorrelationID string `cty:"correlation_id"`
	Source        string `cty:"source"`
}

// NestedObjectInput is the top-level input struct. It uses 'bggo' tags.
type NestedObjectInput struct {
	EventName string   `bggo:"event_name"`
	Meta      Metadata `bggo:"meta"`
}

// nestedObjectModule is the mock module for this test.
type nestedObjectModule struct {
	capturedInput NestedObjectInput
	mu            sync.Mutex
}

func (m *nestedObjectModule) Register(r *registry.Registry) {
	r.RegisterRunner("OnRunNestedObject", &registry.RegisteredRunner{
		NewInput:  func() any { return new(NestedObjectInput) },
		InputType: reflect.TypeOf(NestedObjectInput{}),
		NewDeps:   func() any { return new(struct{}) },
		Fn: func(_ context.Context, _ any, inputRaw any) (any, error) {
			m.mu.Lock()
			m.capturedInput = *inputRaw.(*NestedObjectInput)
			m.mu.Unlock()
			return nil, nil
		},
	})
}

// TestCoreExecution_NestedObject_Success validates that a deeply nested
// object defined in HCL can be decoded correctly into nested Go structs.
func TestCoreExecution_NestedObject_Success(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	// MANIFEST: Defines an input 'meta' as a structured object.
	manifestHCL := `
		runner "nested_object_runner" {
			lifecycle {
				on_run = "OnRunNestedObject"
			}
			input "event_name" {
				type = string
			}
			input "meta" {
				type = object({
					correlation_id = string
					source         = string
				})
			}
		}
	`

	// GRID: Provides a nested object for the 'meta' argument.
	gridHCL := `
		step "nested_object_runner" "test" {
			arguments {
				event_name = "user_login_success"
				meta = {
					correlation_id = "req-id-987"
					source         = "web-app"
				}
			}
		}
	`

	files := map[string]string{
		filepath.Join("modules", "nested", "manifest.hcl"): manifestHCL,
		"main.hcl": gridHCL,
	}

	mockModule := &nestedObjectModule{}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.NoError(t, result.Err, "Run failed unexpectedly for nested object test. Logs:\n%s", result.LogOutput)

	expectedInput := NestedObjectInput{
		EventName: "user_login_success",
		Meta: Metadata{
			CorrelationID: "req-id-987",
			Source:        "web-app",
		},
	}

	mockModule.mu.Lock()
	defer mockModule.mu.Unlock()

	if diff := cmp.Diff(expectedInput, mockModule.capturedInput); diff != "" {
		t.Errorf("Captured nested object data mismatch (-want +got):\n%s", diff)
	}
}
