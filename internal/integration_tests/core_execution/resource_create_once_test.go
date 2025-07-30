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

// mockCreateOnceModule now only registers the Go handlers for its asset and runner.
type mockCreateOnceModule struct {
	createCalls *atomic.Int32
}

// Register registers the "counting_resource" asset and "spy" runner Go handlers.
func (m *mockCreateOnceModule) Register(r *registry.Registry) {
	// --- "counting_resource" Asset: Go Handlers ---
	r.RegisterAssetHandler("CreateCountingResource", &registry.RegisteredAsset{
		NewInput: func() any { return new(schema.StepArgs) },
		CreateFn: func(context.Context, any) (any, error) {
			m.createCalls.Add(1)
			return "dummy_resource_instance", nil
		},
	})
	r.RegisterAssetHandler("DestroyCountingResource", &registry.RegisteredAsset{
		DestroyFn: func(any) error { return nil },
	})

	// --- "spy" Runner: Go Handler ---
	type spyDeps struct {
		R any `hcl:"r"`
	}
	r.RegisterRunner("OnRunSpy", &registry.RegisteredRunner{
		NewInput: func() any { return new(schema.StepArgs) },
		NewDeps:  func() any { return new(spyDeps) },
		Fn:       func(context.Context, any, any) (cty.Value, error) { return cty.NilVal, nil },
	})
}

// Test for: Resource `Create` handler is called only once per instance.
func TestCoreExecution_ResourceCreateHandlerCalledOnce(t *testing.T) {
	// --- Arrange ---
	tempDir := t.TempDir()

	// 1. Define and write the HCL manifest for the "counting_resource" asset.
	assetManifestHCL := `
		asset "counting_resource" {
			lifecycle {
				create  = "CreateCountingResource"
				destroy = "DestroyCountingResource"
			}
		}
	`
	assetModuleDir := filepath.Join(tempDir, "modules", "counting_resource")
	if err := os.MkdirAll(assetModuleDir, 0755); err != nil {
		t.Fatalf("failed to create asset module dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetModuleDir, "manifest.hcl"), []byte(assetManifestHCL), 0600); err != nil {
		t.Fatalf("failed to write asset manifest: %v", err)
	}

	// 2. Define and write the HCL manifest for the "spy" runner.
	runnerManifestHCL := `
		runner "spy" {
			lifecycle { on_run = "OnRunSpy" }
			uses "r" {
				asset_type = "counting_resource"
			}
		}
	`
	runnerModuleDir := filepath.Join(tempDir, "modules", "spy")
	if err := os.MkdirAll(runnerModuleDir, 0755); err != nil {
		t.Fatalf("failed to create runner module dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runnerModuleDir, "manifest.hcl"), []byte(runnerManifestHCL), 0600); err != nil {
		t.Fatalf("failed to write runner manifest: %v", err)
	}

	// 3. This HCL defines one resource and two steps that both depend on it.
	gridHCL := `
		resource "counting_resource" "A" {}

		step "spy" "B" {
			uses {
				r = resource.counting_resource.A
			}
		}

		step "spy" "C" {
			uses {
				r = resource.counting_resource.A
			}
		}
	`
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(gridHCL), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	var createCalls atomic.Int32
	// 4. Configure the app to use the temporary directory for module discovery.
	appConfig := &app.AppConfig{
		GridPath:    gridPath,
		ModulesPath: filepath.Join(tempDir, "modules"),
	}
	mockModule := &mockCreateOnceModule{createCalls: &createCalls}
	testApp, _ := app.SetupAppTest(t, appConfig, mockModule)

	// --- Act ---
	runErr := testApp.Run(context.Background(), appConfig)
	if runErr != nil {
		t.Fatalf("app.Run() returned an unexpected error: %v", runErr)
	}

	// --- Assert ---
	finalCallCount := createCalls.Load()
	if finalCallCount != 1 {
		t.Errorf("expected resource Create handler to be called 1 time, but it was called %d times", finalCallCount)
	}
}
