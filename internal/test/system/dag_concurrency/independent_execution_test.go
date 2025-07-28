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

// Test for: Independent parallel tracks execute concurrently.
func TestDagConcurrency_IndependentExecutionTrackingTest(t *testing.T) {
	// --- Arrange ---
	hcl := `
		// Track 1
		step "sleeper" "track1_A" {
			arguments { id = "1A" }
		}
		step "sleeper" "track1_B" {
			arguments { id = "1B" }
			depends_on = ["sleeper.track1_A"]
		}

		// Track 2
		step "sleeper" "track2_A" {
			arguments { id = "2A" }
		}
		step "sleeper" "track2_B" {
			arguments { id = "2B" }
			depends_on = ["sleeper.track2_A"]
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
		executionTimes: make(map[string]*testutil.ExecutionRecord), // Corrected type
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
	track1A := records["1A"]
	track1B := records["1B"]
	track2A := records["2A"]

	if track2A.Start.After(track1B.End) {
		t.Errorf("independent tracks did not run in parallel (Track 2 started after Track 1 finished)")
	}
	if track1B.Start.Before(track1A.End) {
		t.Errorf("dependency violation in Track 1: step B started before A finished")
	}
}
