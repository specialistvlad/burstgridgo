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

// mockRecorderModule is a self-contained module for the explicit dependency test.
type mockRecorderModule struct {
	wg             *sync.WaitGroup
	executionTimes map[string]time.Time
	mu             sync.Mutex
}

// Register registers the "recorder" runner.
func (m *mockRecorderModule) Register(r *registry.Registry) {
	type recorderInput struct {
		Name string `hcl:"name"`
	}
	r.RegisterHandler("OnRunRecorder", &registry.RegisteredHandler{
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
	r.DefinitionRegistry["recorder"] = &schema.RunnerDefinition{
		Type:      "recorder",
		Lifecycle: &schema.Lifecycle{OnRun: "OnRunRecorder"},
		Inputs:    []*schema.InputDefinition{{Name: "name"}},
	}
}

// Test for: explicit dependency
func TestHclFeatures_ExplicitDependency(t *testing.T) {
	// --- Arrange ---
	// The HCL grid defines two steps. "B" explicitly depends on "A",
	// forcing it to run only after "A" has completed.
	hcl := `
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
	tempDir := t.TempDir()
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(hcl), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2) // We expect two steps to run.

	appConfig := &app.AppConfig{GridPath: gridPath}
	mockModule := &mockRecorderModule{
		wg:             &wg,
		executionTimes: make(map[string]time.Time),
	}
	testApp, _ := testutil.SetupAppTest(t, appConfig, mockModule)

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
