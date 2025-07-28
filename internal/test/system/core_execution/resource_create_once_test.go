package system

import (
	"context"
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

// mockCreateOnceModule is a self-contained module for this specific test.
type mockCreateOnceModule struct {
	createCalls *atomic.Int32
}

// Register registers the "counting_resource" asset and a "spy" runner that uses it.
func (m *mockCreateOnceModule) Register(r *registry.Registry) {
	// --- "counting_resource" Asset ---
	// The CreateFn for this asset increments our atomic counter.
	r.RegisterAssetHandler("CreateCountingResource", &registry.RegisteredAssetHandler{
		NewInput: func() any { return new(schema.StepArgs) },
		CreateFn: func(context.Context, any) (any, error) {
			m.createCalls.Add(1)
			return "dummy_resource_instance", nil // The instance itself can be anything.
		},
	})
	r.RegisterAssetHandler("DestroyCountingResource", &registry.RegisteredAssetHandler{
		DestroyFn: func(any) error { return nil },
	})
	r.AssetDefinitionRegistry["counting_resource"] = &schema.AssetDefinition{
		Type: "counting_resource",
		Lifecycle: &schema.AssetLifecycle{
			Create:  "CreateCountingResource",
			Destroy: "DestroyCountingResource",
		},
	}

	// --- "spy" Runner ---
	// This runner simply declares its dependency on the resource, which is
	// enough to trigger the engine's resource management logic.
	type spyDeps struct {
		R any `hcl:"r"`
	}
	r.RegisterHandler("OnRunSpy", &registry.RegisteredHandler{
		NewInput: func() any { return new(schema.StepArgs) },
		NewDeps:  func() any { return new(spyDeps) },
		Fn:       func(context.Context, any, any) (cty.Value, error) { return cty.NilVal, nil },
	})
	r.DefinitionRegistry["spy"] = &schema.RunnerDefinition{
		Type: "spy",
		Lifecycle: &schema.Lifecycle{
			OnRun: "OnRunSpy",
		},
		Uses: []*schema.UsesDefinition{
			{LocalName: "r", AssetType: "counting_resource"},
		},
	}
}

// Test for: Resource `Create` handler is called only once per instance.
func TestCoreExecution_ResourceCreateHandlerCalledOnce(t *testing.T) {
	// --- Arrange ---
	// This HCL defines one resource and two steps that both depend on it.
	// The engine should be smart enough to only create the resource once.
	hcl := `
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
	tempDir := t.TempDir()
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(hcl), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	var createCalls atomic.Int32
	appConfig := &app.AppConfig{GridPath: gridPath}
	mockModule := &mockCreateOnceModule{createCalls: &createCalls}
	testApp, _ := testutil.SetupAppTest(t, appConfig, mockModule)

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
