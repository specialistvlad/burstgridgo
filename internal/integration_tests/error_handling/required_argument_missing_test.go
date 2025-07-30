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

// mockRequiredArgModule only registers the Go handler for the runner.
type mockRequiredArgModule struct{}

// Register registers the "required_arg_runner" Go handler.
func (m *mockRequiredArgModule) Register(r *registry.Registry) {
	type runnerInput struct {
		Name string `hcl:"name"`
	}
	r.RegisterRunner("OnRunRequiredArg", &registry.RegisteredRunner{
		NewInput: func() any { return new(runnerInput) },
		NewDeps:  func() any { return new(struct{}) },
		Fn:       func(context.Context, any, any) (cty.Value, error) { return cty.NilVal, nil },
	})
}

// Test for: App run fails if a required runner argument is missing.
func TestErrorHandling_RequiredArgumentMissing_FailsRun(t *testing.T) {
	// --- Arrange ---
	tempDir := t.TempDir()

	// 1. Define and write the HCL manifest for the runner, declaring "name" as a required input.
	manifestHCL := `
		runner "required_arg_runner" {
			lifecycle { on_run = "OnRunRequiredArg" }
			input "name" {
				type = string
				# 'optional' is false by default, making this required.
			}
		}
	`
	moduleDir := filepath.Join(tempDir, "modules", "required_arg_runner")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("failed to create module directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "manifest.hcl"), []byte(manifestHCL), 0600); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// 2. This HCL is invalid because the 'name' argument for the step is missing.
	gridHCL := `
		step "required_arg_runner" "A" {
			arguments {
				# The required 'name' argument is omitted here.
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
	testApp, _ := app.SetupAppTest(t, appConfig, &mockRequiredArgModule{})

	// --- Act ---
	runErr := testApp.Run(context.Background(), appConfig)

	// --- Assert ---
	if runErr == nil {
		t.Fatal("app.Run() should have returned an error for a missing required argument, but it returned nil")
	}

	// Check for the error message that the HCL library produces.
	errMsg := runErr.Error()
	expectedErrorSubstring := `The argument "name" is required`
	if !strings.Contains(errMsg, expectedErrorSubstring) {
		t.Errorf("expected error message to contain %q, but got: %s", expectedErrorSubstring, errMsg)
	}
}
