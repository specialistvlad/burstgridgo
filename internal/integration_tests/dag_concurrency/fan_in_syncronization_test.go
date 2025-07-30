package integration_tests

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/vk/burstgridgo/internal/app"
)

// Test for: Fan-in synchronization waits for all parallel nodes.
func TestDagConcurrency_FanInSynchronizationTest(t *testing.T) {
	// --- Arrange ---
	tempDir := t.TempDir()

	// 1. Define and write the HCL manifest for the "sleeper" runner.
	manifestHCL := `
		runner "sleeper" {
			lifecycle { on_run = "OnRunSleeper" }
			input "id" {
				type = string
			}
		}
	`
	moduleDir := filepath.Join(tempDir, "modules", "sleeper")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("failed to create module directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "manifest.hcl"), []byte(manifestHCL), 0600); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// 2. Define the user's grid file with a fan-in dependency structure.
	gridHCL := `
		step "sleeper" "A" {
			arguments { id = "A" }
		}
		step "sleeper" "B" {
			arguments { id = "B" }
		}
		step "sleeper" "C" {
			arguments { id = "C" }
		}
		step "sleeper" "D" {
			arguments { id = "D" }
			depends_on = ["sleeper.A", "sleeper.B", "sleeper.C"]
		}
	`
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(gridHCL), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(4)

	// 3. Configure the app for module discovery.
	appConfig := &app.AppConfig{
		GridPath:    gridPath,
		ModulesPath: filepath.Join(tempDir, "modules"),
		WorkerCount: 4,
	}
	mockModule := &mockSleeperModule{
		wg:             &wg,
		executionTimes: make(map[string]*app.ExecutionRecord),
		sleepDuration:  100 * time.Millisecond,
	}
	testApp, _ := app.SetupAppTest(t, appConfig, mockModule)

	// --- Act ---
	runErr := testApp.Run(context.Background(), appConfig)
	if runErr != nil {
		t.Fatalf("app.Run() returned an unexpected error: %v", runErr)
	}

	wg.Wait()

	// --- Assert ---
	records := mockModule.executionTimes
	latestPrereqEndTime := records["A"].End
	if records["B"].End.After(latestPrereqEndTime) {
		latestPrereqEndTime = records["B"].End
	}
	if records["C"].End.After(latestPrereqEndTime) {
		latestPrereqEndTime = records["C"].End
	}

	if records["D"].Start.Before(latestPrereqEndTime) {
		t.Errorf("fan-in synchronization failed: step D started before all prerequisites were complete")
	}
}
