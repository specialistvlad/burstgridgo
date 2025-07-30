package integration_tests

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
	"github.com/zclconf/go-cty/cty"
)

// mockFailerModule now only registers the Go handlers for the runners.
type mockFailerModule struct {
	wasSpyExecuted *atomic.Bool
	injectedError  error
}

// Register registers the "failer" and "spy" runner Go handlers.
func (m *mockFailerModule) Register(r *registry.Registry) {
	// --- "failer" Runner: Go Handler ---
	r.RegisterRunner("OnRunFailer", &registry.RegisteredRunner{
		NewInput: func() any { return new(schema.StepArgs) },
		NewDeps:  func() any { return new(struct{}) },
		Fn:       func(context.Context, any, any) (cty.Value, error) { return cty.NilVal, m.injectedError },
	})

	// --- "spy" Runner: Go Handler ---
	r.RegisterRunner("OnRunSpy", &registry.RegisteredRunner{
		NewInput: func() any { return new(schema.StepArgs) },
		NewDeps:  func() any { return new(struct{}) },
		Fn: func(context.Context, any, any) (cty.Value, error) {
			m.wasSpyExecuted.Store(true) // If this runs, the test has failed.
			return cty.NilVal, nil
		},
	})
}

// Test for: step fail triggers fast fail
func TestErrorHandling_FailingStep_TriggersFailFast(t *testing.T) {
	// --- Arrange ---
	tempDir := t.TempDir()

	// 1. Define and write HCL manifests for the "failer" and "spy" runners.
	failerManifestHCL := `
		runner "failer" {
			lifecycle { on_run = "OnRunFailer" }
		}
	`
	spyManifestHCL := `
		runner "spy" {
			lifecycle { on_run = "OnRunSpy" }
		}
	`
	if err := os.MkdirAll(filepath.Join(tempDir, "modules", "failer"), 0755); err != nil {
		t.Fatalf("failed to create failer module dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tempDir, "modules", "spy"), 0755); err != nil {
		t.Fatalf("failed to create spy module dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "modules/failer/manifest.hcl"), []byte(failerManifestHCL), 0600); err != nil {
		t.Fatalf("failed to write failer manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "modules/spy/manifest.hcl"), []byte(spyManifestHCL), 0600); err != nil {
		t.Fatalf("failed to write spy manifest: %v", err)
	}

	// 2. Define a specific error to inject and later check for.
	expectedErr := errors.New("handler failed as expected")

	// 3. The HCL defines a simple dependency: the failing step runs first.
	gridHCL := `
		step "failer" "A" {
			arguments {}
		}

		step "spy" "B" {
			arguments {}
			depends_on = ["failer.A"]
		}
	`
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(gridHCL), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	var wasSpyExecuted atomic.Bool

	// 4. Set up the app with our test-specific mock module and discovery path.
	appConfig := &app.AppConfig{
		GridPath:    gridPath,
		ModulesPath: filepath.Join(tempDir, "modules"),
	}
	mockModule := &mockFailerModule{
		wasSpyExecuted: &wasSpyExecuted,
		injectedError:  expectedErr,
	}
	testApp, _ := app.SetupAppTest(t, appConfig, mockModule)

	// --- Act ---
	runErr := testApp.Run(context.Background(), appConfig)

	// --- Assert ---
	if runErr == nil {
		t.Fatal("app.Run() should have returned an error, but it returned nil")
	}

	if !errors.Is(runErr, expectedErr) {
		t.Errorf("expected the error chain to contain our injected error, but it did not. Got: %v", runErr)
	}

	if wasSpyExecuted.Load() {
		t.Error("fail-fast did not work: a step dependent on the failing step was executed")
	}
}
