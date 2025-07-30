package integration_tests

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/schema"
	"github.com/zclconf/go-cty/cty"
)

// mockInstanceSharingModule now only registers the necessary Go handlers.
type mockInstanceSharingModule struct {
	wg               *sync.WaitGroup
	capturedPointers map[string]uintptr
	mu               sync.Mutex
}

// sharableResource is the simple object we will be sharing.
type sharableResource struct {
	ID int
}

// Register registers the Go handlers for the test.
func (m *mockInstanceSharingModule) Register(r *registry.Registry) {
	// --- "sharable_resource" Asset: Go Handlers ---
	r.RegisterAssetHandler("CreateSharableResource", &registry.RegisteredAsset{
		NewInput: func() any { return new(schema.StepArgs) },
		CreateFn: func(context.Context, any) (any, error) {
			return &sharableResource{ID: 42}, nil
		},
	})
	r.RegisterAssetHandler("DestroySharableResource", &registry.RegisteredAsset{
		DestroyFn: func(any) error { return nil },
	})

	// --- "spy" Runner: Go Handler ---
	type spyDeps struct {
		Resource *sharableResource `hcl:"r"`
	}
	type spyInput struct {
		Name string `hcl:"name"`
	}
	r.RegisterRunner("OnRunSpy", &registry.RegisteredRunner{
		NewInput: func() any { return new(spyInput) },
		NewDeps:  func() any { return new(spyDeps) },
		Fn: func(_ context.Context, depsRaw any, inputRaw any) (cty.Value, error) {
			defer m.wg.Done()
			deps := depsRaw.(*spyDeps)
			input := inputRaw.(*spyInput)
			m.mu.Lock()
			m.capturedPointers[input.Name] = reflect.ValueOf(deps.Resource).Pointer()
			m.mu.Unlock()
			return cty.NilVal, nil
		},
	})
}

// Test for: All dependent steps receive the exact same resource instance.
func TestCoreExecution_ResourceInstanceSharing(t *testing.T) {
	// --- Arrange ---
	tempDir := t.TempDir()

	// 1. Define and write HCL manifests for the asset and runner.
	assetManifestHCL := `
		asset "sharable_resource" {
			lifecycle {
				create = "CreateSharableResource"
				destroy = "DestroySharableResource"
			}
		}
	`
	runnerManifestHCL := `
		runner "spy" {
			lifecycle { on_run = "OnRunSpy" }
			uses "r" {
				asset_type = "sharable_resource"
			}
			input "name" {
				type = string
			}
		}
	`
	if err := os.MkdirAll(filepath.Join(tempDir, "modules", "sharable_resource"), 0755); err != nil {
		t.Fatalf("failed to create asset module dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tempDir, "modules", "spy"), 0755); err != nil {
		t.Fatalf("failed to create runner module dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "modules/sharable_resource/manifest.hcl"), []byte(assetManifestHCL), 0600); err != nil {
		t.Fatalf("failed to write asset manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "modules/spy/manifest.hcl"), []byte(runnerManifestHCL), 0600); err != nil {
		t.Fatalf("failed to write runner manifest: %v", err)
	}

	// 2. The user's grid file.
	gridHCL := `
		resource "sharable_resource" "A" {}

		step "spy" "B" {
			uses {
				r = resource.sharable_resource.A
			}
			arguments {
				name = "B"
			}
		}

		step "spy" "C" {
			uses {
				r = resource.sharable_resource.A
			}
			arguments {
				name = "C"
			}
		}
	`
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(gridHCL), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// 3. Configure the app to use the temporary directory for module discovery.
	appConfig := &app.AppConfig{
		GridPath:    gridPath,
		ModulesPath: filepath.Join(tempDir, "modules"),
	}
	mockModule := &mockInstanceSharingModule{
		wg:               &wg,
		capturedPointers: make(map[string]uintptr),
	}
	testApp, _ := app.SetupAppTest(t, appConfig, mockModule)

	// --- Act ---
	runErr := testApp.Run(context.Background(), appConfig)
	if runErr != nil {
		t.Fatalf("app.Run() returned an unexpected error: %v", runErr)
	}

	wg.Wait()

	// --- Assert ---
	if len(mockModule.capturedPointers) != 2 {
		t.Fatalf("expected 2 captured pointers, but got %d", len(mockModule.capturedPointers))
	}

	ptrB, okB := mockModule.capturedPointers["B"]
	ptrC, okC := mockModule.capturedPointers["C"]

	if !okB || !okC {
		t.Fatal("expected to capture pointers for both step 'B' and 'C'")
	}

	if ptrB != ptrC {
		t.Errorf("resource instance was not shared correctly. Pointer for B: %v, Pointer for C: %v", ptrB, ptrC)
	}
}
