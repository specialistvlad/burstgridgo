package integration_tests

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

// mockDataPassingModule is updated for the ADR-008 pure Go contract.
type mockDataPassingModule struct {
	sourceOutput  any
	capturedInput any
	mu            sync.Mutex
}

// Register registers the "source" and "spy" Go handlers.
func (m *mockDataPassingModule) Register(r *registry.Registry) {
	// --- "source" Runner: Returns a pure Go struct ---
	r.RegisterRunner("OnRunSource", &registry.RegisteredRunner{
		NewInput:  func() any { return new(struct{}) },
		InputType: reflect.TypeOf(struct{}{}),
		NewDeps:   func() any { return new(struct{}) },
		Fn:        func(context.Context, any, any) (any, error) { return m.sourceOutput, nil },
	})

	// --- "spy" Runner: Receives the data ---
	type spyInput struct {
		// This field is now strongly typed to match the expected data structure.
		Input complexData `bggo:"Input"`
	}
	r.RegisterRunner("OnRunSpy", &registry.RegisteredRunner{
		NewInput:  func() any { return new(spyInput) },
		InputType: reflect.TypeOf(spyInput{}),
		NewDeps:   func() any { return new(struct{}) },
		Fn: func(_ context.Context, _ any, inputRaw any) (any, error) {
			m.mu.Lock()
			// The captured input is now a `complexData` struct, not a generic map.
			m.capturedInput = inputRaw.(*spyInput).Input
			m.mu.Unlock()
			return nil, nil
		},
	})
}

// --- Structs for complex data with CTY tags for output ---

type complexMetadata struct {
	Owner string `cty:"owner"`
}

type complexItem struct {
	ItemID int `cty:"item_id"`
}

type complexData struct {
	ID       int             `cty:"id"`
	Name     string          `cty:"name"`
	Enabled  bool            `cty:"enabled"`
	Metadata complexMetadata `cty:"metadata"`
	Items    []complexItem   `cty:"items"`
}

// TestCoreExecution_ComplexDataPassing validates that complex, nested Go structs
// can be passed between steps correctly by leveraging the cty and bggo tags.
func TestCoreExecution_ComplexDataPassing(t *testing.T) {
	t.Parallel()
	// --- Arrange ---
	sourceManifestHCL := `
		runner "source" {
			lifecycle { on_run = "OnRunSource" }
			output "data" { type = any }
		}
	`
	spyManifestHCL := `
		runner "spy" {
			lifecycle { on_run = "OnRunSpy" }
			input "Input" { type = any }
		}
	`
	gridHCL := `
		step "source" "A" {
			arguments {}
		}
		step "spy" "B" {
			arguments {
				Input = step.source.A.output
			}
		}
	`
	files := map[string]string{
		filepath.Join("modules", "source", "manifest.hcl"): sourceManifestHCL,
		filepath.Join("modules", "spy", "manifest.hcl"):    spyManifestHCL,
		"main.hcl": gridHCL,
	}

	// This is the pure Go struct that the source module will return.
	nativeData := complexData{
		ID:       99,
		Name:     "complex-object",
		Enabled:  true,
		Metadata: complexMetadata{Owner: "test-suite"},
		Items:    []complexItem{{ItemID: 1}, {ItemID: 2}},
	}

	mockModule := &mockDataPassingModule{sourceOutput: nativeData}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.NoError(t, result.Err, "Run failed unexpectedly. Full logs:\n%s", result.LogOutput)

	// Now that the input is strongly typed, we can directly compare the
	// original struct with the one that made the round trip.
	if diff := cmp.Diff(nativeData, mockModule.capturedInput); diff != "" {
		t.Errorf("Captured complex data mismatch (-want +got):\n%s", diff)
	}
}
