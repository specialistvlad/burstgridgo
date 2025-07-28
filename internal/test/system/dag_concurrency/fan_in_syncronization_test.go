package system

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/testutil"
)

// Test for: Fan-in synchronization waits for all parallel nodes.
func TestDagConcurrency_FanInSynchronizationTest(t *testing.T) {
	// --- Arrange ---
	hcl := `
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
	tempDir := t.TempDir()
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(hcl), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(4)

	appConfig := &app.AppConfig{GridPath: gridPath, WorkerCount: 4}
	mockModule := &mockSleeperModule{
		wg:             &wg,
		executionTimes: make(map[string]*testutil.ExecutionRecord),
		sleepDuration:  100 * time.Millisecond,
	}
	testApp, _ := testutil.SetupAppTest(t, appConfig, mockModule)

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
