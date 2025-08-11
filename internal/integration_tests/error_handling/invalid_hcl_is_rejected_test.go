package integration_tests

import (
	"strings"
	"testing"

	"github.com/specialistvlad/burstgridgo/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestErrorHandling_InvalidHCL_IsRejected validates that the application panics
// on startup if HCL configuration is syntactically invalid.
func TestErrorHandling_InvalidHCL_IsRejected(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	// Define an HCL string with a clear syntax error (a missing closing brace).
	invalidHCL := `
		step "print" "A" {
			arguments {
		// Missing closing brace here
	`
	files := map[string]string{
		"main.hcl": invalidHCL,
	}

	// --- Act ---
	// The test harness will catch the panic during app.NewApp and return it as an error.
	// No modules are needed since the failure happens before they are used.
	result := testutil.RunIntegrationTest(t, files)

	// --- Assert ---
	require.Error(t, result.Err, "The app should have panicked on invalid HCL, but it did not.")

	// Check the error message to ensure it's the one we expect.
	errStr := result.Err.Error()
	isExpectedError := strings.Contains(errStr, "application startup panicked") &&
		(strings.Contains(errStr, "failed to parse") || strings.Contains(errStr, "failed to decode"))

	require.True(t, isExpectedError, "Expected panic message to indicate a parsing failure, but got: %v", result.Err)
}
