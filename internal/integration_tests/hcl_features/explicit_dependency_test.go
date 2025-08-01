package integration_tests

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/testutil"
)

type mockRecorderModule struct {
	executionTimes map[string]time.Time
}

func (m *mockRecorderModule) Register(r *registry.Registry) {
	type recorderInput struct {
		Name string `bggo:"name"`
	}
	r.RegisterRunner("OnRunRecorder", &registry.RegisteredRunner{
		NewInput:  func() any { return new(recorderInput) },
		InputType: reflect.TypeOf(recorderInput{}),
		NewDeps:   func() any { return new(struct{}) },
		Fn: func(ctx context.Context, deps any, input any) (any, error) {
			instanceName := input.(*recorderInput).Name
			m.executionTimes[instanceName] = time.Now()
			time.Sleep(10 * time.Millisecond)
			return nil, nil
		},
	})
}

func TestHclFeatures_ExplicitDependency(t *testing.T) {
	t.Parallel()
	// --- Arrange ---
	manifestHCL := `
		runner "recorder" {
		  lifecycle {
		    on_run = "OnRunRecorder"
		  }
		  input "name" {
		    type = string
		  }
		}
	`
	gridHCL := `
		step "recorder" "A" {
			arguments { name = "A" }
		}

		step "recorder" "B" {
			arguments { name = "B" }
			depends_on = ["recorder.A"]
		}
	`
	files := map[string]string{
		"modules/recorder/manifest.hcl": manifestHCL,
		"main.hcl":                      gridHCL,
	}

	mockModule := &mockRecorderModule{
		executionTimes: make(map[string]time.Time),
	}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)

	// --- Assert ---
	require.NoError(t, result.Err, "test run failed unexpectedly")

	timeA, okA := mockModule.executionTimes["A"]
	timeB, okB := mockModule.executionTimes["B"]

	require.True(t, okA && okB, "Expected both steps A and B to have recorded their execution times")
	require.True(t, timeB.After(timeA), "Step B should have executed after Step A")
}
