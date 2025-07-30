package integration_tests

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/zclconf/go-cty/cty"
)

// mockManifestValidationModule now only registers the Go handler.
type mockManifestValidationModule struct{}

// Register registers the Go handler for the "manifest_runner".
func (m *mockManifestValidationModule) Register(r *registry.Registry) {
	type runnerInput struct {
		Name string `hcl:"name"`
	}
	r.RegisterRunner("OnRunManifest", &registry.RegisteredRunner{
		NewInput: func() any { return new(runnerInput) },
		NewDeps:  func() any { return new(struct{}) },
		Fn:       func(context.Context, any, any) (cty.Value, error) { return cty.NilVal, nil },
	})
}

// Test for: App run fails if a step references an output that is not declared in the manifest.
func TestErrorHandling_ReferenceToUndeclaredOutput_FailsRun(t *testing.T) {
	// --- Arrange ---
	tempDir := t.TempDir()

	// 1. Define and write the HCL manifest for the runner.
	// Note: This manifest correctly declares the 'name' input, but has NO 'output' blocks.
	manifestHCL := `
		runner "manifest_runner" {
			lifecycle { on_run = "OnRunManifest" }
			input "name" {
				type = string
			}
		}
	`
	moduleDir := filepath.Join(tempDir, "modules", "manifest_runner")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("failed to create module directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "manifest.hcl"), []byte(manifestHCL), 0600); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// 2. The user's grid file, which attempts to access a non-existent output.
	gridHCL := `
		step "manifest_runner" "A" {
			arguments {
				name = "step A"
			}
		}

		step "manifest_runner" "B" {
			arguments {
				// This references an output that is not declared in the manifest.
				name = step.manifest_runner.A.output.undeclared_value
			}
		}
	`
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(gridHCL), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	// 3. Configure the app for module discovery.
	appConfig := &app.AppConfig{
		GridPath:    gridPath,
		ModulesPath: filepath.Join(tempDir, "modules"),
	}
	mockModule := &mockManifestValidationModule{}
	app, _ := app.SetupAppTest(t, appConfig, mockModule)

	// --- Act ---
	runErr := app.Run(context.Background(), appConfig)

	// --- Assert ---
	if runErr == nil {
		t.Fatal("app.Run() should have failed for a reference to an undeclared output, but it did not")
	}

	expectedErr := "undeclared output"
	if !strings.Contains(runErr.Error(), expectedErr) {
		t.Errorf("expected error to contain %q, but got: %v", expectedErr, runErr)
	}
}
