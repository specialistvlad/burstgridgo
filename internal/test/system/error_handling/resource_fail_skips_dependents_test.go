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

// mockResourceFailModule is a self-contained module for this test.
type mockResourceFailModule struct {
	wasSpyExecuted *atomic.Bool
	injectedError  error
}

// Register registers the failing asset and the spy runner.
func (m *mockResourceFailModule) Register(r *registry.Registry) {
	// --- "failing_resource" Asset ---
	r.RegisterAssetHandler("CreateFailingResource", &registry.RegisteredAssetHandler{
		NewInput: func() any { return new(schema.StepArgs) },
		CreateFn: func(context.Context, any) (any, error) { return nil, m.injectedError },
	})
	r.RegisterAssetHandler("DestroyFailingResource", &registry.RegisteredAssetHandler{
		DestroyFn: func(any) error { return nil },
	})
	r.AssetDefinitionRegistry["failing_resource"] = &schema.AssetDefinition{
		Type: "failing_resource",
		Lifecycle: &schema.AssetLifecycle{
			Create:  "CreateFailingResource",
			Destroy: "DestroyFailingResource",
		},
	}

	// --- "spy" Runner ---
	r.RegisterHandler("OnRunSpy", &registry.RegisteredHandler{
		NewInput: func() any { return new(schema.StepArgs) },
		NewDeps:  func() any { return new(struct{}) },
		Fn: func(context.Context, any, any) (cty.Value, error) {
			m.wasSpyExecuted.Store(true)
			return cty.NilVal, nil
		},
	})
	r.DefinitionRegistry["spy"] = &schema.RunnerDefinition{
		Type:      "spy",
		Lifecycle: &schema.Lifecycle{OnRun: "OnRunSpy"},
	}
}

// Test for: resource fail skips dependents
func TestErrorHandling_ResourceFailure_SkipsDependents(t *testing.T) {
	// --- Arrange ---
	expectedErr := errors.New("resource creation failed as expected")

	hcl := `
		resource "failing_resource" "A" {
			arguments {}
		}

		step "spy" "B" {
			uses {
				r = resource.failing_resource.A
			}
			arguments {}
		}
	`
	tempDir := t.TempDir()
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(hcl), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	var wasSpyExecuted atomic.Bool
	appConfig := &app.AppConfig{GridPath: gridPath}
	mockModule := &mockResourceFailModule{
		wasSpyExecuted: &wasSpyExecuted,
		injectedError:  expectedErr,
	}
	testApp, _ := testutil.SetupAppTest(t, appConfig, mockModule)

	// --- Act ---
	runErr := testApp.Run(context.Background(), appConfig)

	// --- Assert ---
	if runErr == nil {
		t.Fatal("app.Run() should have returned an error, but it returned nil")
	}

	// Log the actual error for debugging before we assert.
	t.Logf("Error returned from app.Run(): %v", runErr)
	t.Logf("Expected to find error: %v", expectedErr)

	if !errors.Is(runErr, expectedErr) {
		t.Errorf("expected the error chain to contain our injected error, but it did not.")
	}

	if wasSpyExecuted.Load() {
		t.Error("fail-fast did not work: a step dependent on the failing resource was executed")
	}
}
