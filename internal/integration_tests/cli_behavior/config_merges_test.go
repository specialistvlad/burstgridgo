package integration_tests

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/schema"
	"github.com/zclconf/go-cty/cty"
)

// mockPrintModuleForCLI now only registers the Go handler for the "print" runner.
type mockPrintModuleForCLI struct{}

// Register provides a mock implementation of the "OnRunPrint" handler.
func (m *mockPrintModuleForCLI) Register(r *registry.Registry) {
	r.RegisterRunner("OnRunPrint", &registry.RegisteredRunner{
		NewInput: func() any { return new(schema.StepArgs) },
		NewDeps:  func() any { return new(struct{}) },
		Fn:       func(context.Context, any, any) (cty.Value, error) { return cty.NilVal, nil },
	})
}

// Test for: config merges
func TestCLI_MergesHCL_FromDirectoryPath(t *testing.T) {
	// --- Arrange ---
	tempDir := t.TempDir()

	// 1. Define and write the HCL manifest for the "print" runner.
	manifestHCL := `
		runner "print" {
			lifecycle {
				on_run = "OnRunPrint"
			}
		}
	`
	moduleDir := filepath.Join(tempDir, "modules", "print")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("failed to create module directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "manifest.hcl"), []byte(manifestHCL), 0600); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// 2. Create multiple grid files in a single directory.
	gridDir := filepath.Join(tempDir, "grids")
	if err := os.MkdirAll(gridDir, 0755); err != nil {
		t.Fatalf("failed to create grid directory: %v", err)
	}

	hclFileA := `
		step "print" "step_A" {
			arguments {}
		}
	`

	hclFileB := `
		step "print" "step_B" {
			arguments {}
		}
	`

	if err := os.WriteFile(filepath.Join(gridDir, "a.hcl"), []byte(hclFileA), 0600); err != nil {
		t.Fatalf("failed to write hcl file a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gridDir, "b.hcl"), []byte(hclFileB), 0600); err != nil {
		t.Fatalf("failed to write hcl file b: %v", err)
	}

	// 3. Configure the app to load from the grid directory and discover from the modules directory.
	appConfig := &app.AppConfig{
		GridPath:    gridDir,
		ModulesPath: filepath.Join(tempDir, "modules"),
	}
	testApp, logBuffer := app.SetupAppTest(t, appConfig, &mockPrintModuleForCLI{})

	// --- Act ---
	err := testApp.Run(context.Background(), appConfig)

	// --- Assert ---
	if err != nil {
		t.Fatalf("app.Run() returned an unexpected error: %v", err)
	}
	logOutput := logBuffer.String()

	if !strings.Contains(logOutput, "step=step.print.step_A") {
		t.Errorf("Expected log output for step_A, but it was not found in logs")
	}
	if !strings.Contains(logOutput, "step=step.print.step_B") {
		t.Errorf("Expected log output for step_B, but it was not found in logs")
	}
}
