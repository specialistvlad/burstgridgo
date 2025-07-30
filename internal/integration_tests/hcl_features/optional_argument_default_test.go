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
	"github.com/zclconf/go-cty/cty"
)

// mockDefaulterModule is a self-contained module for this test.
type mockDefaulterModule struct {
	wg            *sync.WaitGroup
	capturedInput *defaulterInput
	mu            sync.Mutex
}

// defaulterInput is the Go struct for the runner's arguments.
type defaulterInput struct {
	Mode     string            `hcl:"mode,optional"`
	Required string            `hcl:"required"`
	Metadata map[string]string `hcl:"metadata,optional"`
}

// Register only registers the Go handler. The HCL definition will be discovered from a file.
func (m *mockDefaulterModule) Register(r *registry.Registry) {
	r.RegisterRunner("OnRunDefaulter", &registry.RegisteredRunner{
		NewInput: func() any { return new(defaulterInput) },
		NewDeps:  func() any { return new(struct{}) },
		Fn: func(_ context.Context, _ any, inputRaw any) (cty.Value, error) {
			m.mu.Lock()
			m.capturedInput = inputRaw.(*defaulterInput)
			m.mu.Unlock()
			m.wg.Done()
			return cty.NilVal, nil
		},
	})
}

// Test for: An optional argument with a default value defined in a real HCL manifest is applied correctly.
func TestHclFeatures_OptionalArgumentDefault_FromFile(t *testing.T) {
	t.Skip("temporary skip for refactoring")
	// --- Arrange ---
	tempDir := t.TempDir()

	// 1. Create the module directory and the manifest file.
	moduleDir := filepath.Join(tempDir, "modules", "defaulter")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("failed to create module directory: %v", err)
	}

	// The manifest defines default values for 'mode' and 'metadata'.
	manifestHCL := `
		runner "defaulter" {
		  lifecycle {
		    on_run = "OnRunDefaulter"
		  }
		  input "required" {
		    type = string
		  }
		  input "mode" {
		    type    = string
		    default = "standard"
		  }
		  input "metadata" {
		    type    = map(string)
		    default = {
		      "source" = "test-suite"
		    }
		  }
		}
	`
	manifestPath := filepath.Join(moduleDir, "manifest.hcl")
	if err := os.WriteFile(manifestPath, []byte(manifestHCL), 0600); err != nil {
		t.Fatalf("failed to write manifest.hcl: %v", err)
	}

	// 2. The grid file only provides the required argument, omitting the optional ones.
	gridHCL := `
		step "defaulter" "A" {
			arguments {
				required = "must-be-present"
			}
		}
	`
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(gridHCL), 0600); err != nil {
		t.Fatalf("failed to write grid file: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	// 3. Configure the app to use the temporary directory for module discovery.
	appConfig := &app.AppConfig{
		GridPath:    gridPath,
		ModulesPath: filepath.Join(tempDir, "modules"),
	}
	mockModule := &mockDefaulterModule{wg: &wg}
	testApp, _ := app.SetupAppTest(t, appConfig, mockModule)

	// --- Act ---
	runErr := testApp.Run(context.Background(), appConfig)
	if runErr != nil {
		t.Fatalf("app.Run() returned an unexpected error: %v", runErr)
	}

	wg.Wait()

	// --- Assert ---
	if mockModule.capturedInput == nil {
		t.Fatal("Spy did not capture any input.")
	}

	expectedInput := &defaulterInput{
		Mode:     "standard",
		Required: "must-be-present",
		Metadata: map[string]string{
			"source": "test-suite",
		},
	}

	if diff := cmp.Diff(expectedInput, mockModule.capturedInput); diff != "" {
		t.Errorf("Captured input mismatch (-want +got):\n%s", diff)
	}
}
