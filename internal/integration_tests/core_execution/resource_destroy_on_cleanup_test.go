package integration_tests

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/schema"
	"github.com/zclconf/go-cty/cty"
)

// mockDestroySpyModule is a self-contained module for this specific test.
// It now only registers the required Go handlers.
type mockDestroySpyModule struct {
	destroyCalls *atomic.Int32
}

// Register registers the Go handlers for the "destroy_spy_resource" asset and "dummy" runner.
func (m *mockDestroySpyModule) Register(r *registry.Registry) {
	// --- "destroy_spy_resource" Asset: Go Handlers ---
	r.RegisterAssetHandler("CreateDestroySpyResource", &registry.RegisteredAsset{
		NewInput: func() any { return new(schema.StepArgs) },
		CreateFn: func(context.Context, any) (any, error) {
			return "dummy_instance", nil
		},
	})
	r.RegisterAssetHandler("DestroyDestroySpyResource", &registry.RegisteredAsset{
		DestroyFn: func(any) error {
			m.destroyCalls.Add(1)
			return nil
		},
	})

	// --- "dummy" Runner: Go Handler ---
	type dummyDeps struct {
		R any `hcl:"r"`
	}
	r.RegisterRunner("OnRunDummy", &registry.RegisteredRunner{
		NewInput: func() any { return new(schema.StepArgs) },
		NewDeps:  func() any { return new(dummyDeps) },
		Fn:       func(context.Context, any, any) (cty.Value, error) { return cty.NilVal, nil },
	})
}

// Test for: Resource `Destroy` handler is called once on cleanup.
func TestCoreExecution_ResourceDestroyOnCleanup(t *testing.T) {
	// --- Arrange ---
	tempDir := t.TempDir()

	// 1. Define and write the HCL manifest for the asset.
	assetManifestHCL := `
		asset "destroy_spy_resource" {
			lifecycle {
				create = "CreateDestroySpyResource"
				destroy = "DestroyDestroySpyResource"
			}
		}
	`
	assetModuleDir := filepath.Join(tempDir, "modules", "destroy_spy_resource")
	if err := os.MkdirAll(assetModuleDir, 0755); err != nil {
		t.Fatalf("failed to create asset module dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetModuleDir, "manifest.hcl"), []byte(assetManifestHCL), 0600); err != nil {
		t.Fatalf("failed to write asset manifest: %v", err)
	}

	// 2. Define and write the HCL manifest for the runner.
	runnerManifestHCL := `
		runner "dummy" {
			lifecycle { on_run = "OnRunDummy" }
			uses "r" {
				asset_type = "destroy_spy_resource"
			}
		}
	`
	runnerModuleDir := filepath.Join(tempDir, "modules", "dummy")
	if err := os.MkdirAll(runnerModuleDir, 0755); err != nil {
		t.Fatalf("failed to create runner module dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runnerModuleDir, "manifest.hcl"), []byte(runnerManifestHCL), 0600); err != nil {
		t.Fatalf("failed to write runner manifest: %v", err)
	}

	// 3. The user's grid file.
	gridHCL := `
		resource "destroy_spy_resource" "A" {}

		step "dummy" "B" {
			uses {
				r = resource.destroy_spy_resource.A
			}
		}
	`
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(gridHCL), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	var destroyCalls atomic.Int32
	// 4. Configure the app to use the temporary directory for module discovery.
	appConfig := &app.AppConfig{
		GridPath:    gridPath,
		ModulesPath: filepath.Join(tempDir, "modules"),
	}
	mockModule := &mockDestroySpyModule{destroyCalls: &destroyCalls}
	testApp, _ := app.SetupAppTest(t, appConfig, mockModule)

	// --- Act ---
	runErr := testApp.Run(context.Background(), appConfig)
	if runErr != nil {
		t.Fatalf("app.Run() returned an unexpected error: %v", runErr)
	}

	// --- Assert ---
	finalCallCount := destroyCalls.Load()
	if finalCallCount != 1 {
		t.Errorf("expected resource Destroy handler to be called 1 time, but it was called %d times", finalCallCount)
	}
}
