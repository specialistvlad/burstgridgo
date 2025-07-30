package integration_tests

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
	"github.com/zclconf/go-cty/cty"
)

// mockDataPassingModule is a self-contained module for this specific test.
// It now only registers the Go handlers.
type mockDataPassingModule struct {
	wg            *sync.WaitGroup
	sourceOutput  cty.Value
	capturedInput cty.Value
	mu            sync.Mutex
}

// Register registers the "source" and "spy" Go handlers.
func (m *mockDataPassingModule) Register(r *registry.Registry) {
	// --- "source" Runner: Go Handler ---
	r.RegisterRunner("OnRunSource", &registry.RegisteredRunner{
		NewInput: func() any { return new(schema.StepArgs) },
		NewDeps:  func() any { return new(struct{}) },
		Fn:       func(context.Context, any, any) (cty.Value, error) { return m.sourceOutput, nil },
	})

	// --- "spy" Runner: Go Handler ---
	type spyInput struct {
		Value cty.Value `hcl:"input"`
	}
	r.RegisterRunner("OnRunSpy", &registry.RegisteredRunner{
		NewInput: func() any { return new(spyInput) },
		NewDeps:  func() any { return new(struct{}) },
		Fn: func(_ context.Context, _ any, inputRaw any) (cty.Value, error) {
			m.mu.Lock()
			m.capturedInput = inputRaw.(*spyInput).Value
			m.mu.Unlock()
			m.wg.Done()
			return cty.NilVal, nil
		},
	})
}

// Test for: Complex data (objects, lists) passes correctly between steps.
func TestCoreExecution_ComplexDataPassing(t *testing.T) {
	// --- Arrange ---
	tempDir := t.TempDir()

	// 1. Define and write the HCL manifests for the "source" and "spy" runners.
	sourceManifestHCL := `
		runner "source" {
			lifecycle { on_run = "OnRunSource" }
			output "data" {
				type = any
			}
		}
	`
	spyManifestHCL := `
		runner "spy" {
			lifecycle { on_run = "OnRunSpy" }
			input "input" {
				type = any
			}
		}
	`
	if err := os.MkdirAll(filepath.Join(tempDir, "modules", "source"), 0755); err != nil {
		t.Fatalf("failed to create source module dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tempDir, "modules", "spy"), 0755); err != nil {
		t.Fatalf("failed to create spy module dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "modules/source/manifest.hcl"), []byte(sourceManifestHCL), 0600); err != nil {
		t.Fatalf("failed to write source manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "modules/spy/manifest.hcl"), []byte(spyManifestHCL), 0600); err != nil {
		t.Fatalf("failed to write spy manifest: %v", err)
	}

	// 2. Define the complex, nested data structure we will pass.
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

	// 3. The HCL grid wires the source's output directly to the spy's input.
	gridHCL := `
		step "source" "A" {
			arguments {}
		}

		step "spy" "B" {
			arguments {
				input = step.source.A.output
			}
		}
	`
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(gridHCL), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	// 4. Configure the app to use the temporary directory for module discovery.
	appConfig := &app.AppConfig{
		GridPath:    gridPath,
		ModulesPath: filepath.Join(tempDir, "modules"),
	}
	mockModule := &mockDataPassingModule{
		wg:           &wg,
		sourceOutput: expectedData,
	}
	testApp, _ := app.SetupAppTest(t, appConfig, mockModule)

	// --- Act ---
	runErr := testApp.Run(context.Background(), appConfig)
	if runErr != nil {
		t.Fatalf("app.Run() returned an unexpected error: %v", runErr)
	}

	wg.Wait()

	// --- Assert ---
	if diff := cmp.Diff(expectedData.GoString(), mockModule.capturedInput.GoString()); diff != "" {
		t.Errorf("Captured complex data mismatch (-want +got):\n%s", diff)
	}
}
