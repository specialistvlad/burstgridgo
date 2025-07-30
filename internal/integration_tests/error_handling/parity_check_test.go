package integration_tests

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/registry"
)

// mockParityCheckModule now only registers the Go handler implementation.
type mockParityCheckModule struct{}

// Register registers a runner whose Go implementation will be out of sync with its manifest.
func (m *mockParityCheckModule) Register(r *registry.Registry) {
	// The Go implementation has a field named 'go_only_field'.
	type runnerInput struct {
		GoOnlyField string `hcl:"go_only_field"`
	}
	r.RegisterRunner("OnRunMismatched", &registry.RegisteredRunner{
		NewInput: func() any { return new(runnerInput) },
		NewDeps:  func() any { return new(struct{}) },
		Fn:       func() {},
	})
}

// Test for: App fails to start if a runner's manifest and Go struct are out of sync.
func TestStartupValidation_ManifestImplementationMismatch_Fails(t *testing.T) {
	// --- Arrange ---
	tempDir := t.TempDir()
	appConfig := &app.AppConfig{
		ModulesPath: filepath.Join(tempDir, "modules"),
		// GridPath is not needed as the app should fail before execution.
	}
	mockModule := &mockParityCheckModule{}

	// 1. Define an HCL manifest that is intentionally mismatched with the Go implementation.
	// It's missing 'go_only_field' and includes 'hcl_only_field'.
	mismatchedManifest := `
		runner "mismatched_runner" {
			lifecycle { on_run = "OnRunMismatched" }
			input "hcl_only_field" {
				type = string
			}
		}
	`
	// 2. Write the mismatched manifest to a temporary directory for discovery.
	moduleDir := filepath.Join(appConfig.ModulesPath, "mismatched_runner")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("failed to create module directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "manifest.hcl"), []byte(mismatchedManifest), 0600); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// We expect app.New to panic, so we use a recover block to assert on it.
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("app.New() should have panicked due to manifest/Go struct mismatch, but it did not")
		}

		err, ok := r.(error)
		if !ok {
			t.Fatalf("panic was not an error: %v", r)
		}

		// Assert on the specific error messages from the registry validation.
		errMsg := err.Error()
		if !strings.Contains(errMsg, "Go struct has field 'go_only_field' not declared in manifest") {
			t.Errorf("expected error to contain message about 'go_only_field', but it did not. Got: %s", errMsg)
		}
		if !strings.Contains(errMsg, "manifest declares input 'hcl_only_field' not found in Go struct") {
			t.Errorf("expected error to contain message about 'hcl_only_field', but it did not. Got: %s", errMsg)
		}
	}()

	// --- Act ---
	// This call should panic when the registry validation runs.
	app.NewApp(&bytes.Buffer{}, appConfig, mockModule)
}
