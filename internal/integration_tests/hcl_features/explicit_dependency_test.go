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
	"github.com/zclconf/go-cty/cty"
)

// mockRecorderModule is a self-contained module for the explicit dependency test.
// It now only registers the Go handler, not the HCL definition.
type mockRecorderModule struct {
	wg             *sync.WaitGroup
	executionTimes map[string]time.Time
	mu             sync.Mutex
}

// Register registers the "recorder" runner's Go handler.
func (m *mockRecorderModule) Register(r *registry.Registry) {
	type recorderInput struct {
		Name string `hcl:"name"`
	}
	r.RegisterRunner("OnRunRecorder", &registry.RegisteredRunner{
		NewInput: func() any { return new(recorderInput) },
		NewDeps:  func() any { return new(struct{}) },
		Fn: func(ctx context.Context, deps any, input any) (cty.Value, error) {
			instanceName := input.(*recorderInput).Name
			m.mu.Lock()
			m.executionTimes[instanceName] = time.Now()
			m.mu.Unlock()
			m.wg.Done()
			return cty.NilVal, nil
		},
	})
	// The RunnerDefinition is no longer registered here. It will be discovered from the HCL file.
}

// Test for: explicit dependency
func TestHclFeatures_ExplicitDependency(t *testing.T) {
	// --- Arrange ---
	tempDir := t.TempDir()

	// 1. Define the HCL manifest for our test runner.
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
	// 2. Write the manifest to a temporary directory structure that the app can discover.
	moduleDir := filepath.Join(tempDir, "modules", "recorder")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("failed to create module directory: %v", err)
	}
	manifestPath := filepath.Join(moduleDir, "manifest.hcl")
	if err := os.WriteFile(manifestPath, []byte(manifestHCL), 0600); err != nil {
		t.Fatalf("failed to write manifest.hcl: %v", err)
	}

	// 3. The user's grid file, which uses the "recorder" runner.
	gridHCL := `
		step "recorder" "A" {
			arguments {
				name = "A"
			}
		}

		step "recorder" "B" {
			arguments {
				name = "B"
			}
			depends_on = ["recorder.A"]
		}
	`
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(gridHCL), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2) // We expect two steps to run.

	// 4. Configure the app to discover modules from our temporary directory.
	appConfig := &app.AppConfig{
		GridPath:    gridPath,
		ModulesPath: filepath.Join(tempDir, "modules"),
	}
	mockModule := &mockRecorderModule{
		wg:             &wg,
		executionTimes: make(map[string]time.Time),
	}
	testApp, _ := app.SetupAppTest(t, appConfig, mockModule)

	// --- Act ---
	runErr := testApp.Run(context.Background(), appConfig)
	if runErr != nil {
		t.Fatalf("app.Run() returned an unexpected error: %v", runErr)
	}

	wg.Wait() // Wait for both recorders to finish.

	// --- Assert ---
	timeA, okA := mockModule.executionTimes["A"]
	timeB, okB := mockModule.executionTimes["B"]

	if !okA || !okB {
		t.Fatalf("Expected both steps A and B to have recorded their execution times")
	}

	// Assert that the execution time of B is not before A.
	if timeB.Before(timeA) {
		t.Errorf("Step B executed before Step A, but depends_on should have forced B to wait. Time A: %v, Time B: %v", timeA, timeB)
	}
}
