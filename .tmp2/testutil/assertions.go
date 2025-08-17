package testutil

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// AssertStepRan checks the log output within a HarnessResult to confirm that a
// specific step has completed. It abstracts the underlying node ID format, making
// tests more resilient to internal refactoring.
func AssertStepRan(t *testing.T, result *HarnessResult, runnerType, stepName string) {
	t.Helper()
	// This helper assumes a singular instance.
	AssertStepInstanceRan(t, result, runnerType, stepName, 0)
}

// AssertStepInstanceRan checks the log output within a HarnessResult to confirm
// that a specific instance of a step has finished successfully.
func AssertStepInstanceRan(t *testing.T, result *HarnessResult, runnerType, stepName string, index int) {
	t.Helper()
	// The log message for a placeholder's instance uses the placeholder's non-indexed ID.
	placeholderID := fmt.Sprintf("step.%s.%s", runnerType, stepName)
	instanceID := fmt.Sprintf("%s[%d]", placeholderID, index)

	// We check for the "Finished step instance" message which is logged upon success.
	expectedLog := fmt.Sprintf(`msg="✅ Finished step instance" step=%s`, instanceID)
	require.Contains(t, result.LogOutput, expectedLog,
		"expected log for successful run of instance '%s' was not found", instanceID)
}
