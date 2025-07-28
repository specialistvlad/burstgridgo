package system

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/schema"
	"github.com/vk/burstgridgo/internal/testutil"
	"github.com/zclconf/go-cty/cty"
)

// mockSleeperModule contains the logic for a runner that can sleep.
type mockSleeperModule struct{}

// sleeperInput defines the arguments for our mock runner.
type sleeperInput struct {
	Duration string `hcl:"duration"`
}

// Register registers the "sleeper" runner.
func (m *mockSleeperModule) Register(r *registry.Registry) {
	r.RegisterHandler("OnRunSleeper", &registry.RegisteredHandler{
		NewInput: func() any { return new(sleeperInput) },
		NewDeps:  func() any { return new(struct{}) },
		Fn: func(ctx context.Context, deps any, input any) (cty.Value, error) {
			duration, err := time.ParseDuration(input.(*sleeperInput).Duration)
			if err != nil {
				return cty.NilVal, err
			}

			// This select block is the key. It makes the handler responsive
			// to context cancellation. It will exit if either the timer
			// finishes OR the context is canceled, whichever comes first.
			select {
			case <-time.After(duration):
				// This case should not be hit in our test.
				return cty.NilVal, nil
			case <-ctx.Done():
				// This case will be hit when the test's timeout is exceeded.
				return cty.NilVal, ctx.Err()
			}
		},
	})
	r.DefinitionRegistry["sleeper"] = &schema.RunnerDefinition{
		Type:      "sleeper",
		Lifecycle: &schema.Lifecycle{OnRun: "OnRunSleeper"},
		Inputs: []*schema.InputDefinition{
			{Name: "duration"},
		},
	}
}

// Test for: step timeout fails run
func TestErrorHandling_StepTimeout_FailsRun(t *testing.T) {
	// --- Arrange ---
	// Define an HCL grid with a step that will sleep for 1 second.
	hcl := `
		step "sleeper" "A" {
			arguments {
				duration = "1s"
			}
		}
	`
	tempDir := t.TempDir()
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(hcl), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	appConfig := &app.AppConfig{GridPath: gridPath}
	testApp, _ := testutil.SetupAppTest(t, appConfig, &mockSleeperModule{})

	// Create a context with a very short timeout (50ms), which is much
	// shorter than the step's sleep duration (1s).
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// --- Act ---
	// Run the application with the timeout-enabled context.
	runErr := testApp.Run(ctx, appConfig)

	// --- Assert ---
	if runErr == nil {
		t.Fatal("app.Run() should have returned a timeout error, but it returned nil")
	}

	// The error returned by the app should wrap context.DeadlineExceeded.
	if !errors.Is(runErr, context.DeadlineExceeded) {
		t.Errorf("expected the error chain to contain context.DeadlineExceeded, but it did not. Got: %v", runErr)
	}
}
