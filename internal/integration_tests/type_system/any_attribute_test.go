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

// --- Structs and Module for 'any' Attribute Test ---

// EventWithAny is a struct with a field of type 'any' (interface{}).
type EventWithAny struct {
	ID      string `cty:"id"`
	Payload any    `cty:"payload"`
}

// AnyAttributeInput is the top-level input struct.
type AnyAttributeInput struct {
	Event EventWithAny `bggo:"event"`
}

// anyAttributeModule is the mock module for this test.
type anyAttributeModule struct {
	capturedInput AnyAttributeInput
	mu            sync.Mutex
}

func (m *anyAttributeModule) Register(r *registry.Registry) {
	r.RegisterRunner("OnRunAnyAttribute", &registry.RegisteredRunner{
		NewInput:  func() any { return new(AnyAttributeInput) },
		InputType: reflect.TypeOf(AnyAttributeInput{}),
		NewDeps:   func() any { return new(struct{}) },
		Fn: func(_ context.Context, _ any, inputRaw any) (any, error) {
			m.mu.Lock()
			m.capturedInput = *inputRaw.(*AnyAttributeInput)
			m.mu.Unlock()
			return nil, nil
		},
	})
}

// TestCoreExecution_ObjectWithAnyAttribute_Success validates that an object
// containing an attribute of type 'any' decodes correctly.
func TestCoreExecution_ObjectWithAnyAttribute_Success(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	// MANIFEST: Defines an 'event' input with a 'payload' attribute of type 'any'.
	manifestHCL := `
		runner "any_attribute_runner" {
			lifecycle {
				on_run = "OnRunAnyAttribute"
			}
			input "event" {
				type = object({
					id      = string
					payload = any
				})
			}
		}
	`

	// GRID: Provides a string value for the 'payload' attribute.
	gridHCL := `
		step "any_attribute_runner" "test" {
			arguments {
				event = {
					id      = "evt-user-created-123"
					payload = "user 'test-user' created successfully"
				}
			}
		}
	`

	files := map[string]string{
		filepath.Join("modules", "any_attr", "manifest.hcl"): manifestHCL,
		"main.hcl": gridHCL,
	}

	mockModule := &anyAttributeModule{}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.NoError(t, result.Err, "Run failed unexpectedly for 'any' attribute test. Logs:\n%s", result.LogOutput)

	expectedInput := AnyAttributeInput{
		Event: EventWithAny{
			ID:      "evt-user-created-123",
			Payload: "user 'test-user' created successfully", // Expect a native Go string
		},
	}

	mockModule.mu.Lock()
	defer mockModule.mu.Unlock()

	if diff := cmp.Diff(expectedInput, mockModule.capturedInput); diff != "" {
		t.Errorf("Captured data with 'any' attribute mismatch (-want +got):\n%s", diff)
	}
}
