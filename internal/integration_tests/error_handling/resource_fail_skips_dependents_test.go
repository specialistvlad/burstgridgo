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

// mockResourceFailModule now only registers the necessary Go handlers.
type mockResourceFailModule struct {
	wasSpyExecuted *atomic.Bool
	injectedError  error
}

// Register registers the failing asset and the spy runner Go handlers.
func (m *mockResourceFailModule) Register(r *registry.Registry) {
	// --- "failing_resource" Asset: Go Handlers ---
	r.RegisterAssetHandler("CreateFailingResource", &registry.RegisteredAsset{
		NewInput: func() any { return new(schema.StepArgs) },
		CreateFn: func(context.Context, any) (any, error) { return nil, m.injectedError },
	})
	r.RegisterAssetHandler("DestroyFailingResource", &registry.RegisteredAsset{
		DestroyFn: func(any) error { return nil },
	})

	// --- "spy" Runner: Go Handler ---
	r.RegisterRunner("OnRunSpy", &registry.RegisteredRunner{
		NewInput: func() any { return new(schema.StepArgs) },
		NewDeps:  func() any { return new(struct{}) },
		Fn: func(context.Context, any, any) (cty.Value, error) {
			m.wasSpyExecuted.Store(true)
			return cty.NilVal, nil
		},
	})
}

// Test for: resource fail skips dependents
func TestErrorHandling_ResourceFailure_SkipsDependents(t *testing.T) {
	// --- Arrange ---
	tempDir := t.TempDir()

	// 1. Define and write HCL manifests for the asset and runner.
	assetManifest := `
		asset "failing_resource" {
			lifecycle {
				create = "CreateFailingResource"
				destroy = "DestroyFailingResource"
			}
		}
	`
	runnerManifest := `
		runner "spy" {
			lifecycle { on_run = "OnRunSpy" }
			uses "r" {
				asset_type = "failing_resource"
			}
		}
	`
	if err := os.MkdirAll(filepath.Join(tempDir, "modules", "failing_resource"), 0755); err != nil {
		t.Fatalf("failed to create asset module dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tempDir, "modules", "spy"), 0755); err != nil {
		t.Fatalf("failed to create runner module dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "modules/failing_resource/manifest.hcl"), []byte(assetManifest), 0600); err != nil {
		t.Fatalf("failed to write asset manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "modules/spy/manifest.hcl"), []byte(runnerManifest), 0600); err != nil {
		t.Fatalf("failed to write runner manifest: %v", err)
	}

	// 2. Define the user's grid file.
	expectedErr := errors.New("resource creation failed as expected")
	gridHCL := `
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
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(gridHCL), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	var wasSpyExecuted atomic.Bool
	// 3. Configure the app for module discovery.
	appConfig := &app.AppConfig{
		GridPath:    gridPath,
		ModulesPath: filepath.Join(tempDir, "modules"),
	}
	mockModule := &mockResourceFailModule{
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
		t.Error("fail-fast did not work: a step dependent on the failing resource was executed")
	}
}
