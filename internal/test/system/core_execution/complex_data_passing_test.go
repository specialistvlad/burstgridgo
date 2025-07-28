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

// mockDataPassingModule is a self-contained module for this specific test.
type mockDataPassingModule struct {
	wg            *sync.WaitGroup
	sourceOutput  cty.Value
	capturedInput cty.Value
	mu            sync.Mutex
}

// Register registers the "source" and "spy" runners.
func (m *mockDataPassingModule) Register(r *registry.Registry) {
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

// Test for: Complex data (objects, lists) passes correctly between steps.
func TestCoreExecution_ComplexDataPassing(t *testing.T) {
	// --- Arrange ---
	// Define the complex, nested data structure we will pass.
	expectedData := cty.ObjectVal(map[string]cty.Value{
		"id":      cty.NumberIntVal(99),
		"name":    cty.StringVal("complex-object"),
		"enabled": cty.BoolVal(true),
		"metadata": cty.ObjectVal(map[string]cty.Value{
			"owner": cty.StringVal("test-suite"),
		}),
		"items": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{"item_id": cty.NumberIntVal(1)}),
			cty.ObjectVal(map[string]cty.Value{"item_id": cty.NumberIntVal(2)}),
		}),
	})

	// The HCL wires the source's output directly to the spy's input.
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
	wg.Add(1) // The spy runner will call Done().

	appConfig := &app.AppConfig{GridPath: gridPath}
	mockModule := &mockDataPassingModule{
		wg:           &wg,
		sourceOutput: expectedData,
	}
	testApp, _ := testutil.SetupAppTest(t, appConfig, mockModule)

	// --- Act ---
	runErr := testApp.Run(context.Background(), appConfig)
	if runErr != nil {
		t.Fatalf("app.Run() returned an unexpected error: %v", runErr)
	}

	wg.Wait()

	// --- Assert ---
	// Use go-cmp to recursively compare the captured value with the expected one.
	// We compare the GoString() representation for a stable comparison.
	if diff := cmp.Diff(expectedData.GoString(), mockModule.capturedInput.GoString()); diff != "" {
		t.Errorf("Captured complex data mismatch (-want +got):\n%s", diff)
	}
}
