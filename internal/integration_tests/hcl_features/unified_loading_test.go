package integration_tests

import (
	"context"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/registry"
	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
)

type mockUnifiedModule struct {
	wasExecuted *atomic.Bool
}

func (m *mockUnifiedModule) Register(r *registry.Registry) {
	type runnerInput struct {
		Message string `bggo:"message"`
	}
	r.RegisterRunner("OnRunUnified", &registry.RegisteredRunner{
		NewInput:  func() any { return new(runnerInput) },
		InputType: reflect.TypeOf(runnerInput{}),
		NewDeps:   func() any { return new(struct{}) },
		Fn: func(ctx context.Context, deps any, input any) (any, error) {
			m.wasExecuted.Store(true)
			return nil, nil
		},
	})
}

// TestHclFeatures_UnifiedLoading validates that a runner definition and a step
// instance can be loaded from the same file.
func TestHclFeatures_UnifiedLoading(t *testing.T) {
	t.Parallel()
	// --- Arrange ---
	unifiedHCL := `
		runner "unified_runner" {
			lifecycle {
				on_run = "OnRunUnified"
			}
			input "message" {
				type = string
			}
		}

		step "unified_runner" "A" {
			arguments {
				message = "This was loaded from the same file!"
			}
		}
	`
	files := map[string]string{
		"unified.hcl": unifiedHCL,
	}

	var wasExecuted atomic.Bool
	mockModule := &mockUnifiedModule{wasExecuted: &wasExecuted}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.NoError(t, result.Err)
	require.True(t, wasExecuted.Load(), "the step defined in the unified file was not executed")
	require.True(t, strings.Contains(result.LogOutput, "step=step.unified_runner.A"))
}
