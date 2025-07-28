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

// mockDestroySpyModule is a self-contained module for this specific test.
type mockDestroySpyModule struct {
	destroyCalls *atomic.Int32
}

// Register registers the "destroy_spy_resource" asset and a simple runner to use it.
func (m *mockDestroySpyModule) Register(r *registry.Registry) {
	// --- "destroy_spy_resource" Asset ---
	// The DestroyFn for this asset increments our atomic counter.
	r.RegisterAssetHandler("CreateDestroySpyResource", &registry.RegisteredAssetHandler{
		NewInput: func() any { return new(schema.StepArgs) },
		CreateFn: func(context.Context, any) (any, error) {
			return "dummy_instance", nil
		},
	})
	r.RegisterAssetHandler("DestroyDestroySpyResource", &registry.RegisteredAssetHandler{
		DestroyFn: func(any) error {
			m.destroyCalls.Add(1)
			return nil
		},
	})
	r.AssetDefinitionRegistry["destroy_spy_resource"] = &schema.AssetDefinition{
		Type: "destroy_spy_resource",
		Lifecycle: &schema.AssetLifecycle{
			Create:  "CreateDestroySpyResource",
			Destroy: "DestroyDestroySpyResource",
		},
	}

	// --- "dummy" Runner ---
	type dummyDeps struct {
		R any `hcl:"r"`
	}
	r.RegisterHandler("OnRunDummy", &registry.RegisteredHandler{
		NewInput: func() any { return new(schema.StepArgs) },
		NewDeps:  func() any { return new(dummyDeps) },
		Fn:       func(context.Context, any, any) (cty.Value, error) { return cty.NilVal, nil },
	})
	r.DefinitionRegistry["dummy"] = &schema.RunnerDefinition{
		Type:      "dummy",
		Lifecycle: &schema.Lifecycle{OnRun: "OnRunDummy"},
		Uses: []*schema.UsesDefinition{
			{LocalName: "r", AssetType: "destroy_spy_resource"},
		},
	}
}

// Test for: Resource `Destroy` handler is called once on cleanup.
func TestCoreExecution_ResourceDestroyOnCleanup(t *testing.T) {
	// --- Arrange ---
	hcl := `
		resource "destroy_spy_resource" "A" {}

		step "dummy" "B" {
			uses {
				r = resource.destroy_spy_resource.A
			}
		}
	`
	tempDir := t.TempDir()
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(hcl), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	var destroyCalls atomic.Int32
	appConfig := &app.AppConfig{GridPath: gridPath}
	mockModule := &mockDestroySpyModule{destroyCalls: &destroyCalls}
	testApp, _ := testutil.SetupAppTest(t, appConfig, mockModule)

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
