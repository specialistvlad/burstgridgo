package system

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/schema"
	"github.com/vk/burstgridgo/internal/testutil"
	"github.com/zclconf/go-cty/cty"
)

// mockPrintModuleForCLI is a test-specific module for the CLI behavior tests.
// It registers both the Go handler and the HCL definition for the "print" runner.
type mockPrintModuleForCLI struct{}

// Register provides a mock implementation of the "OnRunPrint" handler and its definition.
func (m *mockPrintModuleForCLI) Register(r *registry.Registry) {
	// Register the mock Go handler implementation.
	r.RegisterHandler("OnRunPrint", &registry.RegisteredHandler{
		NewInput: func() any { return new(schema.StepArgs) },
		NewDeps:  func() any { return new(struct{}) },
		Fn:       func(context.Context, any, any) (cty.Value, error) { return cty.NilVal, nil },
	})

	// Register the HCL definition for the runner. This prevents the need
	// for the test to discover it from the filesystem.
	r.DefinitionRegistry["print"] = &schema.RunnerDefinition{
		Type: "print",
		Lifecycle: &schema.Lifecycle{
			OnRun: "OnRunPrint",
		},
	}
}

// Test for: config merges
func TestCLI_MergesHCL_FromDirectoryPath(t *testing.T) {
	// --- Arrange ---
	tempDir := t.TempDir()

	// Corrected: HCL blocks like 'arguments' must be on their own lines.
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

	if err := os.WriteFile(filepath.Join(tempDir, "a.hcl"), []byte(hclFileA), 0600); err != nil {
		t.Fatalf("failed to write hcl file a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "b.hcl"), []byte(hclFileB), 0600); err != nil {
		t.Fatalf("failed to write hcl file b: %v", err)
	}

	appConfig := &app.AppConfig{GridPath: tempDir}

	testApp, logBuffer := testutil.SetupAppTest(t, appConfig, &mockPrintModuleForCLI{})

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
