package system

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/schema"
	"github.com/vk/burstgridgo/internal/testutil"
	"github.com/zclconf/go-cty/cty"
)

// mockFailerModule is a self-contained module for the fail-fast test.
type mockFailerModule struct {
	wasSpyExecuted *atomic.Bool
	injectedError  error
}

// Register registers the "failer" and "spy" runner handlers and definitions.
func (m *mockFailerModule) Register(r *registry.Registry) {
	// --- "failer" Runner ---
	r.RegisterHandler("OnRunFailer", &registry.RegisteredHandler{
		NewInput: func() any { return new(schema.StepArgs) },
		NewDeps:  func() any { return new(struct{}) },
		Fn:       func(context.Context, any, any) (cty.Value, error) { return cty.NilVal, m.injectedError },
	})
	r.DefinitionRegistry["failer"] = &schema.RunnerDefinition{
		Type:      "failer",
		Lifecycle: &schema.Lifecycle{OnRun: "OnRunFailer"},
	}

	// --- "spy" Runner ---
	r.RegisterHandler("OnRunSpy", &registry.RegisteredHandler{
		NewInput: func() any { return new(schema.StepArgs) },
		NewDeps:  func() any { return new(struct{}) },
		Fn: func(context.Context, any, any) (cty.Value, error) {
			m.wasSpyExecuted.Store(true) // If this runs, the test has failed.
			return cty.NilVal, nil
		},
	})
	r.DefinitionRegistry["spy"] = &schema.RunnerDefinition{
		Type:      "spy",
		Lifecycle: &schema.Lifecycle{OnRun: "OnRunSpy"},
	}
}

// Test for: step fail triggers fast fail
func TestErrorHandling_FailingStep_TriggersFailFast(t *testing.T) {
	// --- Arrange ---
	// Define a specific error to inject and later check for.
	expectedErr := errors.New("handler failed as expected")

	// The HCL defines a simple dependency: the failing step runs first,
	// and the spy step will only run if the first one succeeds.
	hcl := `
		step "failer" "A" {
			arguments {}
		}

		step "spy" "B" {
			arguments {}
			depends_on = ["failer.A"]
		}
	`
	tempDir := t.TempDir()
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(hcl), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	// wasSpyExecuted will be set to true if the dependent step runs,
	// which would indicate that fail-fast did *not* work.
	var wasSpyExecuted atomic.Bool

	// Set up the app with our test-specific mock module.
	appConfig := &app.AppConfig{GridPath: gridPath}
	mockModule := &mockFailerModule{
		wasSpyExecuted: &wasSpyExecuted,
		injectedError:  expectedErr,
	}
	testApp, _ := testutil.SetupAppTest(t, appConfig, mockModule)

	// --- Act ---
	// Run the application. We expect this to return an error.
	runErr := testApp.Run(context.Background(), appConfig)

	// --- Assert ---
	// 1. Check that an error was returned.
	if runErr == nil {
		t.Fatal("app.Run() should have returned an error, but it returned nil")
	}

	// 2. Check that the returned error contains our specific injected error.
	if !errors.Is(runErr, expectedErr) {
		t.Errorf("expected the error chain to contain our injected error, but it did not. Got: %v", runErr)
	}

	// 3. Check that the spy step was never executed.
	if wasSpyExecuted.Load() {
		t.Error("fail-fast did not work: a step dependent on the failing step was executed")
	}
}
