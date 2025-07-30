package integration_tests

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/zclconf/go-cty/cty"
)

// mockSleeperModule contains the logic for a runner that can sleep.
// It now only registers the Go handler.
type mockSleeperModule struct{}

// sleeperInput defines the arguments for our mock runner.
type sleeperInput struct {
	Duration string `hcl:"duration"`
}

// Register registers the "sleeper" runner's Go handler.
func (m *mockSleeperModule) Register(r *registry.Registry) {
	r.RegisterRunner("OnRunSleeper", &registry.RegisteredRunner{
		NewInput: func() any { return new(sleeperInput) },
		NewDeps:  func() any { return new(struct{}) },
		Fn: func(ctx context.Context, deps any, input any) (cty.Value, error) {
			duration, err := time.ParseDuration(input.(*sleeperInput).Duration)
			if err != nil {
				return cty.NilVal, err
			}

			select {
			case <-time.After(duration):
				return cty.NilVal, nil // Should not be hit.
			case <-ctx.Done():
				return cty.NilVal, ctx.Err() // Will be hit.
			}
		},
	})
}

// Test for: step timeout fails run
func TestErrorHandling_StepTimeout_FailsRun(t *testing.T) {
	// --- Arrange ---
	tempDir := t.TempDir()

	// 1. Define and write the HCL manifest for the "sleeper" runner.
	manifestHCL := `
		runner "sleeper" {
			lifecycle { on_run = "OnRunSleeper" }
			input "duration" {
				type = string
			}
		}
	`
	moduleDir := filepath.Join(tempDir, "modules", "sleeper")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("failed to create module directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "manifest.hcl"), []byte(manifestHCL), 0600); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// 2. Define a grid with a step that will sleep longer than the context timeout.
	gridHCL := `
		step "sleeper" "A" {
			arguments {
				duration = "1s"
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
	testApp, _ := app.SetupAppTest(t, appConfig, &mockSleeperModule{})

	// Create a context with a very short timeout (50ms).
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// --- Act ---
	runErr := testApp.Run(ctx, appConfig)

	// --- Assert ---
	if runErr == nil {
		t.Fatal("app.Run() should have returned a timeout error, but it returned nil")
	}

	if !errors.Is(runErr, context.DeadlineExceeded) {
		t.Errorf("expected the error chain to contain context.DeadlineExceeded, but it did not. Got: %v", runErr)
	}
}
