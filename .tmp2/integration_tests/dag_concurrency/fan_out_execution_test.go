package integration_tests

import (
	"testing"
	"time"

	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestDagConcurrency_FanOutExecutionTest validates that nodes in a fan-out
// structure run concurrently.
func TestDagConcurrency_FanOutExecutionTest(t *testing.T) {
	t.Parallel()
	// --- Arrange ---
	const stepCount = 4
	manifestHCL := `
        runner "sleeper" {
            lifecycle {
                on_run = "OnRunSleeper"
            }
            input "id" {
                type = string
            }
        }
    `
	gridHCL := `
        step "sleeper" "A" {
            arguments {
                id = "A"
            }
        }
        step "sleeper" "B" {
            arguments {
                id = "B"
            }
            depends_on = ["sleeper.A"]
        }
        step "sleeper" "C" {
            arguments {
                id = "C"
            }
            depends_on = ["sleeper.A"]
        }
        step "sleeper" "D" {
            arguments {
                id = "D"
            }
            depends_on = ["sleeper.A"]
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

	// Concurrency assertions
	records := mockModule.ExecutionTimes
	require.Len(t, records, stepCount, "expected execution records for all 4 steps")

	recordB := records["B"]
	recordC := records["C"]
	recordD := records["D"]

	// Assert that the time ranges of parallel steps B and C overlap.
	if recordB.Start.After(recordC.End) || recordC.Start.After(recordB.End) {
		t.Errorf("steps B and C did not run in parallel")
	}
	// Assert that the time ranges of parallel steps C and D overlap.
	if recordC.Start.After(recordD.End) || recordD.Start.After(recordC.End) {
		t.Errorf("steps C and D did not run in parallel")
	}
}
