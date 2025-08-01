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

// mockCreateOnceModule registers a resource that counts its creation calls.
type mockCreateOnceModule struct {
	createCalls *atomic.Int32
}

func (m *mockCreateOnceModule) Register(r *registry.Registry) {
	// Asset that counts how many times its CreateFn is called.
	r.RegisterAssetHandler("CreateCountingResource", &registry.RegisteredAsset{
		NewInput: func() any { return new(struct{}) },
		CreateFn: func(context.Context, any) (any, error) {
			m.createCalls.Add(1)
			return "dummy_resource_instance", nil
		},
	})
	r.RegisterAssetHandler("DestroyCountingResource", &registry.RegisteredAsset{
		DestroyFn: func(any) error { return nil },
	})

	// Runner that depends on the counting resource.
	type spyDeps struct {
		R any `bggo:"r"` // Updated to bggo tag
	}
	r.RegisterRunner("OnRunSpy", &registry.RegisteredRunner{
		NewInput:  func() any { return new(struct{}) },
		InputType: reflect.TypeOf(struct{}{}),
		NewDeps:   func() any { return new(spyDeps) },
		Fn:        func(context.Context, any, any) (any, error) { return nil, nil },
	})
}

// TestCoreExecution_ResourceCreateHandlerCalledOnce validates that a resource's
// Create handler is called only once, even if multiple steps depend on it.
func TestCoreExecution_ResourceCreateHandlerCalledOnce(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	assetManifestHCL := `
		asset "counting_resource" {
			lifecycle {
				create  = "CreateCountingResource"
				destroy = "DestroyCountingResource"
			}
		}
	`
	runnerManifestHCL := `
		runner "spy" {
			lifecycle { on_run = "OnRunSpy" }
			uses "r" {
				asset_type = "counting_resource"
			}
		}
	`
	// HCL defines one resource instance and two steps that depend on it.
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
	files := map[string]string{
		"modules/counting_resource/manifest.hcl": assetManifestHCL,
		"modules/spy/manifest.hcl":               runnerManifestHCL,
		"main.hcl":                               gridHCL,
	}

	var createCalls atomic.Int32
	mockModule := &mockCreateOnceModule{createCalls: &createCalls}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.NoError(t, result.Err, "app.Run() returned an unexpected error")

	finalCallCount := createCalls.Load()
	require.Equal(t, int32(1), finalCallCount, "expected resource Create handler to be called 1 time, but it was called %d times", finalCallCount)
}
