package system

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
	"github.com/vk/burstgridgo/internal/testutil"
	"github.com/zclconf/go-cty/cty"
)

// mockInstanceSharingModule is a self-contained module for this specific test.
type mockInstanceSharingModule struct {
	wg               *sync.WaitGroup
	capturedPointers map[string]uintptr
	mu               sync.Mutex
}

// sharableResource is the simple object we will be sharing.
// Its memory address is what we will use to verify instance identity.
type sharableResource struct {
	ID int
}

// Register registers the necessary assets and runners for the test.
func (m *mockInstanceSharingModule) Register(r *registry.Registry) {
	// --- "sharable_resource" Asset ---
	// The CreateFn returns a pointer to a new sharableResource.
	r.RegisterAssetHandler("CreateSharableResource", &registry.RegisteredAssetHandler{
		NewInput: func() any { return new(schema.StepArgs) },
		CreateFn: func(context.Context, any) (any, error) {
			return &sharableResource{ID: 42}, nil
		},
	})
	r.RegisterAssetHandler("DestroySharableResource", &registry.RegisteredAssetHandler{
		DestroyFn: func(any) error { return nil },
	})
	r.AssetDefinitionRegistry["sharable_resource"] = &schema.AssetDefinition{
		Type: "sharable_resource",
		Lifecycle: &schema.AssetLifecycle{
			Create:  "CreateSharableResource",
			Destroy: "DestroySharableResource",
		},
	}

	// --- "spy" Runner ---
	// This runner captures the memory address of the resource it receives.
	type spyDeps struct {
		Resource *sharableResource `hcl:"r"`
	}
	type spyInput struct {
		Name string `hcl:"name"`
	}
	r.RegisterHandler("OnRunSpy", &registry.RegisteredHandler{
		NewInput: func() any { return new(spyInput) },
		NewDeps:  func() any { return new(spyDeps) },
		Fn: func(_ context.Context, depsRaw any, inputRaw any) (cty.Value, error) {
			defer m.wg.Done()
			deps := depsRaw.(*spyDeps)
			input := inputRaw.(*spyInput)

			// Correctly capture the memory address (pointer) of the received resource.
			m.mu.Lock()
			m.capturedPointers[input.Name] = reflect.ValueOf(deps.Resource).Pointer()
			m.mu.Unlock()

			return cty.NilVal, nil
		},
	})
	r.DefinitionRegistry["spy"] = &schema.RunnerDefinition{
		Type:      "spy",
		Lifecycle: &schema.Lifecycle{OnRun: "OnRunSpy"},
		Uses:      []*schema.UsesDefinition{{LocalName: "r", AssetType: "sharable_resource"}},
		Inputs:    []*schema.InputDefinition{{Name: "name"}},
	}
}

// Test for: All dependent steps receive the exact same resource instance.
func TestCoreExecution_ResourceInstanceSharing(t *testing.T) {
	// --- Arrange ---
	hcl := `
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
	tempDir := t.TempDir()
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(hcl), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2) // Expect two spy steps to run.

	appConfig := &app.AppConfig{GridPath: gridPath}
	mockModule := &mockInstanceSharingModule{
		wg:               &wg,
		capturedPointers: make(map[string]uintptr),
	}
	testApp, _ := testutil.SetupAppTest(t, appConfig, mockModule)

	// --- Act ---
	runErr := testApp.Run(context.Background(), appConfig)
	if runErr != nil {
		t.Fatalf("app.Run() returned an unexpected error: %v", runErr)
	}

	wg.Wait() // Wait for both spy runners to complete.

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
