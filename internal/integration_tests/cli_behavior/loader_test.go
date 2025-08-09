package integration_tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/hcl_adapter"
	"github.com/zclconf/go-cty/cty"
)

func TestLoader_Load(t *testing.T) {
	// --- Arrange ---
	tempDir := t.TempDir()

	// 1. Create module manifest file.
	moduleDir := filepath.Join(tempDir, "modules", "test_runner")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("failed to create module directory: %v", err)
	}
	manifestHCL := `
		runner "test_runner" {
			description = "A test runner."
			lifecycle {
				on_run = "OnRunTest"
			}
			input "message" {
				type = string
				default = "default_message"
			}
		}
	`
	if err := os.WriteFile(filepath.Join(moduleDir, "manifest.hcl"), []byte(manifestHCL), 0600); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// 2. Create grid file.
	gridHCL := `
		step "test_runner" "A" {
			arguments {
				message = "hello"
			}
		}
	`
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(gridHCL), 0600); err != nil {
		t.Fatalf("failed to write grid file: %v", err)
	}

	// --- Act ---
	loader := hcl_adapter.NewLoader()
	model, converter, err := loader.Load(context.Background(), gridPath, filepath.Join(tempDir, "modules"))

	// --- Assert ---
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	if model == nil {
		t.Fatal("Load() returned a nil model")
	}
	if converter == nil {
		t.Fatal("Load() returned a nil converter")
	}

	// Assert on the Runner Definition
	defaultValue := cty.StringVal("default_message")
	expectedRunner := &config.RunnerDefinition{
		Type:        "test_runner",
		Description: "A test runner.",
		Lifecycle:   &config.Lifecycle{OnRun: "OnRunTest"},
		Inputs: map[string]*config.InputDefinition{
			"message": {
				Name:     "message",
				Type:     cty.String,
				Default:  &defaultValue,
				Optional: true,
			},
		},
	}

	// Define a comparer for cty.Type, which has unexported fields.
	ctyTypeComparer := cmp.Comparer(func(a, b cty.Type) bool {
		return a.Equals(b)
	})

	if runnerDef, ok := model.Runners["test_runner"]; ok {
		// Compare the maps of input definitions
		if diff := cmp.Diff(expectedRunner.Inputs, runnerDef.Inputs, ctyTypeComparer, cmpopts.IgnoreUnexported(cty.Value{})); diff != "" {
			t.Errorf("RunnerDefinition.Inputs mismatch (-want +got):\n%s", diff)
		}
	} else {
		t.Fatal("Expected runner 'test_runner' not found in model")
	}

	// Assert on the Grid Step
	if len(model.Grid.Steps) != 1 {
		t.Fatalf("Expected 1 step in the grid, got %d", len(model.Grid.Steps))
	}
	step := model.Grid.Steps[0]
	if step.RunnerType != "test_runner" || step.Name != "A" {
		t.Errorf("Unexpected step fields: Type=%s, Name=%s", step.RunnerType, step.Name)
	}
	if _, ok := step.Arguments["message"]; !ok {
		t.Error("Expected 'message' argument in step arguments")
	}
}
