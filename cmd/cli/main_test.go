package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun_PanicRecovery(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	// Define an HCL string with a syntax error that is guaranteed to cause a panic
	// during the loading phase inside app.NewApp().
	invalidHCL := `
		step "print" "A" {
			arguments {
		// Missing closing brace here
	`
	// Create a temporary directory and file to hold the invalid config.
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "main.hcl")
	err := os.WriteFile(filePath, []byte(invalidHCL), 0600)
	require.NoError(t, err, "failed to set up test file")

	// Prepare the arguments for the run function.
	args := []string{filePath}
	out := &bytes.Buffer{}

	// --- Act ---
	// Call the run function, which should recover the panic and return it as an error.
	runErr := run(out, args)

	// --- Assert ---
	require.Error(t, runErr, "run() should have returned an error after recovering from a panic")

	errStr := runErr.Error()
	require.True(t, strings.Contains(errStr, "application startup panicked"), "The error message should indicate that a panic was recovered.")
	require.True(t, strings.Contains(errStr, "failed to parse"), "The error message should contain the underlying reason for the panic.")
}
