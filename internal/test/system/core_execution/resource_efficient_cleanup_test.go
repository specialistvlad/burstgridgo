package system

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/schema"
	"github.com/vk/burstgridgo/internal/testutil"
	"github.com/zclconf/go-cty/cty"
)

// --- Test-Specific Mocks for Cleanup Test ---

type eventRecord struct {
	Timestamp time.Time
}

// mockCleanupSpyModule is a self-contained module for the efficient cleanup test.
type mockCleanupSpyModule struct {
	wg        *sync.WaitGroup
	events    *sync.Map // Stores "Create" and "Destroy" times for the resource.
	stepTimes *sync.Map // Stores start/end times for runner steps.
}

// Register registers the "spy_resource" and the "reporter" runner.
func (m *mockCleanupSpyModule) Register(r *registry.Registry) {
	// --- "spy_resource" Asset ---
	r.RegisterAssetHandler("CreateSpyResource", &registry.RegisteredAssetHandler{
		NewInput: func() any { return new(schema.StepArgs) },
		CreateFn: func(context.Context, any) (any, error) {
			m.events.Store("Create", &eventRecord{Timestamp: time.Now()})
			return "spy_instance", nil
		},
	})
	r.RegisterAssetHandler("DestroySpyResource", &registry.RegisteredAssetHandler{
		DestroyFn: func(any) error {
			m.events.Store("Destroy", &eventRecord{Timestamp: time.Now()})
			return nil
		},
	})
	r.AssetDefinitionRegistry["spy_resource"] = &schema.AssetDefinition{
		Type: "spy_resource",
		Lifecycle: &schema.AssetLifecycle{
			Create:  "CreateSpyResource",
			Destroy: "DestroySpyResource",
		},
	}

	// --- "reporter" Runner ---
	type reporterDeps struct {
		R any `hcl:"r,optional"` // Optional so step B doesn't need it.
	}
	type reporterInput struct {
		Name string `hcl:"name"`
	}
	r.RegisterHandler("OnRunReporter", &registry.RegisteredHandler{
		NewInput: func() any { return new(reporterInput) },
		NewDeps:  func() any { return new(reporterDeps) },
		Fn: func(_ context.Context, _ any, inputRaw any) (cty.Value, error) {
			defer m.wg.Done()
			input := inputRaw.(*reporterInput)
			startTime := time.Now()
			// Simulate a small amount of work.
			time.Sleep(50 * time.Millisecond)
			endTime := time.Now()
			m.stepTimes.Store(input.Name, &testutil.ExecutionRecord{Start: startTime, End: endTime})
			return cty.NilVal, nil
		},
	})
	r.DefinitionRegistry["reporter"] = &schema.RunnerDefinition{
		Type:      "reporter",
		Lifecycle: &schema.Lifecycle{OnRun: "OnRunReporter"},
		Uses:      []*schema.UsesDefinition{{LocalName: "r", AssetType: "spy_resource"}},
		Inputs:    []*schema.InputDefinition{{Name: "name"}},
	}
}

// Test for: Resource is cleaned up efficiently.
func TestCoreExecution_ResourceEfficientCleanup(t *testing.T) {
	// --- Arrange ---
	hcl := `
		resource "spy_resource" "R" {}

		step "reporter" "A" {
			uses { r = resource.spy_resource.R }
			arguments { name = "A" }
		}

		step "reporter" "B" {
			depends_on = ["reporter.A"]
			arguments { name = "B" }
		}
	`
	tempDir := t.TempDir()
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(hcl), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	appConfig := &app.AppConfig{GridPath: gridPath}
	mockModule := &mockCleanupSpyModule{
		wg:        &wg,
		events:    new(sync.Map),
		stepTimes: new(sync.Map),
	}
	testApp, _ := testutil.SetupAppTest(t, appConfig, mockModule)

	// --- Act ---
	runErr := testApp.Run(context.Background(), appConfig)
	if runErr != nil {
		t.Fatalf("app.Run() returned an unexpected error: %v", runErr)
	}

	wg.Wait()
	time.Sleep(20 * time.Millisecond) // A small extra wait to ensure the async destroy call has time to fire.

	// --- Assert ---
	destroyEvent, ok := mockModule.events.Load("Destroy")
	if !ok {
		t.Fatal("Resource was never destroyed")
	}
	destroyTime := destroyEvent.(*eventRecord).Timestamp

	stepBRecord, ok := mockModule.stepTimes.Load("B")
	if !ok {
		t.Fatal("Step B never recorded its execution time")
	}
	stepB := stepBRecord.(*testutil.ExecutionRecord)

	// FLIPPED ASSERTION: With the bug fixed, the destroy time should now
	// be *before* step B finishes.
	if !destroyTime.Before(stepB.End) {
		t.Errorf("BUG REMAINS: Resource was not destroyed efficiently.")
		t.Logf("  Step B End Time:  %v", stepB.End.UnixNano())
		t.Logf("  Destroy Time:     %v", destroyTime.UnixNano())
	} else {
		t.Logf("FIX CONFIRMED: Resource was destroyed efficiently before step B finished.")
		t.Logf("  Destroy Time:     %v", destroyTime.UnixNano())
		t.Logf("  Step B End Time:  %v", stepB.End.UnixNano())
	}
}
