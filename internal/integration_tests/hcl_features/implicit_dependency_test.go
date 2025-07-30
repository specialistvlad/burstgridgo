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

// mockSourceSpyModule now only registers the Go handlers for its runners.
type mockSourceSpyModule struct {
	wg            *sync.WaitGroup
	sourceOutput  cty.Value
	capturedInput cty.Value
	mu            sync.Mutex
}

// Register registers the "source" and "spy" Go handlers.
func (m *mockSourceSpyModule) Register(r *registry.Registry) {
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

// Test for: implicit dependency
func TestHclFeatures_ImplicitDependency(t *testing.T) {
	// --- Arrange ---
	tempDir := t.TempDir()

	// 1. Define and write the HCL manifest for the "source" runner.
	sourceManifestHCL := `
		runner "source" {
			lifecycle { on_run = "OnRunSource" }
			output "data" {
				type = any
			}
		}
	`
	sourceModuleDir := filepath.Join(tempDir, "modules", "source")
	if err := os.MkdirAll(sourceModuleDir, 0755); err != nil {
		t.Fatalf("failed to create source module directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceModuleDir, "manifest.hcl"), []byte(sourceManifestHCL), 0600); err != nil {
		t.Fatalf("failed to write source manifest: %v", err)
	}

	// 2. Define and write the HCL manifest for the "spy" runner.
	spyManifestHCL := `
		runner "spy" {
			lifecycle { on_run = "OnRunSpy" }
			input "input" {
				type = any
			}
		}
	`
	spyModuleDir := filepath.Join(tempDir, "modules", "spy")
	if err := os.MkdirAll(spyModuleDir, 0755); err != nil {
		t.Fatalf("failed to create spy module directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(spyModuleDir, "manifest.hcl"), []byte(spyManifestHCL), 0600); err != nil {
		t.Fatalf("failed to write spy manifest: %v", err)
	}

	// 3. Define the data that our source will produce.
	expectedData := cty.ObjectVal(map[string]cty.Value{
		"message": cty.StringVal("hello from source"),
		"id":      cty.NumberIntVal(123),
	})

	// 4. The HCL grid where the 'spy' step's input is interpolated from the 'source' step's output.
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

	// 5. Configure the app to discover modules from our temporary directory.
	appConfig := &app.AppConfig{
		GridPath:    gridPath,
		ModulesPath: filepath.Join(tempDir, "modules"),
	}
	mockModule := &mockSourceSpyModule{
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
		t.Errorf("Captured input mismatch (-want +got):\n%s", diff)
	}
}
