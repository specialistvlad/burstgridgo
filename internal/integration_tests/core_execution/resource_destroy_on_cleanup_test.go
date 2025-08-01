package integration_tests

import (
	"context"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/testutil"
)

// mockDestroySpyModule is a test-specific module that registers a resource
// which counts how many times its Destroy handler is called.
type mockDestroySpyModule struct {
	destroyCalls *atomic.Int32
}

func (m *mockDestroySpyModule) Register(r *registry.Registry) {
	// Asset with a counting Destroy handler.
	r.RegisterAssetHandler("CreateDestroySpyResource", &registry.RegisteredAsset{
		NewInput: func() any { return new(struct{}) },
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

	// Runner that uses the resource.
	type dummyDeps struct {
		R any `bggo:"r"`
	}
	r.RegisterRunner("OnRunDummy", &registry.RegisteredRunner{
		NewInput:  func() any { return new(struct{}) },
		InputType: reflect.TypeOf(struct{}{}),
		NewDeps:   func() any { return new(dummyDeps) },
		Fn:        func(context.Context, any, any) (any, error) { return nil, nil },
	})
}

// TestCoreExecution_ResourceDestroyOnCleanup validates that a resource's
// Destroy handler is called once during application cleanup.
func TestCoreExecution_ResourceDestroyOnCleanup(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	assetManifestHCL := `
		asset "destroy_spy_resource" {
			lifecycle {
				create = "CreateDestroySpyResource"
				destroy = "DestroyDestroySpyResource"
			}
		}
	`
	runnerManifestHCL := `
		runner "dummy" {
			lifecycle { on_run = "OnRunDummy" }
			uses "r" {
				asset_type = "destroy_spy_resource"
			}
		}
	`
	gridHCL := `
		resource "destroy_spy_resource" "A" {}

		step "dummy" "B" {
			uses {
				r = resource.destroy_spy_resource.A
			}
		}
	`
	files := map[string]string{
		"modules/destroy_spy_resource/manifest.hcl": assetManifestHCL,
		"modules/dummy/manifest.hcl":                runnerManifestHCL,
		"main.hcl":                                  gridHCL,
	}

	var destroyCalls atomic.Int32
	mockModule := &mockDestroySpyModule{destroyCalls: &destroyCalls}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.NoError(t, result.Err, "app.Run() returned an unexpected error")

	finalCallCount := destroyCalls.Load()
	require.Equal(t, int32(1), finalCallCount, "expected resource Destroy handler to be called 1 time, but it was called %d times", finalCallCount)
}
