package integration_tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vk/burstgridgo/internal/testutil"
)

// TestDagConcurrency_FanInSynchronizationTest validates that a fan-in node
// waits for all of its parallel dependencies to complete before starting.
func TestDagConcurrency_FanInSynchronizationTest(t *testing.T) {
	t.Parallel()
	// --- Arrange ---
	const stepCount = 4
	manifestHCL := `
        runner "sleeper" {
            lifecycle { on_run = "OnRunSleeper" }
            input "id" { type = string }
        }
    `
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
	files := map[string]string{
		"modules/sleeper/manifest.hcl": manifestHCL,
		"main.hcl":                     gridHCL,
	}

	completionChan := make(chan string, stepCount)
	mockModule := testutil.NewMockSleeperModule(completionChan, 100*time.Millisecond)

	// --- Act ---
	result := testutil.RunIntegrationTest(t, files, mockModule)
	require.NoError(t, result.Err, "test run failed unexpectedly")

	// --- Assert ---
	completed := make(map[string]struct{})
	for i := 0; i < stepCount; i++ {
		select {
		case id := <-completionChan:
			completed[id] = struct{}{}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for steps to complete. Completed %d of %d steps. Got: %v", len(completed), stepCount, completed)
		}
	}

	records := mockModule.ExecutionTimes
	require.Len(t, records, stepCount, "expected execution records for all 4 steps")

	latestPrereqEndTime := records["A"].End
	if records["B"].End.After(latestPrereqEndTime) {
		latestPrereqEndTime = records["B"].End
	}
	if records["C"].End.After(latestPrereqEndTime) {
		latestPrereqEndTime = records["C"].End
	}

	require.False(t, records["D"].Start.Before(latestPrereqEndTime), "fan-in step D started before all prerequisites were complete")
}
