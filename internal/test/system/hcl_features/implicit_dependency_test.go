package system

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/schema"
	"github.com/vk/burstgridgo/internal/testutil"
	"github.com/zclconf/go-cty/cty"
)

// mockSourceSpyModule is a self-contained module for the implicit dependency test.
type mockSourceSpyModule struct {
	wg            *sync.WaitGroup
	sourceOutput  cty.Value
	capturedInput cty.Value
	mu            sync.Mutex
}

// Register registers the "source" and "spy" runners.
func (m *mockSourceSpyModule) Register(r *registry.Registry) {
	// --- "source" Runner: Produces a predictable, hardcoded output. ---
	r.RegisterHandler("OnRunSource", &registry.RegisteredHandler{
		NewInput: func() any { return new(schema.StepArgs) },
		NewDeps:  func() any { return new(struct{}) },
		Fn:       func(context.Context, any, any) (cty.Value, error) { return m.sourceOutput, nil },
	})
	r.DefinitionRegistry["source"] = &schema.RunnerDefinition{
		Type:      "source",
		Lifecycle: &schema.Lifecycle{OnRun: "OnRunSource"},
		Outputs:   []*schema.OutputDefinition{{Name: "data"}},
	}

	// --- "spy" Runner: Captures its input for the test to inspect. ---
	type spyInput struct {
		Value cty.Value `hcl:"input"`
	}
	r.RegisterHandler("OnRunSpy", &registry.RegisteredHandler{
		NewInput: func() any { return new(spyInput) },
		NewDeps:  func() any { return new(struct{}) },
		Fn: func(_ context.Context, _ any, inputRaw any) (cty.Value, error) {
			m.mu.Lock()
			m.capturedInput = inputRaw.(*spyInput).Value
			m.mu.Unlock()
			m.wg.Done() // Signal that the spy has captured the input.
			return cty.NilVal, nil
		},
	})
	r.DefinitionRegistry["spy"] = &schema.RunnerDefinition{
		Type:      "spy",
		Lifecycle: &schema.Lifecycle{OnRun: "OnRunSpy"},
		Inputs:    []*schema.InputDefinition{{Name: "input"}},
	}
}

// Test for: implicit dependency
func TestHclFeatures_ImplicitDependency(t *testing.T) {
	// --- Arrange ---
	// Define the data that our source will produce and that we expect our spy to receive.
	expectedData := cty.ObjectVal(map[string]cty.Value{
		"message": cty.StringVal("hello from source"),
		"id":      cty.NumberIntVal(123),
	})

	// The HCL grid where the 'spy' step's input is directly interpolated
	// from the 'source' step's output. No explicit 'depends_on' is used.
	hcl := `
		step "source" "A" {
			arguments {}
		}

		step "spy" "B" {
			arguments {
				input = step.source.A.output
			}
		}
	`
	tempDir := t.TempDir()
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(hcl), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1) // The spy runner will call Done() when it executes.

	appConfig := &app.AppConfig{GridPath: gridPath}
	mockModule := &mockSourceSpyModule{
		wg:           &wg,
		sourceOutput: expectedData,
	}
	testApp, _ := testutil.SetupAppTest(t, appConfig, mockModule)

	// --- Act ---
	runErr := testApp.Run(context.Background(), appConfig)
	if runErr != nil {
		t.Fatalf("app.Run() returned an unexpected error: %v", runErr)
	}

	// Wait for the spy runner to finish executing. If the dependency was not
	// inferred, the test may time out here if the spy never runs or runs too early.
	wg.Wait()

	// --- Assert ---
	// Use go-cmp to compare the captured cty.Value with the expected value.
	if diff := cmp.Diff(expectedData.GoString(), mockModule.capturedInput.GoString()); diff != "" {
		t.Errorf("Captured input mismatch (-want +got):\n%s", diff)
	}
}
