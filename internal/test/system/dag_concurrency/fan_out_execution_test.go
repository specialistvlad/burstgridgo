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

// Test for: Fan-out execution runs nodes in parallel.
func TestDagConcurrency_FanOutExecutionTest(t *testing.T) {
	// --- Arrange ---
	hcl := `
		step "sleeper" "A" {
			arguments { id = "A" }
		}
		step "sleeper" "B" {
			arguments { id = "B" }
			depends_on = ["sleeper.A"]
		}
		step "sleeper" "C" {
			arguments { id = "C" }
			depends_on = ["sleeper.A"]
		}
		step "sleeper" "D" {
			arguments { id = "D" }
			depends_on = ["sleeper.A"]
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
	recordB := records["B"]
	recordC := records["C"]
	recordD := records["D"]

	if recordB.Start.After(recordC.End) || recordC.Start.After(recordB.End) {
		t.Errorf("steps B and C did not run in parallel")
	}
	if recordC.Start.After(recordD.End) || recordD.Start.After(recordC.End) {
		t.Errorf("steps C and D did not run in parallel")
	}
}
