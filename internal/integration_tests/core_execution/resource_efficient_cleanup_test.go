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

// --- Test-Specific Mocks for Cleanup Test ---

type eventRecord struct {
	Timestamp time.Time
}

type mockCleanupSpyModule struct {
	events         *sync.Map
	stepTimes      *sync.Map
	completionChan chan<- string
}

func (m *mockCleanupSpyModule) Register(r *registry.Registry) {
	// --- "spy_resource" Asset: Go Handlers ---
	r.RegisterAssetHandler("CreateSpyResource", &registry.RegisteredAsset{
		NewInput: func() any { return new(struct{}) },
		CreateFn: func(context.Context, any) (any, error) {
			m.events.Store("Create", &eventRecord{Timestamp: time.Now()})
			return "spy_instance", nil
		},
	})
	r.RegisterAssetHandler("DestroySpyResource", &registry.RegisteredAsset{
		DestroyFn: func(any) error {
			m.events.Store("Destroy", &eventRecord{Timestamp: time.Now()})
			return nil
		},
	})

	// --- "reporter" Runner: Go Handler ---
	type reporterDeps struct {
		R any `bggo:"r,optional"`
	}
	type reporterInput struct {
		Name string `bggo:"name"`
	}
	r.RegisterRunner("OnRunReporter", &registry.RegisteredRunner{
		NewInput:  func() any { return new(reporterInput) },
		InputType: reflect.TypeOf(reporterInput{}), // This line was missing
		NewDeps:   func() any { return new(reporterDeps) },
		Fn: func(_ context.Context, _ any, inputRaw any) (any, error) {
			input := inputRaw.(*reporterInput)
			startTime := time.Now()
			time.Sleep(50 * time.Millisecond)
			endTime := time.Now()
			m.stepTimes.Store(input.Name, &testutil.ExecutionRecord{Start: startTime, End: endTime})
			if m.completionChan != nil {
				m.completionChan <- input.Name
			}
			return nil, nil
		},
	})
}

// TestCoreExecution_ResourceEfficientCleanup validates that a resource is destroyed
// as soon as it's no longer needed by any downstream steps.
func TestCoreExecution_ResourceEfficientCleanup(t *testing.T) {
	t.Parallel()
	// --- Arrange ---
	const stepCount = 2
	assetManifest := `
		asset "spy_resource" {
			lifecycle {
				create = "CreateSpyResource"
				destroy = "DestroySpyResource"
			}
		}
	`
	runnerManifest := `
		runner "reporter" {
			lifecycle {
				on_run = "OnRunReporter"
			}
			uses "r" {
				asset_type = "spy_resource"
			}
			input "name" {
				type = string
			}
		}
	`
	gridHCL := `
		resource "spy_resource" "R" {}

		step "reporter" "A" {
			uses {
				r = resource.spy_resource.R
			}
			arguments {
				name = "A"
			}
		}

		step "reporter" "B" {
			depends_on = ["reporter.A"]
			arguments {
				name = "B"
			}
		}
	`
	files := map[string]string{
		"modules/spy_resource/manifest.hcl": assetManifest,
		"modules/reporter/manifest.hcl":     runnerManifest,
		"main.hcl":                          gridHCL,
	}

	completionChan := make(chan string, stepCount)
	mockModule := &mockCleanupSpyModule{
		events:         new(sync.Map),
		stepTimes:      new(sync.Map),
		completionChan: completionChan,
	}

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)
	require.NoError(t, result.Err, "test run failed unexpectedly")

	// --- Assert ---
	// Wait for steps to complete using the channel
	completed := make(map[string]struct{})
	for i := 0; i < stepCount; i++ {
		select {
		case id := <-completionChan:
			completed[id] = struct{}{}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for steps to complete. Completed %d of %d steps. Got: %v", len(completed), stepCount, completed)
		}
	}

	destroyEvent, ok := mockModule.events.Load("Destroy")
	require.True(t, ok, "Resource was never destroyed")
	destroyTime := destroyEvent.(*eventRecord).Timestamp

	stepBRecord, ok := mockModule.stepTimes.Load("B")
	require.True(t, ok, "Step B never recorded its execution time")
	stepB := stepBRecord.(*testutil.ExecutionRecord)

	require.True(t, destroyTime.Before(stepB.End), "Resource was not destroyed efficiently before step B finished")
}
