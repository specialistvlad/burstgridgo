package integration_tests

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
	"github.com/zclconf/go-cty/cty"
)

// --- Test-Specific Mocks for Cleanup Test ---

type eventRecord struct {
	Timestamp time.Time
}

// mockCleanupSpyModule only registers the Go handlers for the asset and runner.
type mockCleanupSpyModule struct {
	wg        *sync.WaitGroup
	events    *sync.Map
	stepTimes *sync.Map
}

// Register registers the "spy_resource" and "reporter" Go handlers.
func (m *mockCleanupSpyModule) Register(r *registry.Registry) {
	// --- "spy_resource" Asset: Go Handlers ---
	r.RegisterAssetHandler("CreateSpyResource", &registry.RegisteredAsset{
		NewInput: func() any { return new(schema.StepArgs) },
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
		R any `hcl:"r,optional"`
	}
	type reporterInput struct {
		Name string `hcl:"name"`
	}
	r.RegisterRunner("OnRunReporter", &registry.RegisteredRunner{
		NewInput: func() any { return new(reporterInput) },
		NewDeps:  func() any { return new(reporterDeps) },
		Fn: func(_ context.Context, _ any, inputRaw any) (cty.Value, error) {
			defer m.wg.Done()
			input := inputRaw.(*reporterInput)
			startTime := time.Now()
			time.Sleep(50 * time.Millisecond)
			endTime := time.Now()
			m.stepTimes.Store(input.Name, &app.ExecutionRecord{Start: startTime, End: endTime})
			return cty.NilVal, nil
		},
	})
}

// Test for: Resource is cleaned up efficiently.
func TestCoreExecution_ResourceEfficientCleanup(t *testing.T) {
	// --- Arrange ---
	tempDir := t.TempDir()

	// 1. Define and write HCL manifests for the asset and runner.
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
			lifecycle { on_run = "OnRunReporter" }
			uses "r" {
				asset_type = "spy_resource"
				# This 'uses' is optional in the manifest if the Go struct tag is ',optional'
			}
			input "name" {
				type = string
			}
		}
	`
	if err := os.MkdirAll(filepath.Join(tempDir, "modules", "spy_resource"), 0755); err != nil {
		t.Fatalf("failed to create asset module dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tempDir, "modules", "reporter"), 0755); err != nil {
		t.Fatalf("failed to create runner module dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "modules/spy_resource/manifest.hcl"), []byte(assetManifest), 0600); err != nil {
		t.Fatalf("failed to write asset manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "modules/reporter/manifest.hcl"), []byte(runnerManifest), 0600); err != nil {
		t.Fatalf("failed to write runner manifest: %v", err)
	}

	// 2. The user's grid file.
	gridHCL := `
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
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(gridHCL), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// 3. Configure the app for discovery.
	appConfig := &app.AppConfig{
		GridPath:    gridPath,
		ModulesPath: filepath.Join(tempDir, "modules"),
	}
	mockModule := &mockCleanupSpyModule{
		wg:        &wg,
		events:    new(sync.Map),
		stepTimes: new(sync.Map),
	}
	testApp, _ := app.SetupAppTest(t, appConfig, mockModule)

	// --- Act ---
	runErr := testApp.Run(context.Background(), appConfig)
	if runErr != nil {
		t.Fatalf("app.Run() returned an unexpected error: %v", runErr)
	}

	wg.Wait()
	time.Sleep(20 * time.Millisecond)

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
	stepB := stepBRecord.(*app.ExecutionRecord)

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
