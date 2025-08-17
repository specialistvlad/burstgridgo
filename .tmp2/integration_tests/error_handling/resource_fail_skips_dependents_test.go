package integration_tests

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/registry"
	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
)

// mockResourceFailModule holds the handlers for a failing asset and a spy runner.
type mockResourceFailModule struct {
	wasSpyExecuted *atomic.Bool
	injectedError  error
}

func (m *mockResourceFailModule) Register(r *registry.Registry) {
	// --- "failing_resource" Asset: Go Handlers ---
	r.RegisterAssetHandler("CreateFailingResource", &registry.RegisteredAsset{
		NewInput: func() any { return new(struct{}) },
		CreateFn: func(context.Context, any) (any, error) {
			return nil, m.injectedError
		},
	})
	r.RegisterAssetHandler("DestroyFailingResource", &registry.RegisteredAsset{
		DestroyFn: func(any) error { return nil },
	})

	// --- "spy" Runner: Go Handler ---
	r.RegisterRunner("OnRunSpy", &registry.RegisteredRunner{
		NewInput:  func() any { return new(struct{}) },
		InputType: reflect.TypeOf(struct{}{}),
		NewDeps:   func() any { return new(struct{}) },
		Fn: func(context.Context, any, any) (any, error) {
			m.wasSpyExecuted.Store(true)
			return nil, nil
		},
	})
}

// TestErrorHandling_ResourceFailure_SkipsDependents validates that a step
// dependent on a failing resource is not executed.
func TestErrorHandling_ResourceFailure_SkipsDependents(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
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
			lifecycle {
				on_run = "OnRunSpy"
			}
			uses "r" {
				asset_type = "failing_resource"
			}
		}
	`
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
	files := map[string]string{
		"modules/failing_resource/manifest.hcl": assetManifest,
		"modules/spy/manifest.hcl":              runnerManifest,
		"main.hcl":                              gridHCL,
	}

	var wasSpyExecuted atomic.Bool
	expectedErr := errors.New("resource creation failed as expected")

	mockModule := &mockResourceFailModule{
		wasSpyExecuted: &wasSpyExecuted,
		injectedError:  expectedErr,
	}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.Error(t, result.Err, "app.Run() should have returned an error, but it returned nil")
	require.ErrorIs(t, result.Err, expectedErr, "expected the error chain to contain our injected error")
	require.False(t, wasSpyExecuted.Load(), "fail-fast did not work: a step dependent on the failing resource was executed")
}
