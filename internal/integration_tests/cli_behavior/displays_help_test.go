package integration_tests

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vk/burstgridgo/internal/cli"
)

// Test for: displays help
func TestCLI_DisplaysHelp_WhenNoGridPathIsProvided(t *testing.T) {
	t.Parallel() // This test is safe to run in parallel with others.

	// --- Arrange ---
	// Create a buffer to capture the output from the CLI parser.
	// This lets us check what's "printed" to the console.
	outW := &bytes.Buffer{}

	// --- Act ---
	// Call the CLI parser with an empty slice of arguments,
	// simulating the user running the program with no commands.
	appConfig, shouldExit, err := cli.Parse([]string{}, outW)

	// --- Assert ---
	if err != nil {
		t.Fatalf("cli.Parse() returned an unexpected error: %v", err)
	}

	if !shouldExit {
		t.Fatal("cli.Parse() should have indicated an exit, but it did not")
	}

	// Verify that the help text was printed by checking for a known string.
	if !strings.Contains(outW.String(), "Usage:") {
		t.Errorf("expected output to contain 'Usage:', but got:\n%s", outW.String())
	}

	// If the program is exiting to show help, no config should be returned.
	if appConfig != nil {
		t.Errorf("expected a nil AppConfig when displaying help, but got a non-nil config")
	}
}
