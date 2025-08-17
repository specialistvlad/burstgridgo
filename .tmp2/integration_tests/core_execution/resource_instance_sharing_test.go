package integration_tests

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/specialistvlad/burstgridgo/internal/registry"
	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
)

// sharableResource is the simple object we will be sharing.
type sharableResource struct {
	ID int
}

// mockInstanceSharingModule uses the new pure-Go contract.
type mockInstanceSharingModule struct {
	capturedPointers map[string]uintptr
	mu               sync.Mutex
	completionChan   chan<- string
}

func (m *mockInstanceSharingModule) Register(r *registry.Registry) {
	// --- "sharable_resource" Asset: Go Handlers ---
	r.RegisterAssetHandler("CreateSharableResource", &registry.RegisteredAsset{
		NewInput: func() any { return new(struct{}) },
		CreateFn: func(context.Context, any) (any, error) {
			return &sharableResource{ID: 42}, nil
		},
	})
	r.RegisterAssetHandler("DestroySharableResource", &registry.RegisteredAsset{
		DestroyFn: func(any) error { return nil },
	})

	// --- "spy" Runner: Go Handler ---
	type spyDeps struct {
		Resource *sharableResource `bggo:"r"`
	}
	type spyInput struct {
		Name string `bggo:"name"`
	}
	r.RegisterRunner("OnRunSpy", &registry.RegisteredRunner{
		NewInput:  func() any { return new(spyInput) },
		InputType: reflect.TypeOf(spyInput{}), // This line was missing
		NewDeps:   func() any { return new(spyDeps) },
		Fn: func(_ context.Context, depsRaw any, inputRaw any) (any, error) {
			deps := depsRaw.(*spyDeps)
			input := inputRaw.(*spyInput)
			m.mu.Lock()
			m.capturedPointers[input.Name] = reflect.ValueOf(deps.Resource).Pointer()
			m.mu.Unlock()
			if m.completionChan != nil {
				m.completionChan <- input.Name
			}
			return nil, nil
		},
	})
}

// TestCoreExecution_ResourceInstanceSharing validates that multiple steps
// depending on the same resource receive the exact same Go object instance.
func TestCoreExecution_ResourceInstanceSharing(t *testing.T) {
	t.Parallel()
	// --- Arrange ---
	const stepCount = 2
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
			lifecycle {
				on_run = "OnRunSpy"
			}
			uses "r" {
				asset_type = "sharable_resource"
			}
			input "name" {
				type = string
			}
		}
	`
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
	files := map[string]string{
		"modules/sharable_resource/manifest.hcl": assetManifestHCL,
		"modules/spy/manifest.hcl":               runnerManifestHCL,
		"main.hcl":                               gridHCL,
	}

	completionChan := make(chan string, stepCount)
	mockModule := &mockInstanceSharingModule{
		capturedPointers: make(map[string]uintptr),
		completionChan:   completionChan,
	}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)
	require.NoError(t, result.Err, "test run failed unexpectedly")

	// --- Assert ---
	completed := make(map[string]struct{})
	for i := 0; i < stepCount; i++ {
		select {
		case id := <-completionChan:
			completed[id] = struct{}{}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for steps to complete. Completed %d of %d steps. Got: %v", len(completed), stepCount, completed)
		}
	}

	pointers := mockModule.capturedPointers
	require.Len(t, pointers, 2, "expected 2 captured pointers")

	ptrB, okB := pointers["B"]
	ptrC, okC := pointers["C"]
	require.True(t, okB && okC, "expected to capture pointers for both steps 'B' and 'C'")

	require.Equal(t, ptrB, ptrC, "resource instance was not shared correctly")
}
