package integration_tests

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/testutil"
)

// mockFailerAndSpyModule is a test-specific module that registers two runners:
// one that is designed to fail, and a "spy" to verify it doesn't run.
type mockFailerAndSpyModule struct {
	wasSpyExecuted *atomic.Bool
	injectedError  error
}

func (m *mockFailerAndSpyModule) Register(r *registry.Registry) {
	// "failer" Runner: This one will return an error.
	r.RegisterRunner("OnRunFailer", &registry.RegisteredRunner{
		NewInput:  func() any { return new(struct{}) },
		InputType: reflect.TypeOf(struct{}{}),
		NewDeps:   func() any { return new(struct{}) },
		Fn: func(context.Context, any, any) (any, error) {
			return nil, m.injectedError
		},
	})

	// "spy" Runner: If this runs, the test has failed.
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

// TestErrorHandling_FailingStep_TriggersFailFast validates that if one step
// fails, its dependents are not executed.
func TestErrorHandling_FailingStep_TriggersFailFast(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	failerManifestHCL := `
		runner "failer" {
			lifecycle { on_run = "OnRunFailer" }
		}
	`
	spyManifestHCL := `
		runner "spy" {
			lifecycle { on_run = "OnRunSpy" }
		}
	`
	gridHCL := `
		step "failer" "A" {
			arguments {}
		}

		step "spy" "B" {
			arguments {}
			depends_on = ["failer.A"]
		}
	`
	files := map[string]string{
		"modules/failer/manifest.hcl": failerManifestHCL,
		"modules/spy/manifest.hcl":    spyManifestHCL,
		"main.hcl":                    gridHCL,
	}

	var wasSpyExecuted atomic.Bool
	expectedErr := errors.New("handler failed as expected")

	// The mock module is now defined at the package level.
	mockModule := &mockFailerAndSpyModule{
		wasSpyExecuted: &wasSpyExecuted,
		injectedError:  expectedErr,
	}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.Error(t, result.Err, "app.Run() should have returned an error, but it returned nil")
	require.ErrorIs(t, result.Err, expectedErr, "expected the error chain to contain our injected error")
	require.False(t, wasSpyExecuted.Load(), "fail-fast did not work: a step dependent on the failing step was executed")
}
