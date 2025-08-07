package testutil

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// AssertStepRan checks the log output within a HarnessResult to confirm that a
// specific step has completed. It abstracts the underlying node ID format, making
// tests more resilient to internal refactoring.
func AssertStepRan(t *testing.T, result *HarnessResult, runnerType, stepName string) {
	t.Helper()

	// This completes the Phase 2 refactoring by aligning the test suite with
	// the new internal reality.
	expectedLogSubstring := fmt.Sprintf("step=step.%s.%s[0]", runnerType, stepName)

	require.True(t,
		strings.Contains(result.LogOutput, expectedLogSubstring),
		"expected log output for step '%s.%s[0]' was not found in logs", runnerType, stepName,
	)
}
