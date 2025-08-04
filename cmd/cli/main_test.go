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

func TestRun_ShouldExit(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	// The "-h" (help) flag should cause cli.Parse to return `shouldExit=true`.
	args := []string{"-h"}
	out := &bytes.Buffer{}

	// --- Act ---
	// The run function should see `shouldExit=true` and return a nil error.
	err := run(out, args)

	// --- Assert ---
	require.NoError(t, err, "run() should return a nil error when shouldExit is true")
	require.Contains(t, out.String(), "Usage:", "Expected help text to be printed to the output buffer")
}

func TestRun_ParseError(t *testing.T) {
	t.Parallel()

	// --- Arrange ---
	// Providing an unknown flag will cause cli.Parse to return an error.
	args := []string{"--this-is-not-a-valid-flag"}
	out := &bytes.Buffer{}

	// --- Act ---
	// The run function should propagate the error from cli.Parse.
	err := run(out, args)

	// --- Assert ---
	require.Error(t, err, "run() should return an error when argument parsing fails")
	require.Contains(t, err.Error(), "flag provided but not defined: -this-is-not-a-valid-flag")
}
