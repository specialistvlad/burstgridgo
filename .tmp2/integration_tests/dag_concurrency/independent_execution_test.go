package integration_tests

import (
	"testing"
	"time"

	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestDagConcurrency_IndependentExecutionTest validates that two independent
// dependency chains run concurrently.
func TestDagConcurrency_IndependentExecutionTest(t *testing.T) {
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
		# Track 1
		step "sleeper" "track1_A" {
			arguments {
				id = "1A"
			}
		}
		step "sleeper" "track1_B" {
			arguments {
				id = "1B"
			}
			depends_on = ["sleeper.track1_A"]
		}

		# Track 2
		step "sleeper" "track2_A" {
			arguments {
				id = "2A"
			}
		}
		step "sleeper" "track2_B" {
			arguments {
				id = "2B"
			}
			depends_on = ["sleeper.track2_A"]
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

	track1A := records["1A"]
	track1B := records["1B"]
	track2A := records["2A"]

	// Critical assertion: Track 2 should start before Track 1 has fully finished.
	require.True(t, track2A.Start.Before(track1B.End), "independent tracks did not run in parallel")

	// Also validate that dependencies within a single track are still respected.
	require.True(t, track1B.Start.After(track1A.End), "dependency violation in Track 1")
}
