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

// Test for: Independent parallel tracks execute concurrently.
func TestDagConcurrency_IndependentExecutionTrackingTest(t *testing.T) {
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

	// 2. Define the user's grid file with two independent tracks.
	gridHCL := `
		# Track 1
		step "sleeper" "track1_A" {
			arguments { id = "1A" }
		}
		step "sleeper" "track1_B" {
			arguments { id = "1B" }
			depends_on = ["sleeper.track1_A"]
		}

		# Track 2
		step "sleeper" "track2_A" {
			arguments { id = "2A" }
		}
		step "sleeper" "track2_B" {
			arguments { id = "2B" }
			depends_on = ["sleeper.track2_A"]
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
	track1A := records["1A"]
	track1B := records["1B"]
	track2A := records["2A"]

	// The critical assertion: Track 2 should start before Track 1 has fully finished.
	if track2A.Start.After(track1B.End) {
		t.Errorf("independent tracks did not run in parallel (Track 2 started after Track 1 finished)")
	}
	// Also validate that dependencies within a single track are still respected.
	if track1B.Start.Before(track1A.End) {
		t.Errorf("dependency violation in Track 1: step B started before A finished")
	}
}
