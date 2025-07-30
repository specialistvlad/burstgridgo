package integration_tests

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vk/burstgridgo/internal/app"
)

// Test for: invalid hcl is rejected
func TestErrorHandling_InvalidHCL_IsRejected(t *testing.T) {
	// --- Arrange ---
	// Define an HCL string with a clear syntax error (a missing closing brace).
	invalidHCL := `
		step "print" "A" {
			arguments {
		// Missing closing brace here
	`
	tempDir := t.TempDir()
	gridPath := filepath.Join(tempDir, "main.hcl")
	if err := os.WriteFile(gridPath, []byte(invalidHCL), 0600); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}

	// For this test, we don't need any real modules since the failure should
	// happen during parsing, long before any handlers are invoked.
	appConfig := &app.AppConfig{GridPath: gridPath}
	testApp, _ := app.SetupAppTest(t, appConfig)

	// --- Act ---
	// Run the application. We expect an error during the config loading phase.
	runErr := testApp.Run(context.Background(), appConfig)

	// --- Assert ---
	if runErr == nil {
		t.Fatal("app.Run() should have returned an error for invalid HCL, but it returned nil")
	}

	// Check for keywords that indicate a parsing or decoding error, which
	// confirms the failure happened at the expected stage.
	errMsg := runErr.Error()
	if !strings.Contains(errMsg, "failed to parse") && !strings.Contains(errMsg, "failed to decode") {
		t.Errorf("expected error message to indicate an HCL parsing failure, but got: %s", errMsg)
	}
}
