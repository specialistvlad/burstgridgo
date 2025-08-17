package integration_tests

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/specialistvlad/burstgridgo/internal/registry"
	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestErrorHandling_StepTimeout_FailsRun validates that a step is cancelled
// by the context if it runs for too long.
func TestErrorHandling_StepTimeout_FailsRun(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	manifestHCL := `
		runner "sleeper" {
			lifecycle {
				on_run = "OnRunSleeper"
			}
			input "duration" {
				type = string
			}
		}
	`
	gridHCL := `
		step "sleeper" "A" {
			arguments {
				duration = "1s" // This is longer than the context timeout.
			}
		}
	`
	files := map[string]string{
		"modules/sleeper/manifest.hcl": manifestHCL,
		"main.hcl":                     gridHCL,
	}

	type sleeperInput struct {
		Duration string `bggo:"duration"`
	}

	mockModule := &testutil.SimpleModule{
		RunnerName: "OnRunSleeper",
		Runner: &registry.RegisteredRunner{
			NewInput:  func() any { return new(sleeperInput) },
			InputType: reflect.TypeOf(sleeperInput{}),
			NewDeps:   func() any { return new(struct{}) },
			Fn: func(ctx context.Context, deps any, input any) (any, error) {
				in := input.(*sleeperInput)
				duration, err := time.ParseDuration(in.Duration)
				if err != nil {
					return nil, err
				}

				select {
				case <-time.After(duration):
					return nil, errors.New("step sleep was not canceled by context")
				case <-ctx.Done():
					return nil, ctx.Err() // This should return context.DeadlineExceeded
				}
			},
		},
	}

	// --- Act ---
	// Create a context with a very short timeout to trigger the failure condition.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result := testutil.RunIntegrationTestWithContext(ctx, t, files, mockModule)

	// --- Assert ---
	require.Error(t, result.Err, "app.Run() should have returned a timeout error, but it returned nil")
	require.ErrorIs(t, result.Err, context.DeadlineExceeded, "the error chain should contain context.DeadlineExceeded")
}
